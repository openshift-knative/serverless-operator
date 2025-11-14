# Project AGENTS.md for OpenShift Serverless Operator

This AGENTS.md file provides comprehensive guidance for AI assistants and coding agents (like Claude, Gemini, Cursor, and others) to work with this codebase.

This repository contains the **Red Hat OpenShift Serverless Operator**, which provides a collection of APIs to support deploying and serving serverless applications and functions on OpenShift. It manages the lifecycle of Knative Serving, Knative Eventing, and Knative Kafka components.

## Project Overview

The OpenShift Serverless Operator is an **Operator Lifecycle Manager (OLM)**-based operator that:
- Deploys and manages Knative Serving (serverless application runtime)
- Deploys and manages Knative Eventing (event-driven architecture)
- Deploys and manages Knative Kafka (Kafka integration for eventing)
- Integrates with OpenShift Service Mesh, Strimzi (Kafka operator), and distributed tracing
- Provides OpenShift-specific customizations and downstream patches to upstream Knative

## Project Structure and Repository Layout

```
serverless-operator/
├── hack/                       # Build, install, and test automation scripts
│   ├── install.sh             # Main installation script
│   ├── images.sh              # Image building script
│   ├── dev.sh                 # Development mode setup
│   ├── generate/              # Code generation scripts
│   └── lib/                   # Shared shell libraries
├── knative-operator/          # Upstream Knative operator vendored code
├── openshift-knative-operator/ # OpenShift-specific operator wrapper
├── pkg/                       # Go packages
│   ├── client/                # Kubernetes client helpers
│   ├── common/                # Common utilities
│   └── istio/                 # Service Mesh integration
├── test/                      # Test code and helpers
├── olm-catalog/               # OLM bundle metadata
│   └── serverless-operator/
│       └── project.yaml       # Version and dependency configuration
├── templates/                 # Templates for generated files
├── must-gather/               # Must-gather support for debugging
├── docs/                      # Documentation
├── .tekton/                   # Tekton CI/CD pipelines
├── .github/                   # GitHub Actions workflows
├── Makefile                   # Build targets and automation
└── README.md                  # Project documentation
```

## Upstream Relationships

This project maintains downstream forks and integrations with several upstream projects:
- **Knative Serving** - https://github.com/openshift-knative/serving (fork of knative.dev/serving)
- **Knative Eventing** - https://github.com/openshift-knative/eventing (fork of knative.dev/eventing)
- **Knative Kafka** - Integration with Apache Kafka via Knative Eventing
- **Strimzi** - Kafka operator for Kubernetes/OpenShift
- **Istio/Service Mesh** - For secure traffic and gateway management

## Development Environment Setup

### Prerequisites

Before working with this repository, ensure you have:

- **Container runtime**: `podman` (aliased to `docker`) or `docker` (17.05+)
- **Shell**: `bash` (4.0.0+)
- **Build tools**: `make`
- **Kubernetes tools**: `helm`
- **OpenShift cluster**: CRC (recommended for local development) or any OpenShift cluster
- **Go**: Version specified in `go.mod` (check current requirement)

### Recommended CRC Configuration

For local development with CodeReady Containers (CRC):

```bash
crc start --cpus=6 --memory 16384
```

This configuration has been tested to work with operator E2E tests.

### Environment Variables

- **`DOCKER_REPO_OVERRIDE`**: Set to your container registry (e.g., `quay.io/username`) for building custom images
- **`ON_CLUSTER_BUILDS`**: Set to `true` to build images on-cluster using OpenShift Build
- **`GOPATH`**: Required for upstream integration tests (must contain knative.dev/serving and knative.dev/eventing)

## Building

### Quick Start

```bash
# Format code, tidy dependencies, and build images
make images

# Build and push to your registry
export DOCKER_REPO_OVERRIDE=quay.io/username
make images
```

### Build Targets

- **`make images`**: Build and push all container images
- **`make dev`**: Deploy operator without Knative components (development mode)
- **`make install`**: Deploy operator with Serving and Eventing
- **`make install-all`**: Deploy operator with Serving, Eventing, and Kafka
- **`make install-serving`**: Deploy operator with only Knative Serving
- **`make install-eventing`**: Deploy operator with only Knative Eventing
- **`make install-kafka`**: Deploy operator with Knative Kafka (requires Strimzi)

### On-Cluster Builds

Instead of building locally with podman/docker:

```bash
ON_CLUSTER_BUILDS=true make images
# Images will be at: image-registry.openshift-image-registry.svc:5000/openshift-serverless-builds/<image_name>

# Install using those images
DOCKER_REPO_OVERRIDE=image-registry.openshift-image-registry.svc:5000/openshift-serverless-builds make install
```

## Testing

### Quick Local Testing

Run the most relevant tests (same as CI):

```bash
crc start --cpus=6 --memory 16384
export DOCKER_REPO_OVERRIDE=quay.io/username
make images test-operator
```

