# Release Process

This document describes the required steps to release a new version of Serverless Operator.

## Konflux

Regardless of the release type (minor or patch), we use Konflux for the build and release pipeline:

* Serverless Konflux Instance: [rh02](https://konflux-ui.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com)
* Serverless Konflux OCP Cluster: [rh02 console](https://console-openshift-console.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com/pipelines/ns/ocp-serverless-tenant)
* Tenant: `ocp-serverless-tenant`

### Konflux Access

To get access to the `ocp-serverless-tenant` namespace, you need to get the according permissions (contributor, maintainer, admin). 
You can request this from one of the admins in [#team-serverless](https://redhat.enterprise.slack.com/archives/CD87JDUB0) or in [#konflux-users](https://redhat.enterprise.slack.com/archives/C04PZ7H0VA8)

### Konflux UI

The rh02 UI is available [here](https://konflux-ui.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com). 

### Konflux CLI login (via `kubectl`)

1. Request [here](https://oauth-openshift.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com/oauth/token/request) a token for the cluster
2. Login via `oc login ...`
3. Access the cluster via `kubectl`

## Minor Release (1.X.0)

We have some automation in place, to generate & update the Konflux release pipelines and to keep the dependencies up to date. The generators for those live in the [hack](https://github.com/openshift-knative/hack) repo (in particular the [Generate CI config](https://github.com/openshift-knative/hack/actions/workflows/release-generate-ci.yaml) workflow). More details can be found in the [hack repo](#hack-repo) section.
The important part on creating a release is to make sure the relevant PRs are merged.

Preconditions:

1. Make sure the branch is cut correctly and the [Release Checklist](.github/ISSUE_TEMPLATE/release.md) was followed
2. Copy the Konflux ReleasePlanAdmissions (RPA) from the hack repo for the to-be-released version to the [konflux-release-data](https://gitlab.cee.redhat.com/releng/konflux-release-data) Gitlab repository into the [`config/kflux-prd-rh02.0fk9.p1/product/ReleasePlanAdmission/ocp-serverless`](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/config/kflux-prd-rh02.0fk9.p1/product/ReleasePlanAdmission/ocp-serverless) directory.

   Make sure the RPA is up-to-date. Sometimes there are some automated changes in the Gitlab repo from the Konflux team, which we'd need to include into the RPA generator to have the generated RPAs aligned.

Releasing:

1. Make sure, all dependent components have their Konflux related PRs merged. You can use this search query for example: [`is:pr org:openshift-knative "Update Konflux" state:open`](https://github.com/search?q=is%3Apr+org%3Aopenshift-knative+%22Update+Konflux%22+state%3Aopen&type=pullrequests)
2. Make sure, all component builds passed via the [build dashboard](https://openshift-knative.github.io/hack/) (mostly they failed only because of a flake and a retrigger should help. Otherwise some investigation is needed)
3. Update the ClusterServiceVersion (CSV) of the Serverless Operator to have the latest builds references. Therefore you can use the [validate.yaml](https://github.com/openshift-knative/serverless-operator/actions/workflows/validate.yaml) workflow, which will create a PR to update the manifests
4. After the CSV got updated, wait for the Konflux Builds of Serverless Operator to pass (e.g. via the dashboard or the GitHub UI). When the builds passed, update the OLM catalog (FBC) to reference the new bundle (containing the CSV). This can be done also via the [validate.yaml](https://github.com/openshift-knative/serverless-operator/actions/workflows/validate.yaml) workflow.
5. After the catalogs got updated, we need to update our override snapshot. The later release will point to this one and define, which images to use. The override snapshot update can also be triggered via the [validate.yaml](https://github.com/openshift-knative/serverless-operator/actions/workflows/validate.yaml) workflow.
6. Create a stage release via the [Apply Konflux Override Snapshot and create Release CR PR](https://github.com/openshift-knative/hack/actions/workflows/generate-release-crs.yaml) workflow. This applies the override-snapshot and creates a PR with a Release CR, which will then trigger a Konflux Release Pipeline. After the PR is merged and the Release CR got applied (e.g. as the [Apply Konflux Manifests](https://github.com/openshift-knative/hack/actions/workflows/apply-konflux-manifests.yaml) workflow runs on manifests changes), you can follow the release pipeline via the Konflux UI. It is important to merge the component PR first and make sure the release pipeline pass before merging the FBC release PR, as the FBC release references the components.
7. After QE, Docs team, etc. signed-off on the release and agreed to release to prod, you can trigger a prod release via the same workflow as above, targeting `prod` this time. Also make sure to have a successful components release, before merging the FBC release PR!

**Hints:**

* We have some CI checks in place ([CSV revision check](https://github.com/openshift-knative/serverless-operator/actions/workflows/csv-revision-check.yaml)), which make sure that all builds of one component builds belong to the same Git SHA. This helps to detect build issues and makes sure that the built versions are aligned (e.g. no mixes of commit1 with commit2)
* To reduce noise, there should be a code-freeze on the release branches for the components during step 4-6, otherwise the CSV will get updated with the new built images during the validate.yaml workflow runs.

# hack Repo

The [hack repository](https://github.com/openshift-knative/hack) contains all the automation to generate the Konflux (and Prow) manifests.

The most important workflows are:

| Workflow                                                                                                                                          | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
|---------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [Generate CI config](https://github.com/openshift-knative/hack/actions/workflows/release-generate-ci.yaml)                                        | Iterates over all our repositories (from the [./config](https://github.com/openshift-knative/hack/tree/main/config) folder) and makes sure the Konflux pipeline & manifests (components, image repositories, ...), Prow configs (tests), rpm.lock files, Dockerfiles (the Dockerfiles are generated via the [Dockerfile generator](https://github.com/openshift-knative/hack/blob/main/cmd/generate/generate.go) to have the correct labels, base images, etc.) or OWNER files are up to date on each repository.                                                                                                                                                                                                                                           |
| [Discover Branches](https://github.com/openshift-knative/hack/actions/workflows/release-discover-branches.yaml)                                   | Iterates over all our repositories and checks if there are new branches, which need to get added to the configs (e.g. like [here](https://github.com/openshift-knative/hack/blob/ddeab515c6f51b066f87cc84e6a7a01274e2a6c3/config/eventing.yaml#L49-L61). Based on this information the "Generate CI config" workflow updates / creates the CI configs)                                                                                                                                                                                                                                                                                                                                                                                                      |
| [Apply Konflux Manifests](https://github.com/openshift-knative/hack/actions/workflows/apply-konflux-manifests.yaml)                               | Cronjob to apply the Konflux manifests, so that Konflux is aware of them                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| [Apply Konflux Override Snapshot and create Release CR PR](https://github.com/openshift-knative/hack/actions/workflows/generate-release-crs.yaml) | Job, which will apply the given override snapshot (for a given Serverless Operator revision on branch), and creates a PR with the Konflux Release CR (having the override-snapshot linked) target environment (`prod` means, that this will be release to the Red Hat production catalog - `staging` is good for Release Candidates and testing). When the created PR is merged, and the release CR got applied (done automatically via the "Apply Konflux Manifests" workflow), then a Konflux release pipeline will be triggered. <br /> ⚠️ It is important to merge the component PR first and let the release pipeline succeed before merging the FBC release PR, as the FBC references the components and thus needs them to be released first. |

