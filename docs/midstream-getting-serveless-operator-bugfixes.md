# Getting bugfixes for midstream Serverless Operator

## 1. Introduction

This document explains how to update the midstream Serverless operator image so that you can get bugfixes.

Serverless Operator midstream, unlike the fully released product, does not create patch releases.
When we need to fix a bug, we overwrite the existing image with a new one.

For example, when we identify a bug in 1.24 version, we change the code in 1.24 branch and the CI/CD overwrites the `1.24.0` image.

## 2. Prerequisites

- Usage of the midstream Serverless Operator and not the fully released product
- Access to OCP cluster with installed Serverless Operator
- Permission to restart deployments and delete replicasets in `openshift-serverless` namespace.
- If using Knative Eventing and Knative Kafka components, permission to delete pods and replicasets in `knative-eventing` namespace.
- If using Knative Serving, permission to delete pods and replicasets in `knative-serving`, `knative-serving-ingress` namespaces.

## 3. Execute/Resolution

### 3.1. Update Serverless operator images

1. Get the Serverless Operator images used in the OpenShift cluster right now, for later comparison:
  ```shell
  > kubectl get pods -n openshift-serverless -o jsonpath='{range .items[*]}{"\n"}{.metadata.name}{": "}{range .status.containerStatuses[*]}{.imageID}{end}{end}' | sort 
  knative-openshift-5dbb4bd4b7-q6wdp: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:26c3127e3dab1e102b94c305916d405b57d397c56348e0d73c7e1c2cddf3310b
  knative-openshift-ingress-78766bdc5c-zfzfl: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:7a417307328f76462560c5d4f814566135e59fc1b20758be720d090047ec682e
  knative-operator-65487bf7fc-vbgkk: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:589735abf033c61247e594f3fba9e7d87ac1093d1ebc0abe73f825944fe5e465
  ```
2. Restart Serverless Operator deployments to force the pods to be recreated with the new image (we explicitly set `imagePullPolicy` to `Always`):
  ```shell
  > kubectl rollout restart deployment -n openshift-serverless -l "olm.owner.namespace=openshift-serverless" -l "olm.owner.kind"="ClusterServiceVersion"
  deployment.apps/knative-openshift restarted
  deployment.apps/knative-openshift-ingress restarted
  deployment.apps/knative-operator restarted
  ```

### 3.2. Cleaning up Knative Serving control plane

**Note: This step is only required if you are using Knative Serving.**

1. Delete Knative Serving control plane replicasets. They will be recreated by the operator: 
  ```shell
  > kubectl delete replicasets.apps -n knative-serving -l 'app in (controller, webhook)'
  replicaset.apps "controller-57466669cf" deleted
  replicaset.apps "controller-7bb7748cd8" deleted
  replicaset.apps "webhook-6cb8b848bd" deleted
  ```

2. Delete jobs as their images cannot be mutated:
  ```shell
  > kubectl delete jobs -n knative-serving -l 'app in (storage-version-migration-serving)'
  job.batch "storage-version-migration-serving-serving-1.3.0" deleted
  ```

3. Delete Knative Serving Ingress control plane replicasets. They will be recreated by the operator:
  ```shell
  > kubectl delete replicasets.apps -n knative-serving-ingress -l 'app in (net-kourier-controller)'
  replicaset.apps "net-kourier-controller-84d4b75589" deleted
  ```

### 3.3. Cleaning up Knative Eventing and Knative Kafka control plane

**Note: This step is only required if you are using Knative Eventing and/or Knative Kafka components.**

1. Delete Knative Eventing and Knative Kafka control plane replicasets. They will be recreated by the operator:
  ```shell
  > kubectl delete replicasets.apps -n knative-eventing -l 'app in (kafka-controller, eventing-controller, eventing-webhook, kafka-webhook-eventing)'
  replicaset.apps "eventing-controller-5999f874f8" deleted
  replicaset.apps "eventing-webhook-86dd7d855b" deleted
  replicaset.apps "kafka-controller-6bd78d9f4f" deleted
  replicaset.apps "kafka-webhook-eventing-8444b7ccb4" deleted
  ```

