module github.com/openshift-knative/serverless-operator

go 1.18

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/coreos/go-semver v0.3.1
	github.com/google/go-cmp v0.5.9
	github.com/jaegertracing/jaeger v1.33.0
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20220525160904-9e1acff93e4a
	github.com/openshift/machine-config-operator v0.0.1-0.20220201192635-14a1ca2cb91f
	github.com/operator-framework/api v0.17.7
	github.com/operator-framework/operator-lifecycle-manager v0.25.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.67.1
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.67.1
	github.com/prometheus/client_golang v1.16.0
	github.com/prometheus/common v0.44.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.25.0
	golang.org/x/sync v0.3.0
	google.golang.org/grpc v1.57.0
	k8s.io/api v0.27.2
	k8s.io/apimachinery v0.27.2
	k8s.io/client-go v0.27.2
	knative.dev/eventing v0.37.1
	knative.dev/eventing-kafka-broker v0.37.0
	knative.dev/hack v0.0.0-20230712131415-ddae80293c43
	knative.dev/networking v0.0.0-20230419144338-e5d04e805e50
	knative.dev/operator v0.37.3
	knative.dev/pkg v0.0.0-20230718152110-aef227e72ead
	knative.dev/serving v0.37.0
	sigs.k8s.io/controller-runtime v0.15.0
	sigs.k8s.io/yaml v1.3.0
)

require (
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/cloudevents/conformance v0.2.0 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230525234035-dd9d682886f9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230629202037-9506855d4529 // indirect
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute v1.19.1 // indirect
	cloud.google.com/go/iam v0.13.0 // indirect
	cloud.google.com/go/storage v1.28.1 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.0 // indirect
	contrib.go.opencensus.io/exporter/zipkin v0.1.2 // indirect
	github.com/Shopify/sarama v1.37.0 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudevents/sdk-go/sql/v2 v2.13.0 // indirect
	github.com/cloudevents/sdk-go/v2 v2.14.0
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.3.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/gobuffalo/flect v0.2.4 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-containerregistry v0.13.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.1
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.8.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.3 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manifestival/client-go-client v0.5.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin/zipkin-go v0.3.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rickb777/date v1.14.1 // indirect
	github.com/rickb777/plural v1.2.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.9.2 // indirect
	github.com/tsenart/vegeta/v12 v12.8.4 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/wavesoftware/go-ensure v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.1 // indirect
	github.com/xdg-go/stringprep v1.0.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/automaxprocs v1.4.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/oauth2 v0.8.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/term v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/api v0.124.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230526161137-0005af68ea54 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	istio.io/api v0.0.0-20220420164308-b6a03a9e477e // indirect
	istio.io/client-go v1.13.3 // indirect
	k8s.io/apiserver v0.27.2 // indirect
	k8s.io/code-generator v0.27.2 // indirect
	k8s.io/component-base v0.27.2 // indirect
	k8s.io/gengo v0.0.0-20221011193443-fad74ee6edd9 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106
	knative.dev/caching v0.0.0-20230418150356-05a86ad5d7db // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

require (
	github.com/go-logr/logr v1.2.4
	go.uber.org/atomic v1.11.0 // indirect
	k8s.io/apiextensions-apiserver v0.27.2
	knative.dev/reconciler-test v0.0.0-20230810072039-8843b15af690
)

replace (
	// Knative components
	knative.dev/eventing => github.com/openshift-knative/eventing v0.99.1-0.20230811091754-f22178d5c46c
	knative.dev/eventing-kafka-broker => github.com/openshift-knative/eventing-kafka-broker v0.25.1-0.20230803130431-bf8f4e608e85
	knative.dev/hack => knative.dev/hack v0.0.0-20230417170854-f591fea109b3
	knative.dev/networking => knative.dev/networking v0.0.0-20230419144338-e5d04e805e50
	knative.dev/pkg => knative.dev/pkg v0.0.0-20230418073056-dfad48eaa5d0
	knative.dev/reconciler-test => knative.dev/reconciler-test v0.0.0-20230810072039-8843b15af690
	knative.dev/serving => github.com/openshift-knative/serving v0.10.1-0.20230721070749-1af735104330
)

replace (
	// OpenShift components
	github.com/openshift/api => github.com/openshift/api v0.0.0-20221013123533-341d389bd4a7
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20211209144617-7385dd6338e3
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20230616202415-59a8d308bfa4
)

replace (
	// Adjustments to align transitive deps
	k8s.io/api => k8s.io/api v0.25.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.4
	k8s.io/client-go => k8s.io/client-go v0.25.4
	k8s.io/code-generator => k8s.io/code-generator v0.25.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1
	k8s.io/utils => k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.12.3
)
