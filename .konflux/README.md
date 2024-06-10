# Konflux setup

This page describes the steps done to setup Konflux for SO.

## 1. Manifest folder

The CRs for the SO-Konflux setup are located in [.konflux/manifests](../.konflux/manifests).

## 2. Custom Build pipelines

The custom build pipeline are located under [./tekton](../.tekton).
This requires the [Red Hat Konflux](https://github.com/apps/red-hat-konflux) Github App installed and given access to the repositories (e.g. openshift-knative/serverless-operator and openshift-knative/eventing).

## 3. Adding registry.redhat.io pull secret

The custom pipeline for the index image has an `fbc-validate` task, which requires us to use `registry.redhat.io/openshift4/ose-operator-registry` as a base image for the index (otherwise the validation fails). Therefor we need to add a pull secret for the registry:
Therefor we [created](https://access.redhat.com/terms-based-registry/accounts) a registry service account and then created a pull secret with this information: https://access.redhat.com/terms-based-registry/token/konflux-openshift-serverless/docker-config:

0. if not done already: [Create](https://access.redhat.com/terms-based-registry/accounts) a registry service account (we used the [`konflux-openshift-serverless` SA](https://access.redhat.com/terms-based-registry/token/konflux-openshift-serverless)
1. Create a pull secret with this information (check e.g. [here](https://access.redhat.com/terms-based-registry/token/konflux-openshift-serverless/docker-config) for the credentials config of the `konflux-openshift-serverless` SA).
2. Change the `.metadata.name` to `registry-redhat-io-docker`
3. Apply the secret
4. Link the secret to the `appstudio-pipeline` service account: `oc secrets link appstudio-pipeline registry-redhat-io-docker`

--> maybe recheck on https://redhat-appstudio.github.io/docs.appstudio.io/Documentation/main/how-to-guides/Import-code/proc_importing_code/#configuring-your-application-to-use-a-red-hat-container-registry-token

Attention: I had some issues when creating the secret via the UI, as this creates a `kubernetes.io/dockercfg` instead of a `kubernetes.io/dockerconfigjson`.

