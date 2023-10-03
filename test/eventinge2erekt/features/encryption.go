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

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/resources/knativeservice"
)

// LogFilter defines which logs should be checked.
type LogFilter struct {
	PodNamespace  string
	PodSelector   metav1.ListOptions
	PodLogOptions *corev1.PodLogOptions
	JSONLogFilter func(map[string]interface{}) bool
}

func VerifyEncryptedTrafficToActivatorToApp(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("path to activator to app").
		Must("has encrypted traffic to activator", VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func VerifyEncryptedTrafficToActivator(refs []corev1.ObjectReference, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		_, privateURL, err := getKsvcNameAndURL(ctx, refs)
		if err != nil {
			t.Fatalf("Unable to get Knative Service URL: %v", err)
		}

		// source -> activator
		// When running within Mesh a mesh-specific VirtualService is used which
		// gets istio-ingressgateway out of the path.
		logFilter := LogFilter{
			PodNamespace:  test.ServingNamespace,
			PodSelector:   metav1.ListOptions{LabelSelector: "app=activator"},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return GetMapValueAsString(m, "path") == "/" &&
					GetMapValueAsString(m, "authority") == privateURL.Host
			}}

		err = VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func VerifyEncryptedTrafficToApp(refs []corev1.ObjectReference, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		ksvcName, _, err := getKsvcNameAndURL(ctx, refs)
		if err != nil {
			t.Fatalf("Unable to get Knative Service URL: %v", err)
		}

		// activator -> application
		logFilter := LogFilter{
			PodNamespace:  environment.FromContext(ctx).Namespace(),
			PodSelector:   metav1.ListOptions{LabelSelector: "serving.knative.dev/service=" + ksvcName},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return GetMapValueAsString(m, "path") == "/" &&
					strings.HasPrefix(GetMapValueAsString(m, "upstream_cluster"), "inbound|80")
			}}

		err = VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func getKsvcNameAndURL(ctx context.Context, refs []corev1.ObjectReference) (string, *url.URL, error) {
	var (
		ksvcName string
		numKsvc  int
	)
	for _, ref := range refs {
		if ref.GroupVersionKind().GroupVersion() == knativeservice.GVR().GroupVersion() {
			// Make sure we verify traffic for the right Knative Service.
			// This is for safety and to guarantee the feature invariance.
			if numKsvc != 0 {
				return "", nil, fmt.Errorf("found more than one Knative Service: %s, %s", ksvcName, ref.Name)
			}
			ksvcName = ref.Name
			numKsvc++
		}
	}

	namespace := environment.FromContext(ctx).Namespace()
	ksvc, err := dynamicclient.Get(ctx).Resource(knativeservice.GVR()).Namespace(namespace).
		Get(ctx, ksvcName, metav1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("unable to get ksvc %s: %w", ksvcName, err)
	}

	address, _, _ := unstructured.NestedString(ksvc.Object, "status", "address", "url")
	privateURL, err := url.Parse(address)
	if err != nil {
		return "", nil, fmt.Errorf("unable to parse URL %s: %w", address, err)
	}

	return ksvcName, privateURL, nil
}

func VerifyPodLogsEncryptedRequestToHost(ctx context.Context, logFilter LogFilter) error {
	var (
		encrypted, unencrypted int
		err                    error
	)
	interval, timeout := k8s.PollTimings(ctx, nil)
	if pollErr := wait.PollImmediate(interval, timeout, func() (bool, error) {
		encrypted, unencrypted, err = getMatchingRequestsToHost(ctx, logFilter)
		if err != nil {
			return false, err
		}
		// Keep trying until we find matching lines.
		if encrypted == 0 && unencrypted == 0 {
			return false, nil
		}
		return true, nil
	}); pollErr != nil {
		return pollErr
	}

	if unencrypted != 0 {
		return fmt.Errorf("unencrypted request found in %v logs", logFilter.PodSelector)
	}
	return nil
}

func getMatchingRequestsToHost(ctx context.Context, logFilter LogFilter) (encrypted int, unencrypted int, err error) {
	podList, err := kubeclient.Get(ctx).CoreV1().Pods(logFilter.PodNamespace).List(context.Background(), logFilter.PodSelector)
	if err != nil {
		return 0, 0, fmt.Errorf("error listing pods in %s: %w", logFilter.PodNamespace, err)
	}
	if len(podList.Items) == 0 {
		return 0, 0, fmt.Errorf("no %v pods found in %s", logFilter.PodSelector, logFilter.PodNamespace)
	}

	for _, pod := range podList.Items {
		podName := pod.Name
		if err = ForEachLine(ctx, logFilter.PodNamespace, podName, logFilter.PodLogOptions, func(line string) error {
			var ret map[string]interface{}
			if err := json.Unmarshal([]byte(line), &ret); err == nil {
				if logFilter.JSONLogFilter(ret) {
					logging.FromContext(ctx).Infof("%s: %s", podName, line)
					downstreamTLSCipher := GetMapValueAsString(ret, "downstream_tls_cipher")
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

func GetMapValueAsString(m map[string]interface{}, key string) string {
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
