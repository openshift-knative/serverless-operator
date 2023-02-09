```shell

address=http://kafka-broker-ingress.knative-eventing.svc.cluster.local/default/br
curl -X POST -v -H "content-type: application/json"  -H "ce-specversion: 1.0"  -H "ce-source: my/curl/command"  -H "ce-type: my.demo.event"  -H "ce-id: 0815"  -d '{"name":"Eventing"}' "${address}"
```
