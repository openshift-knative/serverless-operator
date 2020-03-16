package metrics

import (
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog"
)

var (
	consoleURL = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Name: "console_url",
			Help: "URL of the console exposed on the cluster",
		},
		[]string{"url"},
	)

	consoleBuildInfo = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Name: "openshift_console_operator_build_info",
			Help: "A metric with a constant '1' value labeled by major, minor, git commit & git version from which OpenShift Console Operator was built.",
		},
		[]string{"major", "minor", "gitCommit", "gitVersion"},
	)
)

func init() {
	legacyregistry.MustRegister(consoleURL)
	legacyregistry.MustRegister(consoleBuildInfo)
}

func HandleConsoleURL(oldURL, newURL string) {
	// if neither have been set, there is nothing to update
	if noHost(oldURL, newURL) {
		klog.V(4).Infof("metric console_url has no host")
		return
	}

	// only a new URL
	if isNewHost(oldURL, newURL) {
		klog.V(4).Infof("metric console_url new host: %s %s", oldURL, newURL)
		consoleURL.WithLabelValues(newURL).Set(1)
		return
	}

	// route or ingress update
	if isHostTransition(oldURL, newURL) {
		klog.V(4).Infof("metric console_url host transition: %s to %s", oldURL, newURL)
		consoleURL.WithLabelValues(oldURL).Set(0)
		consoleURL.WithLabelValues(newURL).Set(1)
		return
	}

	// something went wrong and we no longer have a route or ingress with a host
	if hostDied(oldURL, newURL) {
		klog.V(4).Infof("metric console_url host lost: %s %s", oldURL, newURL)
		consoleURL.WithLabelValues(oldURL).Set(0)
		return
	}
	klog.Error("metric console_url unhandled")
}

func noHost(old, new string) bool {
	return len(old) == 0 && len(new) == 0
}
func isNewHost(old, new string) bool {
	return len(old) == 0 && len(new) != 0
}
func isHostTransition(old, new string) bool {
	return len(old) != 0 && len(new) != 0
}
func hostDied(old, new string) bool {
	return len(old) != 0 && len(new) == 0
}

func RegisterVersion(major, minor, gitCommit, gitVersion string) {
	defer recoverMetricPanic()
	consoleBuildInfo.WithLabelValues(major, minor, gitCommit, gitVersion).Set(1)
}

// We will never want to panic our operator because of metric saving.
// Therefore, we will recover our panics here and error log them
// for later diagnosis but will never fail the operator.
func recoverMetricPanic() {
	if r := recover(); r != nil {
		klog.Errorf("Recovering from metric function - %v", r)
	}
}
