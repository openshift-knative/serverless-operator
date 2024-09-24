## Installing components with Service Mesh

```shell
make images install-strimzi install-kafka-with-mesh install-serving-with-mesh
```

## Manually testing basic cases

### Setup resources

```shell
dir=docs/service-mesh/eventing
# Create Kafka Broker
kubectl apply -n serverless-tests -f "${dir}/resources/kafka-broker-example.yaml"
# Create KafkaNamespaced Broker
kubectl apply -n serverless-tests -f "${dir}/resources/kafka-broker-namespaced-example.yaml"
# Create PingSource
kubectl apply -n serverless-tests -f "${dir}/resources/ping-source.yaml"
# Create KafkaSource and KafkaSink
kubectl apply -n kafka -f "${dir}/resources/kafka-source-topic.yaml"
kubectl apply -n serverless-tests -f "${dir}/resources/kafka-source-example.yaml"
kubectl apply -n serverless-tests -f "${dir}/resources/kafka-sink-example.yaml"
kubectl apply -n serverless-tests -f "${dir}/resources/ping-source-kservice.yaml"
```

Wait for them to become ready by running:

```shell
kubectl get all -n serverless-tests
```

### Using a custom source

Run the custom source with Istio's proxy injected:

```shell
kubectl -n serverless-tests run curl --labels=sidecar.istio.io/inject=true --image=radial/busyboxplus:curl -i --tty --rm
```

Send an event to a broker address:

```shell
address=http://kafka-broker-ingress.knative-eventing.svc.cluster.local/serverless-tests/br
curl -X POST -v -H "content-type: application/json"  -H "ce-specversion: 1.0"  -H "ce-source: my/curl/command"  -H "ce-type: my.demo.event"  -H "ce-id: 0815"  -d '{"name":"Eventing"}' "${address}"
```

### Viewing events

```shell
kubectl logs -n serverless-tests event-display -f
```

It should be receiving events from:

- Kafka Broker coming from PingSource
- KafkaNamespaced Broker coming from PingSource
- KafkaSource

### Testing

Automated tests for Eventing with Istio are in https://github.com/openshift-knative/eventing-istio,
this repository uses those tests as part of the `test-upstream-e2e-mesh` Makefile target.

### Notes

```shell
{"@timestamp":"2023-03-15T15:36:33.064Z","@version":"1","message":"Couldn't resolve server my-cluster-kafka-bootstrap.kafka:9092 from bootstrap.servers as DNS resolution failed for my-cluster-kafka-bootstrap.kafka","logger_name":"org.apache.kafka.clients.ClientUtils","thread_name":"vert.x-eventloop-thread-2","level":"WARN","level_value":30000}
```
