package encryption

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/networking/pkg/certificates"
)

// this needs to match the ClusterIssuer in hack/lib/certmanager_resources/serving-ca-issuer.yaml.
const (
	certManagerCASecret  = "knative-selfsigned-ca"
	certManagerNamespace = "cert-manager"
)

func getCertManagerCA(clients *test.Clients) (*corev1.Secret, error) {
	var secret *corev1.Secret
	err := wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 1*time.Minute, true, func(context.Context) (bool, error) {
		caSecret, err := clients.Kube.CoreV1().Secrets(certManagerNamespace).Get(context.Background(), certManagerCASecret, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// CA not yet populated
		if len(caSecret.Data[certificates.CertName]) == 0 {
			return false, nil
		}

		secret = caSecret
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error while waiting for cert-manager self-signed CA to be popluated: %w", err)
	}

	return secret, nil
}
