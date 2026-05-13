# How to Contribute

OpenShift Serverless projects are [Apache 2.0 licensed](LICENSE) and accept
contributions via GitHub pull requests.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

## Makefile targets

### Building

| Target | Description |
| --- | --- |
| `make images` | Build and push all container images. Requires `DOCKER_REPO_OVERRIDE` to be set. |
| `make generated-files` | Regenerate all templated files (release files, code generation, vendoring). |
| `make release-files` | Regenerate only the release files from templates. |

### Installing

| Target | Description |
| --- | --- |
| `make dev` | Deploy the operator only (no Knative components). Useful for development. |
| `make install` | Deploy the operator with Knative Serving and Eventing. |
| `make install-all` | Deploy the operator with Serving, Eventing, Kafka, and tracing. |
| `make install-serving` | Deploy the operator with Knative Serving only. |
| `make install-eventing` | Deploy the operator with Knative Eventing only. |
| `make install-kafka` | Deploy the operator with Eventing and Kafka. Requires `make install-strimzi`. |
| `make install-previous` | Same as `make install` but deploys the previous operator version (for upgrade testing). |
| `make install-mesh` | Install Service Mesh operator, Istio Gateway, and PeerAuthentication. |
| `make install-strimzi` | Install the Strimzi operator and a Kafka cluster in the `kafka` namespace. |
| `make install-certmanager` | Install cert-manager. |
| `make install-keda` | Install KEDA. |
| `make teardown` | Tear down the Serverless installation. |

### Testing

| Target | Description |
| --- | --- |
| `make test-unit` | Run unit tests. |
| `make test-e2e` | Install and run E2E tests (without Kafka). |
| `make test-e2e-with-kafka` | Install and run E2E tests including Kafka. |
| `make test-operator` | Run both unit and E2E tests. |
| `make test-upgrade` | Install previous version and run upgrade tests. |
| `make test-kitchensink-e2e` | Run the full kitchensink E2E test suite. |

### Linting

| Target | Description |
| --- | --- |
| `make lint` | Run all linters (same as CI). |
| `make fix-lint` | Auto-fix linting issues where possible. |

**Tip:** You can chain targets, e.g. `make images install` or `make images dev`.

### Deploying with custom images

When developing the operator, use `DOCKER_REPO_OVERRIDE` to build and deploy your own images:

```bash
export DOCKER_REPO_OVERRIDE=quay.io/username
make images install
```

To override a single component image (e.g. to test a custom eventing controller build),
set the corresponding environment variable before running `make images`. The image
variables use `${VAR:-default}` semantics, so any export takes precedence. The override
is baked into the CSV during the bundle build in `make images`, so you must rebuild
the images for the override to take effect:

```bash
export DOCKER_REPO_OVERRIDE=quay.io/username
export KNATIVE_EVENTING_CONTROLLER=quay.io/username/custom-eventing-controller:latest
make images install
```

The available image override variables are defined in
[`hack/lib/images.bash`](./hack/lib/images.bash). Some common ones:

| Variable | Component |
| --- | --- |
| `SERVERLESS_KNATIVE_OPERATOR` | Knative Operator |
| `SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR` | OpenShift Knative Operator |
| `SERVERLESS_INGRESS` | Ingress |
| `KNATIVE_EVENTING_CONTROLLER` | Eventing controller |
| `KNATIVE_EVENTING_WEBHOOK` | Eventing webhook |
| `KNATIVE_SERVING_CONTROLLER` | Serving controller |
| `KNATIVE_SERVING_WEBHOOK` | Serving webhook |

### Required linter tools

- [`woke`](https://github.com/get-woke/woke) to detect non-inclusive language
- [`golangci-lint`](https://golangci-lint.run/) to lint golang code
- [`shellcheck`](https://www.shellcheck.net/) to lint shell files
- [`operator-sdk`](https://sdk.operatorframework.io/docs/installation/) to lint the bundle files
- [`misspell`](https://github.com/client9/misspell) to lint typos
- [`prettier`](https://prettier.io/) to format YAML

## CI and build infrastructure

Images are built and released via [Konflux](https://konflux-ci.dev/). The OpenShift Serverless Builds run on the 
[rh02](https://konflux-ui.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com/) Konflux instance in the 
[ocp-serverless-tenant](https://console-openshift-console.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com/pipelines/ns/ocp-serverless-tenant).

All CI configuration (Konflux & Prow) for all repos are generated and updated via the Konflux manifest generator which lives in the [openshift-knative/hack](https://github.com/openshift-knative/hack) 
repository.

## Create a new version

To create a new version of the serverless-operator, follow the
[Release Checklist](./.github/ISSUE_TEMPLATE/release.md) issue template which contains
the full step-by-step process.
