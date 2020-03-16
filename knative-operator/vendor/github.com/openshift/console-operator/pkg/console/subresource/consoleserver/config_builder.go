package consoleserver

import (
	corev1 "k8s.io/api/core/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/pkg/console/subresource/util"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/klog"
)

const (
	clientSecretFilePath       = "/var/oauth-config/clientSecret"
	defaultIngressCertFilePath = "/var/default-ingress-cert/ca-bundle.crt"
	oauthEndpointCAFilePath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	// serving info
	certFilePath = "/var/serving-cert/tls.crt"
	keyFilePath  = "/var/serving-cert/tls.key"
)

// ConsoleServerCLIConfigBuilder
// Director will be DefaultConfigMap()
//
// b := ConsoleYamlConfigBuilder{}
// return the default config value immediately:
//   b.Config()
//   b.ConfigYAML()
// set all the values:
//   b.Host(host).LogoutURL("").Brand("").DocURL("").APIServerURL("").Config()
// set only some values:
//   b.Host().Brand("").Config()
type ConsoleServerCLIConfigBuilder struct {
	host              string
	logoutRedirectURL string
	brand             operatorv1.Brand
	docURL            string
	apiServerURL      string
	statusPageID      string
	customProductName string
	customLogoFile    string
	CAFile            string
	monitoring        map[string]string
}

func (b *ConsoleServerCLIConfigBuilder) Host(host string) *ConsoleServerCLIConfigBuilder {
	b.host = host
	return b
}
func (b *ConsoleServerCLIConfigBuilder) LogoutURL(logoutRedirectURL string) *ConsoleServerCLIConfigBuilder {
	b.logoutRedirectURL = logoutRedirectURL
	return b
}
func (b *ConsoleServerCLIConfigBuilder) Brand(brand operatorv1.Brand) *ConsoleServerCLIConfigBuilder {
	b.brand = brand
	return b
}
func (b *ConsoleServerCLIConfigBuilder) DocURL(docURL string) *ConsoleServerCLIConfigBuilder {
	b.docURL = docURL
	return b
}
func (b *ConsoleServerCLIConfigBuilder) APIServerURL(apiServerURL string) *ConsoleServerCLIConfigBuilder {
	b.apiServerURL = apiServerURL
	return b
}
func (b *ConsoleServerCLIConfigBuilder) CustomProductName(customProductName string) *ConsoleServerCLIConfigBuilder {
	b.customProductName = customProductName
	return b
}
func (b *ConsoleServerCLIConfigBuilder) CustomLogoFile(customLogoFile string) *ConsoleServerCLIConfigBuilder {
	if customLogoFile != "" {
		b.customLogoFile = "/var/logo/" + customLogoFile // append path here to prevent customLogoFile from always being just /var/logo/
	}
	return b
}

func (b *ConsoleServerCLIConfigBuilder) StatusPageID(id string) *ConsoleServerCLIConfigBuilder {
	b.statusPageID = id
	return b
}

func (b *ConsoleServerCLIConfigBuilder) DefaultIngressCert(useDefaultDefaultIngressCert bool) *ConsoleServerCLIConfigBuilder {
	if useDefaultDefaultIngressCert {
		b.CAFile = oauthEndpointCAFilePath
		return b
	}
	b.CAFile = defaultIngressCertFilePath
	return b
}

func (b *ConsoleServerCLIConfigBuilder) Monitoring(monitoringConfig *corev1.ConfigMap) *ConsoleServerCLIConfigBuilder {
	if monitoringConfig != nil {
		b.monitoring = monitoringConfig.Data
	}
	return b
}

func (b *ConsoleServerCLIConfigBuilder) Config() Config {
	return Config{
		Kind:           "ConsoleConfig",
		APIVersion:     "console.openshift.io/v1",
		Auth:           b.authServer(),
		ClusterInfo:    b.clusterInfo(),
		Customization:  b.customization(),
		ServingInfo:    b.servingInfo(),
		Providers:      b.providers(),
		MonitoringInfo: b.monitoringInfo(),
	}
}

func (b *ConsoleServerCLIConfigBuilder) ConfigYAML() (consoleConfigYAML []byte, marshallError error) {
	conf := b.Config()
	yml, err := yaml.Marshal(conf)
	if err != nil {
		klog.V(4).Infof("could not create config yaml %v", err)
		return nil, err
	}
	return yml, nil
}

func (b *ConsoleServerCLIConfigBuilder) servingInfo() ServingInfo {
	return ServingInfo{
		BindAddress: "https://[::]:8443",
		CertFile:    certFilePath,
		KeyFile:     keyFilePath,
	}
}

func (b *ConsoleServerCLIConfigBuilder) clusterInfo() ClusterInfo {
	conf := ClusterInfo{
		ConsoleBasePath: "",
	}

	if len(b.apiServerURL) > 0 {
		conf.MasterPublicURL = b.apiServerURL
	}
	if len(b.host) > 0 {
		conf.ConsoleBaseAddress = util.HTTPS(b.host)
	}
	return conf
}

func (b *ConsoleServerCLIConfigBuilder) monitoringInfo() MonitoringInfo {
	conf := MonitoringInfo{}
	if len(b.monitoring) > 0 {
		var monitoringURLs MonitoringInfo
		urls, err := yaml.Marshal(b.monitoring)
		if err != nil {
			return conf
		}
		err = yaml.Unmarshal(urls, &monitoringURLs)
		if err != nil {
			return conf
		}
		if len(monitoringURLs.AlertmanagerPublicURL) > 0 {
			conf.AlertmanagerPublicURL = monitoringURLs.AlertmanagerPublicURL
		}
		if len(monitoringURLs.GrafanaPublicURL) > 0 {
			conf.GrafanaPublicURL = monitoringURLs.GrafanaPublicURL
		}
		if len(monitoringURLs.PrometheusPublicURL) > 0 {
			conf.PrometheusPublicURL = monitoringURLs.PrometheusPublicURL
		}
		if len(monitoringURLs.ThanosPublicURL) > 0 {
			conf.ThanosPublicURL = monitoringURLs.ThanosPublicURL
		}
	}
	return conf
}

func (b *ConsoleServerCLIConfigBuilder) authServer() Auth {
	// we need this fallback due to the way our unit test are structured,
	// where the ConsoleServerCLIConfigBuilder object is being instantiated empty
	if b.CAFile == "" {
		b.CAFile = oauthEndpointCAFilePath
	}
	conf := Auth{
		ClientID:            api.OpenShiftConsoleName,
		ClientSecretFile:    clientSecretFilePath,
		OAuthEndpointCAFile: b.CAFile,
	}
	if len(b.logoutRedirectURL) > 0 {
		conf.LogoutRedirect = b.logoutRedirectURL
	}
	return conf
}

func (b *ConsoleServerCLIConfigBuilder) customization() Customization {
	conf := Customization{}
	if len(b.brand) > 0 {
		conf.Branding = string(b.brand)
	}
	if len(b.docURL) > 0 {
		conf.DocumentationBaseURL = b.docURL
	}
	if len(b.customProductName) > 0 {
		conf.CustomProductName = b.customProductName
	}
	if len(b.customLogoFile) > 0 {
		conf.CustomLogoFile = b.customLogoFile
	}
	return conf

}

func (b *ConsoleServerCLIConfigBuilder) providers() Providers {
	if len(b.statusPageID) > 0 {
		return Providers{
			StatuspageID: b.statusPageID,
		}
	}
	return Providers{}
}
