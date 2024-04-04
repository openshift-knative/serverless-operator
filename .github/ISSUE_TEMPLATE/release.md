---
name: Release Checklist
about: Start a new release checklist
title: 'Release x.xx'
labels: ''
assignees: ''
---

# Release Checklist

## Resources
https://github.com/openshift-knative/serverless-operator/blob/main/README.md#create-a-new-version
Video from pierdipi showing the release process

## Pre-checks
- [ ] Check if the OCP versions in the release-matrix match [our current CI config](https://github.com/openshift-knative/hack/tree/main/config). If not, fix the versions first
- [ ] Check if there are open PRs for Serverless-Operator that need to be merged
- [ ] Check with the teams if they have all the necessary patches in the dependent repositories
- [ ] [Run the `validate` action in `serverless-operator` and wait for it to complete](https://github.com/openshift-knative/serverless-operator/actions/workflows/validate.yaml)
- [ ] [Merge PRs created by GitHub actions](https://github.com/openshift-knative/serverless-operator/pulls/app%2Fgithub-actions)
- [ ] Make sure that all builds are passing

## Cutting the new release branch
- [ ] Create a new branch from `main` with the pattern `release-x.xx`
- [ ] [Approve CI setup for `release-1.X` branch in `openshift/release`](https://github.com/openshift/release/pulls/serverless-qe) and wait for the PR to be merged.
- [ ] [Approve PRs created by GitHub actions bot in `serverless-operator`](https://github.com/openshift-knative/serverless-operator/pulls/app%2Fgithub-actions)
- [ ] Verify that [`knative-istio-authz-chart`](https://github.com/openshift-knative/knative-istio-authz-chart/branches) has a branch with the same name as the `release-1.X` branch created previously in   serverless-operator
- [ ] Verify that [`knative-istio-authz-chart`'s `Chart.yaml`](https://github.com/openshift-knative/knative-istio-authz-chart/blob/main/Chart.yaml) has `version` and `appVersion` set to the next version.

### Pre-checks
- [ ] Make sure that all dependent repos have a release branch for the new version and [CI set up](https://github.com/openshift-knative/hack)
- [ ] Ask the respective team to make sure that all relevant OpenShift specific patches are included in the new version
- [ ] Make sure that the [automated PR to point the CSV to the old branch is merged](https://github.com/openshift-knative/serverless-operator/pulls?q=is%3Apr+author%3Aapp%2Fgithub-actions+release-+is%3Aopen) like in https://github.com/openshift-knative/serverless-operator/pull/1881

### Serving Manifests
- [ ] Bump versions of `serving`, `serving_artifacts_branch`, `kourier`, `net_kourier_artifacts_branch`, `net_istio` and `net_istio_artifacts_branch`  [here](https://github.com/openshift-knative/serverless-operator/blob/main/olm-catalog/serverless-operator/project.yaml#L34)
- [ ] Run `make generated-files` and send a PR

### Eventing Manifests
- [ ] Bump versions of `eventing*`  [here](https://github.com/openshift-knative/serverless-operator/blob/main/olm-catalog/serverless-operator/project.yaml#L34)
- [ ] Run `make generated-files` and send a PR

### Operator Manifests
- [ ] Bump versions of `operator`  [here](https://github.com/openshift-knative/serverless-operator/blob/main/olm-catalog/serverless-operator/project.yaml#L34)
- [ ] Run `make generated-files` and send a PR (like [this](https://github.com/openshift-knative/serverless-operator/pull/2177) for 1.10)


### Bump all golang deps
- [ ] Update all versions  [here](https://github.com/openshift-knative/serverless-operator/blob/main/hack/update-deps.sh#L18)
- [ ] Run `./hack/update-deps.sh --upgrade`
- [ ] Run `make generated-files`
- [ ] Send a PR with the changes
- [ ] Pray that it works 😸  Otherwise try bump in steps and/or find a dependency version mix (with `go mod replace`) that works
