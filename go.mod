module github.com/openshift-knative/serverless-operator

go 1.22

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/coreos/go-semver v0.3.1
	github.com/google/go-cmp v0.6.0
	github.com/jaegertracing/jaeger v1.60.0
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20230120202327-72f107311084
	github.com/openshift/machine-config-operator v0.0.1-0.20220201192635-14a1ca2cb91f
	github.com/operator-framework/api v0.23.0
	github.com/operator-framework/operator-lifecycle-manager v0.25.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.73.2
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.73.2
	github.com/prometheus/client_golang v1.20.3
	github.com/prometheus/common v0.55.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/stretchr/testify v1.9.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.8.0
	google.golang.org/grpc v1.66.1
	k8s.io/api v0.29.3
	k8s.io/apimachinery v0.29.3
	k8s.io/client-go v0.29.3
	knative.dev/eventing v0.42.0
	knative.dev/eventing-kafka-broker v0.37.0
	knative.dev/hack v0.0.0-20240814130635-06f7aff93954
	knative.dev/networking v0.0.0-20240716111826-bab7f2a3e556
	knative.dev/operator v0.42.4
	knative.dev/pkg v0.0.0-20240716082220-4355f0c73608
	knative.dev/serving v0.42.1
	sigs.k8s.io/controller-runtime v0.17.3
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/IBM/sarama v1.43.2 // indirect
	github.com/cloudevents/conformance v0.2.0 // indirect
	github.com/coreos/go-oidc/v3 v3.9.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/rs/dnscache v0.0.0-20211102005908-e0241e321417 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
)

require (
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.2 // indirect
	contrib.go.opencensus.io/exporter/zipkin v0.1.2 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudevents/sdk-go/sql/v2 v2.15.2 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/eapache/go-resiliency v1.6.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-containerregistry v0.13.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manifestival/client-go-client v0.6.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rickb777/date v1.14.1 // indirect
	github.com/rickb777/plural v1.2.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tsenart/vegeta/v12 v12.11.1 // indirect
	github.com/wavesoftware/go-ensure v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/automaxprocs v1.5.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/api v0.183.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	istio.io/api v0.0.0-20231206023236-e7cadb36da57 // indirect
	istio.io/client-go v1.18.7 // indirect
	k8s.io/apiserver v0.29.3 // indirect
	k8s.io/code-generator v0.29.3 // indirect
	k8s.io/component-base v0.29.3 // indirect
	k8s.io/gengo v0.0.0-20240129211411-f967bbeff4b4 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240322212309-b815d8309940 // indirect
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57
	knative.dev/caching v0.0.0-20240716132144-989f54c83776 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

require (
	github.com/go-logr/logr v1.4.2
	go.uber.org/atomic v1.11.0 // indirect
	k8s.io/apiextensions-apiserver v0.29.3
	knative.dev/reconciler-test v0.0.0-20240716134925-00d94f40c470
)

replace (
	// Knative components
	knative.dev/eventing => github.com/openshift-knative/eventing v0.99.1-0.20240912142802-3ddd281e981d
	knative.dev/eventing-kafka-broker => github.com/openshift-knative/eventing-kafka-broker v0.25.1-0.20240912142823-cce3a989bfc1
	knative.dev/hack => knative.dev/hack v0.0.0-20240814135050-f9643117f6a1
	knative.dev/networking => knative.dev/networking v0.0.0-20240716111826-bab7f2a3e556
	knative.dev/pkg => knative.dev/pkg v0.0.0-20240716082220-4355f0c73608
	knative.dev/reconciler-test => knative.dev/reconciler-test v0.0.0-20240716134925-00d94f40c470
	knative.dev/serving => github.com/openshift-knative/serving v0.10.1-0.20240906121033-618992fcbd27
)

replace (
	// OpenShift components
	github.com/openshift/api => github.com/openshift/api v0.0.0-20231118005202-0f638a8a4705
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20230120202327-72f107311084
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20240813185428-b1a8f6946397
)

// This replacement can be removed once we bump to k8s.io@v1.30.x, as from release 1.30+
// https://github.com/kubernetes/code-generator/blob/release-1.30/go.mod, code-generator will use
// the new v2 version of kube-openapi.
replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00