### Test Targets

#### Operator Tests
- **`make test-unit`**: Run unit tests
- **`make test-e2e`**: Run E2E tests (excluding Kafka)
- **`make test-e2e-with-kafka`**: Run E2E tests including Kafka components
- **`make test-operator`**: Run both unit and E2E tests
- **`make install-mesh test-e2e`**: Run E2E tests with Service Mesh integration

#### Upstream Integration Tests
- **`make test-upstream-upgrade`**: Install previous version and run Knative Serving upgrade tests
  - Requires: Running OCP cluster, Knative Serving images published to CI registry, `${GOPATH}/src/knative.dev/serving` checked out
- **`make test-upstream-e2e-no-upgrade`**: Run Knative Serving and Eventing E2E tests without upgrades
  - Requires: Both Serving and Eventing sources in GOPATH

#### Individual Tests

Run individual tests from the respective Knative repositories:

```bash
# From knative.dev/serving or knative.dev/eventing
make TEST=<test_name> BRANCH=<ci_promotion_name> test-e2e-local

# Build a single test image
make IMAGE=<image_name> DOCKER_REPO_OVERRIDE=<registry> test-image-single

# Run test with custom image
make TEST=<test_name> DOCKER_REPO_OVERRIDE=<registry> test-e2e-local
```

### Test Requirements

Tests require network access on first run to download `envtest` environment from `sigs.k8s.io/controller-runtime/tools/setup-envtest`.

## Linting and Code Quality

### Linting Tools Required

- **`woke`**: Detect non-inclusive language - https://github.com/get-woke/woke
- **`golangci-lint`**: Go linting - https://golangci-lint.run/
- **`shellcheck`**: Shell script linting - https://www.shellcheck.net/
- **`operator-sdk`**: OLM bundle validation - https://sdk.operatorframework.io/
- **`misspell`**: Spell checking - https://github.com/client9/misspell
- **`prettier`**: YAML formatting - https://prettier.io/

### Linting Commands

```bash
# Run all linters (same as CI)
make lint

# Auto-fix linting issues where possible
make fix-lint
```

Linting configuration:
- Go linting: `.golangci.yaml`
- Inclusive language: `.wokeignore`

## Code Generation and Release Files

### Updating Generated Files

The repository uses templates and code generation for consistency:

```bash
# Generate all release files from templates
make release-files

# Generate all files (release files + other generated content)
make generated-files
```

Templates are located in `templates/` and variables are defined in `olm-catalog/serverless-operator/project.yaml`.

### Key Configuration File

**`olm-catalog/serverless-operator/project.yaml`** contains:
- Project version and metadata
- Component dependencies (Knative Serving, Eventing, Kafka versions)
- OLM metadata (replaces, skipRange)
- Image coordinates

Always update this file when bumping versions or dependencies, then run `make generated-files`.

## Contributing and Version Management

### Developer Certificate of Origin (DCO)

All commits must be signed off with DCO. Add `-s` flag to git commits:

```bash
git commit -s -m "your commit message"
```

See [DCO](DCO) for details.

### Creating a New Version

Follow this process when creating a new version (usually after a release branch cut):

#### 1. Update Version Metadata

Edit `olm-catalog/serverless-operator/project.yaml`:
- Bump `project.version` (e.g., 1.12 → 1.13)
- Update `olm.replaces`
- Adjust `olm.skipRange`

Add old version to CatalogSource in `hack/lib/catalogsource.bash`.

```bash
# After changes, regenerate files
make generated-files
```

#### 2. Update Component Dependencies

Edit `olm-catalog/serverless-operator/project.yaml`:
- Update versions in `dependencies` section
- Review and update manifest downloads in `openshift-knative-operator/hack/update-manifests.sh`
- Review patches in `update-manifests.sh` (some may be removable or need adjustment)

```bash
make generated-files
```

#### 3. Update Go Dependencies

Edit `hack/update-deps.sh`:
- Update `KN_VERSION` for new Knative release
- Update `OCP_VERSION` if bumping minimum OpenShift version

```bash
./hack/update-deps.sh --upgrade
```

### Commit Strategy

When making version-related changes:
1. Commit configuration changes separately
2. Commit generated changes separately (for easier review)
3. Keep Go dependency updates in their own commit

## Integration Components

### Strimzi (Kafka Operator)

```bash
# Install Strimzi and Kafka cluster
make install-strimzi

# Uninstall
make uninstall-strimzi
```

Strimzi is installed in `kafka` namespace by default.

### Service Mesh

```bash
# Install Service Mesh operator, Istio Gateway, and PeerAuthentication
make install-mesh

# Uninstall
make uninstall-mesh
```

Enables secure traffic for Knative Serving + Eventing.

### Distributed Tracing

```bash
# Install tracing with OpenTelemetry (default)
make install-tracing-opentelemetry

# Install tracing with Zipkin
make install-tracing-zipkin

# Uninstall
make uninstall-tracing-opentelemetry
# or
make uninstall-tracing-zipkin
```

