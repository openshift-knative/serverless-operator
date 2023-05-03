package features

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/knativeservice"
)

func VerifyEncryptedTrafficToActivatorToApp(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	// Just a small sleep to let the logs be written.
	time.Sleep(10 * time.Second)

	f := feature.NewFeature()

	f.Stable("path to activator to app").
		Must("has encrypted traffic", func(ctx context.Context, t feature.T) {
			env := environment.FromContext(ctx)

			var ksvcName string
			for _, ref := range refs {
				if ref.GroupVersionKind().GroupVersion() == knativeservice.GVR().GroupVersion() {
					ksvcName = ref.Name
				}
			}

			ksvc, err := dynamicclient.Get(ctx).Resource(knativeservice.GVR()).Namespace(env.Namespace()).
				Get(ctx, ksvcName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Unable to get ksvc %s: %v", ksvcName, err)
			}

			address, _, _ := unstructured.NestedString(ksvc.Object, "status", "address", "url")
			privateURL, err := url.Parse(address)
			if err != nil {
				t.Fatalf("Unable to parse URL %s: %v", address, err)
			}

			// source -> activator
			// When running within Mesh a mesh-specific VirtualService is used which
			// gets istio-ingressgateway out of the path.
			err = VerifyPodLogsEncryptedRequestToHost(ctx,
				"knative-serving",
				metav1.ListOptions{LabelSelector: "app=activator"},
				&corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
				func(m map[string]interface{}) bool {
					return getMapValueAsString(m, "path") == "/" &&
						getMapValueAsString(m, "authority") == privateURL.Host
				})
			if err != nil {
				t.Fatal(err)
			}

			// activator -> application
			err = VerifyPodLogsEncryptedRequestToHost(ctx,
				env.Namespace(),
				metav1.ListOptions{LabelSelector: "serving.knative.dev/service=" + ksvcName},
				&corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
				func(m map[string]interface{}) bool {
					return getMapValueAsString(m, "path") == "/" &&
						strings.HasPrefix(getMapValueAsString(m, "upstream_cluster"), "inbound|80")
				})
			if err != nil {
				t.Fatal(err)
			}

		})

	return f
}

func VerifyPodLogsEncryptedRequestToHost(ctx context.Context, podNamespace string, podSelector metav1.ListOptions, podLogOptions *corev1.PodLogOptions, jsonLogFilter func(map[string]interface{}) bool) error {
	encrypted, unencrypted, err := countEncryptedRequestsToHost(ctx, podNamespace, podSelector, podLogOptions, jsonLogFilter)
	if err != nil {
		return err
	}
	if encrypted == 0 && unencrypted == 0 {
		return fmt.Errorf("no log lines matching filter")
	}
	if unencrypted != 0 {
		return fmt.Errorf("unencrypted request found in %v logs", podSelector)
	}
	return nil
}

func VerifyNoPodLogsEncryptedRequestToHost(ctx context.Context, podNamespace string, listOptions metav1.ListOptions, podLogOptions *corev1.PodLogOptions, jsonLogFilter func(map[string]interface{}) bool) error {
	encrypted, unencrypted, err := countEncryptedRequestsToHost(ctx, podNamespace, listOptions, podLogOptions, jsonLogFilter)
	if err != nil {
		return err
	}
	if encrypted == 0 && unencrypted == 0 {
		return fmt.Errorf("no log lines matching filter")
	}
	if encrypted != 0 {
		return fmt.Errorf("an encrypted request found in %v logs", listOptions)
	}
	return nil
}

func countEncryptedRequestsToHost(ctx context.Context,
	podNamespace string,
	podSelector metav1.ListOptions,
	podLogOptions *corev1.PodLogOptions,
	jsonRequestLogFilter func(map[string]interface{}) bool) (encrypted int, unencrypted int, err error) {

	podList, err := kubeclient.Get(ctx).CoreV1().Pods(podNamespace).List(context.Background(), podSelector)
	if err != nil {
		return 0, 0, fmt.Errorf("error listing pods in %s: %w", podNamespace, err)
	}
	if len(podList.Items) == 0 {
		return 0, 0, fmt.Errorf("no %v pods found in %s", podSelector, podNamespace)
	}

	for _, pod := range podList.Items {
		podName := pod.Name
		if err = ForEachLine(ctx, podNamespace, podName, podLogOptions, func(line string) error {
			var ret map[string]interface{}
			if err := json.Unmarshal([]byte(line), &ret); err == nil {
				if jsonRequestLogFilter(ret) {
					logging.FromContext(ctx).Infof("%s: %s", podName, line)
					downstreamTLSCipher := getMapValueAsString(ret, "downstream_tls_cipher")
					// This is a bit arbitrary, but we just want to match something that is surely encrypted,
					// so we just match what is used at the time of writing...
					if downstreamTLSCipher != "ECDHE-RSA-AES256-GCM-SHA384" /* TLS 1.2 */ &&
						downstreamTLSCipher != "TLS_AES_256_GCM_SHA384" /* TLS 1.3 */ {
						logging.FromContext(ctx).Errorf("%s request unexpected downstream_tls_cipher %q",
							podName,
							downstreamTLSCipher)
						unencrypted++
					} else {
						encrypted++
					}
				}
			}
			return nil
		}); err != nil {
			return 0, 0, fmt.Errorf("error reading logs from %s: %w", podName, err)
		}
	}

	return
}

func getMapValueAsString(m map[string]interface{}, key string) string {
	valueInterface, ok := m[key]
	if ok {
		valueString, ok := valueInterface.(string)
		if ok {
			return valueString
		}
	}
	return ""
}

// ForEachLine calls onLineFunc on each line in pod logs.
func ForEachLine(ctx context.Context, namespace string, podName string, opts *corev1.PodLogOptions, onLineFunc func(string) error) error {
	stream, err := kubeclient.Get(ctx).CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(context.Background())
	if err != nil {
		return err
	}

	defer stream.Close()

	r := bufio.NewReader(stream)
	for {
		s, err := r.ReadBytes('\n')
		if len(s) > 0 {
			line := string(s)
			if err = onLineFunc(line); err != nil {
				return err
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
