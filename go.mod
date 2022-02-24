module github.com/openshift-knative/serverless-operator

go 1.17

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/google/go-cmp v0.5.6
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.8.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210607005641-f05ea078ab46
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.49.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.32.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	knative.dev/eventing v0.27.1
	knative.dev/eventing-kafka v0.27.0
	knative.dev/eventing-kafka-broker v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20211210083629-92d8a0a00cb6
	knative.dev/networking v0.0.0-20211101215640-8c71a2708e7d
	knative.dev/operator v0.27.1
	knative.dev/pkg v0.0.0-20211210132429-e86584fd3c69
	knative.dev/serving v0.27.1
	sigs.k8s.io/controller-runtime v0.9.7
	sigs.k8s.io/yaml v1.3.0
)

require (
	cloud.google.com/go v0.98.0 // indirect
	cloud.google.com/go/storage v1.18.2 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.0 // indirect
	contrib.go.opencensus.io/exporter/zipkin v0.1.2 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/Shopify/sarama v1.30.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2 v2.4.1 // indirect
	github.com/cloudevents/sdk-go/v2 v2.4.1 // indirect
	github.com/cncf/udpa/go v0.0.0-20210930031921-04548b0d99d4 // indirect
	github.com/cncf/xds/go v0.0.0-20211011173535-cb28da3451f1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/envoyproxy/go-control-plane v0.9.10-0.20210907150352-cf90f659a021 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-kit/log v0.1.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.2 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gobuffalo/flect v0.2.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-containerregistry v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manifestival/client-go-client v0.5.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/openzipkin/zipkin-go v0.3.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.0-beta.2 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rickb777/date v1.14.1 // indirect
	github.com/rickb777/plural v1.2.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/tsenart/vegeta/v12 v12.8.4 // indirect
	github.com/wavesoftware/go-ensure v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/automaxprocs v1.4.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/crypto v0.0.0-20210920023735-84f357641f63 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211205041911-012df41ee64c // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881 // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.61.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211129164237-f09f9a12af12 // indirect
	google.golang.org/grpc v1.42.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiextensions-apiserver v0.21.4 // indirect
	k8s.io/apiserver v0.21.4 // indirect
	k8s.io/code-generator v0.21.4 // indirect
	k8s.io/component-base v0.21.4 // indirect
	k8s.io/gengo v0.0.0-20211129171323-c02415ce4185 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176 // indirect
	knative.dev/caching v0.0.0-20211210083629-d73b6fdf89f7 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace (
	// Kubernetes v1.21.4
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/code-generator => k8s.io/code-generator v0.21.4

	// Knative forks.
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20220128153812-24bd789baf96
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20220128120000-64108ef18344
	knative.dev/eventing-kafka-broker => github.com/openshift-knative/eventing-kafka-broker v0.25.1-0.20220207154621-fd992876ea56
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20220208090216-dbf02d888824
)
