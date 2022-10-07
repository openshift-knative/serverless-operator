# Getting bugfixes for Knative Eventing and Knative Kafka components when using midstream Serverless Operator

## 1. Introduction

This document explains how to update the midstream Serverless operator image so that you can get bugfixes.

Serverless Operator midstream, unlike the fully released product, does not create patch releases.
When we need to fix a bug, we overwrite the existing image with a new one.

For example, when we identify a bug in 1.24 version, we change the code in 1.24 branch and the CI/CD overwrites the `1.24.0` image.

NOTE: instructions are for Knative Eventing and Knative Eventing Kafka components.

## 2. Prerequisites

- Usage of the midstream Serverless Operator and not the fully released product
- Access to OCP cluster with installed Serverless Operator
- Permission to delete pods and replicasets in `openshift-serverless` and `knative-eventing` namespaces.

## 3. Execute/Resolution

1. Get the Serverless Operator images used in the OpenShift cluster right now, for later comparison:
  ```shell
  > kubectl get pods -n openshift-serverless -o jsonpath='{range .items[*]}{"\n"}{.metadata.name}{": "}{range .status.containerStatuses[*]}{.imageID}{end}{end}' | sort 
  knative-openshift-5dbb4bd4b7-q6wdp: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:26c3127e3dab1e102b94c305916d405b57d397c56348e0d73c7e1c2cddf3310b
  knative-openshift-ingress-78766bdc5c-zfzfl: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:7a417307328f76462560c5d4f814566135e59fc1b20758be720d090047ec682e
  knative-operator-65487bf7fc-vbgkk: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:589735abf033c61247e594f3fba9e7d87ac1093d1ebc0abe73f825944fe5e465
  ```
2. Delete Serverless Operator pods to force them to be recreated with the new image (we explicitly set `imagePullPolicy` to `Always`):
  ```shell
  > kubectl delete pods -n openshift-serverless -l 'name in (knative-operator, knative-openshift, knative-openshift-ingress)'
  pod "knative-openshift-555d4d98d7-8ns5b" deleted
  pod "knative-openshift-ingress-7bc7c7b47b-gc5c9" deleted
  pod "knative-operator-67c4958cc6-zg8b9" deleted
  ```

3. Delete Knative control plane replicasets. They will be recreated by the operator:
  ```
  > kubectl delete replicasets.apps -n knative-eventing -l 'app in (kafka-controller, eventing-controller, eventing-webhook, kafka-webhook-eventing)'
  ```

4. Delete jobs that are not needed anymore:
  ```
  > kubectl delete jobs -n knative-eventing -l 'app in (kafka-controller-post-install, knative-kafka-storage-version-migrator, storage-version-migration-eventing)'
  ```

5. Wait until the pods are recreated and the new image is used.
  

## 4. Validate

1. Get the Serverless Operator images used in the OpenShift cluster again and validate that they are changed:

  ```shell
  > kubectl get pods -n openshift-serverless -o jsonpath='{range .items[*]}{"\n"}{.metadata.name}{": "}{range .status.containerStatuses[*]}{.imageID}{end}{end}' | sort
  knative-openshift-555d4d98d7-q6wdp: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  knative-openshift-ingress-7bc7c7b47b-zfzfl: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  knative-operator-67c4958cc6-vbgkk: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  ```

## 5. Troubleshooting

N/A
