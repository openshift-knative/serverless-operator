## Installing Eventing with Service Mesh

```shell
make images install-kafka-with-mesh
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
# Create KafkaSource
kubectl apply -n serverless-tests -f "${dir}/resources/kafka-source-example.yaml"
```

### Custom source

```shell
address=http://kafka-broker-ingress.knative-eventing.svc.cluster.local/default/br
curl -X POST -v -H "content-type: application/json"  -H "ce-specversion: 1.0"  -H "ce-source: my/curl/command"  -H "ce-type: my.demo.event"  -H "ce-id: 0815"  -d '{"name":"Eventing"}' "${address}"
```

### Testing

Automated tests for Eventing with Istio are in https://github.com/openshift-knative/eventing-istio,
this repository uses those tests as part of the `test-upstream-e2e-mesh` Makefile target.
