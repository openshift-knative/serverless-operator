package consoleserver

// This file is a copy of the struct within the console itself:
//   https://github.com/openshift/console/blob/master/cmd/bridge/config.go
// These structs need to remain in sync.
//
// `yaml:",omitempty"` has not been applied to any of the properties currently
// in use by the operator.  This is for backwards compatibilty purposes. If
// we have been sending an empty string value, we will continue to send it.
// Anything we have not been explicitly setting should have the `yaml:",omitempty"` tag.

// Config is the top-level console server cli configuration.
type Config struct {
	APIVersion     string `yaml:"apiVersion"`
	Kind           string `yaml:"kind"`
	ServingInfo    `yaml:"servingInfo"`
	ClusterInfo    `yaml:"clusterInfo"`
	Auth           `yaml:"auth"`
	Customization  `yaml:"customization"`
	Providers      `yaml:"providers"`
	MonitoringInfo `yaml:"monitoringInfo,omitempty"`
}

// ServingInfo holds configuration for serving HTTP.
type ServingInfo struct {
	BindAddress string `yaml:"bindAddress,omitempty"`
	CertFile    string `yaml:"certFile,omitempty"`
	KeyFile     string `yaml:"keyFile,omitempty"`

	// These fields are defined in `HTTPServingInfo`, but are not supported for console. Fail if any are specified.
	// https://github.com/openshift/api/blob/0cb4131a7636e1ada6b2769edc9118f0fe6844c8/config/v1/types.go#L7-L38
	BindNetwork           string        `yaml:"bindNetwork,omitempty"`
	ClientCA              string        `yaml:"clientCA,omitempty"`
	NamedCertificates     []interface{} `yaml:"namedCertificates,omitempty"`
	MinTLSVersion         string        `yaml:"minTLSVersion,omitempty"`
	CipherSuites          []string      `yaml:"cipherSuites,omitempty"`
	MaxRequestsInFlight   int64         `yaml:"maxRequestsInFlight,omitempty"`
	RequestTimeoutSeconds int64         `yaml:"requestTimeoutSeconds,omitempty"`
}

// ClusterInfo holds information the about the cluster such as master public URL and console public URL.
type ClusterInfo struct {
	ConsoleBaseAddress string `yaml:"consoleBaseAddress,omitempty"`
	ConsoleBasePath    string `yaml:"consoleBasePath,omitempty"`
	MasterPublicURL    string `yaml:"masterPublicURL,omitempty"`
}

// Monitoring holds URLs for monitoring related services
type MonitoringInfo struct {
	AlertmanagerPublicURL string `yaml:"alertmanagerPublicURL,omitempty"`
	GrafanaPublicURL      string `yaml:"grafanaPublicURL,omitempty"`
	PrometheusPublicURL   string `yaml:"prometheusPublicURL,omitempty"`
	ThanosPublicURL       string `yaml:"thanosPublicURL,omitempty"`
}

// Auth holds configuration for authenticating with OpenShift. The auth method is assumed to be "openshift".
type Auth struct {
	ClientID            string `yaml:"clientID,omitempty"`
	ClientSecretFile    string `yaml:"clientSecretFile,omitempty"`
	OAuthEndpointCAFile string `yaml:"oauthEndpointCAFile,omitempty"`
	LogoutRedirect      string `yaml:"logoutRedirect,omitempty"`
}

// Customization holds configuration such as what logo to use.
type Customization struct {
	Branding             string `yaml:"branding,omitempty"`
	DocumentationBaseURL string `yaml:"documentationBaseURL,omitempty"`
	CustomProductName    string `yaml:"customProductName,omitempty"`
	CustomLogoFile       string `yaml:"customLogoFile,omitempty"`
}

type Providers struct {
	StatuspageID string `yaml:"statuspageID,omitempty"`
}

type HelmChartRepo struct {
	URL    string `yaml:"url,omitempty"`
	CAFile string `yaml:"caFile,omitempty"`
}

type Helm struct {
	ChartRepo HelmChartRepo `yaml:"chartRepository"`
}
