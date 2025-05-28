module github.com/openshift-knative/serverless-operator

go 1.24

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/coreos/go-semver v0.3.1
	github.com/google/go-cmp v0.7.0
	github.com/jaegertracing/jaeger v1.62.0
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20240510131258-f646d5f29250
	github.com/operator-framework/api v0.31.0
	github.com/operator-framework/operator-lifecycle-manager v0.32.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.82.2
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.82.2
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.64.0
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.14.0
	google.golang.org/grpc v1.72.1
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/client-go v0.32.3
	knative.dev/eventing v0.44.5
	knative.dev/eventing-kafka-broker v0.37.0
	knative.dev/hack v0.0.0-20250220110655-b5e4ff820460
	knative.dev/networking v0.0.0-20250117155906-67d1c274ba6a
	knative.dev/operator v0.44.8
	knative.dev/pkg v0.0.0-20250306143800-fff4f701c7af
	knative.dev/serving v0.44.1
	sigs.k8s.io/controller-runtime v0.20.4
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/IBM/sarama v1.43.3 // indirect
	github.com/cert-manager/cert-manager v1.16.3 // indirect
	github.com/cloudevents/conformance v0.2.0 // indirect
	github.com/coreos/go-oidc/v3 v3.9.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/rs/dnscache v0.0.0-20230804202142-fc85eb664529 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/exp v0.0.0-20250103183323-7d7fa50e5329 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250422160041-2d3770c4ea7f // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	k8s.io/gengo/v2 v2.0.0-20240911193312-2b36238f13e9 // indirect
	sigs.k8s.io/gateway-api v1.1.0 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
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
	github.com/cloudevents/sdk-go/v2 v2.16.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-containerregistry v0.20.3 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3 // indirect
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
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/magiconair/properties v1.8.9 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
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
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rickb777/date v1.14.1 // indirect
	github.com/rickb777/plural v1.2.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tsenart/vegeta/v12 v12.12.0 // indirect
	github.com/wavesoftware/go-ensure v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/api v0.198.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	istio.io/api v0.0.0-20231206023236-e7cadb36da57 // indirect
	istio.io/client-go v1.18.7 // indirect
	k8s.io/apiserver v0.32.3 // indirect
	k8s.io/code-generator v0.32.3 // indirect
	k8s.io/gengo v0.0.0-20240404160639-a0386bf69313 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	k8s.io/utils v0.0.0-20250321185631-1f6e0b77f77e
	knative.dev/caching v0.0.0-20250117155405-a76aa7cd2bb6 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
)

require (
	github.com/go-logr/logr v1.4.2
	go.uber.org/atomic v1.11.0 // indirect
	k8s.io/apiextensions-apiserver v0.32.3
	knative.dev/reconciler-test v0.0.0-20250411085513-cfed924c2716
)

replace (
	// Knative components
	knative.dev/eventing => github.com/openshift-knative/eventing v0.99.1-0.20250514204204-6fb978ae15c5
	knative.dev/eventing-kafka-broker => github.com/openshift-knative/eventing-kafka-broker v0.25.1-0.20250517033211-3771b93c8b09
	knative.dev/hack => knative.dev/hack v0.0.0-20250116150306-c142b4835bc5
	knative.dev/networking => knative.dev/networking v0.0.0-20250117155906-67d1c274ba6a
	knative.dev/pkg => knative.dev/pkg v0.0.0-20250117084104-c43477f0052b
	knative.dev/reconciler-test => knative.dev/reconciler-test v0.0.0-20250117161605-4965c77b4dfa
	knative.dev/serving => github.com/openshift-knative/serving v0.10.1-0.20250508055645-0f344759f14a
)

replace (
	// OpenShift components
	github.com/openshift/api => github.com/openshift/api v0.0.0-20250510010900-d21e7b969933
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20241107164952-923091dd2b1a
)

replace (
	// Kubernetes components
	k8s.io/api => k8s.io/api v0.31.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.31.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.31.7
	k8s.io/client-go => k8s.io/client-go v0.31.7
	k8s.io/code-generator => k8s.io/code-generator v0.31.7
	k8s.io/component-base => k8s.io/component-base v0.31.7
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.19.7
)