2. Delete jobs as their images cannot be mutated:
  ```shell
  > kubectl delete jobs -n knative-eventing -l 'app in (kafka-controller-post-install, knative-kafka-storage-version-migrator, storage-version-migration-eventing)'
  job.batch "kafka-controller-post-install-1.24.0" deleted
  job.batch "knative-kafka-storage-version-migrator-1.24.0" deleted
  ```

## 4. Validate

1. Get the Serverless Operator images used in the OpenShift cluster again and validate that they are changed:

  ```shell
  > kubectl get pods -n openshift-serverless -o jsonpath='{range .items[*]}{"\n"}{.metadata.name}{": "}{range .status.containerStatuses[*]}{.imageID}{end}{end}' | sort
  knative-openshift-555d4d98d7-q6wdp: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  knative-openshift-ingress-7bc7c7b47b-zfzfl: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  knative-operator-67c4958cc6-vbgkk: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0@sha256:different-hash
  ```

2. If using Knative Serving, make sure `KnativeServing` CR is `Ready`:
  ```shell
  > kubectl get knativeserving -n knative-serving knative-serving -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
  True
  ```

3. If using Knative Eventing and Knative Kafka components, make sure `KnativeEventing` and `KnativeKafka` CRs are `Ready`:
  ```shell
  > kubectl get knativeeventing -n knative-eventing knative-eventing -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
  True
  > kubectl get knativekafka    -n knative-eventing knative-kafka    -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
  True
  ```

## 5. Troubleshooting

* `KnativeServing` / `KnativeEventing`/ `KnativeKafka` CRs don't get ready:
  * Make sure you give the operators enough time to reconcile the CRs and recreate images. It can take up to 5 minutes.
  * Watch conditions of the CRs and check for errors. If there are errors, check the logs of the operator pods.
    ```shell
    # KnativeServing
    > kubectl get knativeserving -n knative-serving knative-serving -o jsonpath="{range .status.conditions[*]} LastTransitionTime:{.lastTransitionTime} Type:{.type} Status:{.status}  Reason:{.reason}{'\n'}{end}"
    LastTransitionTime:2022-10-07T11:45:35Z Type:DependenciesInstalled Status:True  Reason:
    LastTransitionTime:2022-10-07T11:46:10Z Type:DeploymentsAvailable Status:True  Reason:
    LastTransitionTime:2022-10-07T11:45:35Z Type:InstallSucceeded Status:True  Reason:
    LastTransitionTime:2022-10-07T11:46:10Z Type:Ready Status:True  Reason:
    LastTransitionTime:2022-10-07T11:45:09Z Type:VersionMigrationEligible Status:True  Reason:
    
    # KnativeEventing
    > kubectl get knativeeventing -n knative-eventing knative-eventing -o jsonpath="{range .status.conditions[*]} LastTransitionTime:{.lastTransitionTime} Type:{.type} Status:{.status}  Reason:{.reason}{'\n'}{end}"
    LastTransitionTime:2022-10-07T07:16:58Z Type:DependenciesInstalled Status:True  Reason:
    LastTransitionTime:2022-10-07T07:17:28Z Type:DeploymentsAvailable Status:True  Reason:
    LastTransitionTime:2022-10-07T07:16:58Z Type:InstallSucceeded Status:True  Reason:
    LastTransitionTime:2022-10-07T07:17:28Z Type:Ready Status:True  Reason:
    LastTransitionTime:2022-10-07T07:16:29Z Type:VersionMigrationEligible Status:True  Reason:
    
    # KnativeKafka
    > kubectl get knativekafka -n knative-eventing knative-kafka -o jsonpath="{range .status.conditions[*]} LastTransitionTime:{.lastTransitionTime} Type:{.type} Status:{.status}  Reason:{.reason}{'\n'}{end}"
    LastTransitionTime:2022-10-07T11:49:46Z Type:DeploymentsAvailable Status:True  Reason:
    LastTransitionTime:2022-10-07T07:16:31Z Type:InstallSucceeded Status:True  Reason:
    LastTransitionTime:2022-10-07T11:49:46Z Type:Ready Status:True  Reason:
    ```
  * Check operator logs:
  ```shell
  > kubectl logs -n openshift-serverless -l 'name in (knative-operator, knative-openshift, knative-openshift-ingress)'
  ```