### Cert Manager

```bash
# Install cert-manager (required for some Kafka configurations)
make install-certmanager

# Uninstall
make uninstall-certmanager
```

### KEDA (Kubernetes Event-Driven Autoscaling)

```bash
# Install KEDA
make install-keda

# Uninstall
make uninstall-keda

# Install Kafka with KEDA integration
make install-kafka-with-keda
```

## Upgrade Testing

### Test Upgrade from Previous Version

```bash
# Install previous version
make install-previous

# Check InstallPlans
oc get installplan -n openshift-operators

# Manually approve the latest version
oc edit installplan <install-plan-name> -n openshift-operators
# Set spec.approved: true
```

The operator will automatically upgrade after approval.

## Operator Lifecycle Manager (OLM)

### Manual Subscription Creation

```bash
cat <<-EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator
  namespace: openshift-operators
spec:
  source: serverless-operator
  sourceNamespace: openshift-marketplace
  name: serverless-operator
  channel: stable
EOF
```

## Common Development Workflows

### Local Development Cycle

```bash
# 1. Make code changes
vim pkg/...

# 2. Build and deploy to CRC
export DOCKER_REPO_OVERRIDE=quay.io/username
make images dev

# 3. Install Knative components as needed
oc apply -f config/knativeserving.yaml
oc apply -f config/knativeeventing.yaml

# 4. Run tests
make test-unit
make test-e2e
```

### Working with Upstream Changes

```bash
# 1. Update dependency versions in project.yaml
vim olm-catalog/serverless-operator/project.yaml

# 2. Update Go dependencies
./hack/update-deps.sh --upgrade

# 3. Update manifests
./openshift-knative-operator/hack/update-manifests.sh

# 4. Generate release files
make generated-files

# 5. Test
make images install test-e2e
```

### Debugging with Must-Gather

The repository includes must-gather support for debugging production issues:

```bash
# Must-gather is located in must-gather/ directory
# Used by OpenShift support to collect diagnostic information
```

## Coding Conventions

- **Go version**: Target version specified in `go.mod`
- **Testing**: Use standard library `testing` package
- **Code formatting**: Run `go fmt` (automatically done by `make images`)
- **Dependencies**: Run `make tidy` or `go mod tidy` after adding dependencies
- **Inclusive language**: Code is checked with `woke` linter
- **Operator SDK**: Follow Operator SDK patterns and best practices
- **Shell scripts**: Must pass `shellcheck` linting

## Makefile Chaining

You can chain make targets for efficiency:

```bash
# Build images and deploy in one command
make images dev

# Build, install, and test
make images install test-e2e
```

## AI-Specific Guidance

### When Adding Features

1. **Operator logic**: Add to `openshift-knative-operator/` or vendor from upstream `knative-operator/`
2. **Tests**: Add unit tests in the same package, E2E tests in `test/`
3. **Generated files**: Always run `make generated-files` after changes to templates or `project.yaml`
4. **Linting**: Run `make fix-lint` before committing
5. **Dependencies**: Run `go mod tidy` and vendor if needed

### When Fixing Bugs

1. **Reproduce**: Try to reproduce with `make install` or specific component install
2. **Add test**: Add a regression test in `test/`
3. **Fix**: Make minimal changes
4. **Verify**: Run `make test-operator` to ensure no breakage
5. **Lint**: Run `make lint` before submitting

### When Updating Dependencies

1. **Check project.yaml**: Understand current dependency versions
2. **Update carefully**: Use `./hack/update-deps.sh --upgrade`
3. **Test thoroughly**: Run full test suite including upstream tests
4. **Document**: Note any breaking changes or migration steps

### Known Limitations

- **First test run**: Requires internet to download envtest environment
- **CRC resources**: Minimum 6 CPUs and 16GB RAM for reliable testing
- **Upstream tests**: Require GOPATH structure with knative.dev/serving and eventing checked out
- **Image builds**: Can be slow; consider on-cluster builds for faster iteration
- **Service Mesh tests**: Require additional cluster resources and longer setup time

## Continuous Integration

The repository uses:
- **GitHub Actions**: `.github/workflows/`
- **Tekton Pipelines**: `.tekton/`
- **Konflux Release**: `.konflux-release/`

CI runs:
- Linting (all linters)
- Unit tests
- E2E tests (with and without Kafka)
- Upgrade tests
- Operator bundle validation

## Additional Resources

- **Knative Documentation**: https://knative.dev/docs/
- **Operator Framework**: https://operatorframework.io/
- **OpenShift Documentation**: https://docs.openshift.com/
- **Tekton**: https://tekton.dev/

## Help and Support

- **README.md**: Primary documentation
- **docs/**: Additional documentation
- **CONTRIBUTING.md**: Contribution guidelines
- **DCO**: Developer Certificate of Origin
