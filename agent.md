# Agent Guide: Red Hat Serverless Operator

## Overview

The Red Hat Serverless Operator is a Kubernetes operator that provides serverless capabilities on OpenShift clusters. It manages the installation and lifecycle of Knative components (Serving, Eventing, and Kafka) on OpenShift.

**Current Version:** 1.37.0
**Supported OCP Versions:** 4.14 - 4.20
**Knative Version:** v1.17

## Architecture

### Component Structure

```
serverless-operator/
├── openshift-knative-operator/  # Main operator (OpenShift-specific)
├── knative-operator/            # Upstream Knative operator wrapper
├── serving/                     # Serving-specific customizations
├── pkg/                         # Shared packages
├── test/                        # E2E and integration tests
├── olm-catalog/                 # OLM/Operator Lifecycle Manager metadata
├── hack/                        # Build and development scripts
└── templates/                   # Manifest templates
```

### Key Operators

1. **OpenShift Knative Operator** (`openshift-knative-operator/`)
   - Main operator managing the lifecycle of Knative components
   - Handles OpenShift-specific integrations
   - Location: `openshift-knative-operator/pkg/`

2. **Knative Operator** (`knative-operator/`)
   - Wraps upstream Knative operator
   - Manages KnativeServing, KnativeEventing, and KnativeKafka CRs
   - Controllers: `knative-operator/pkg/controller/`

3. **Metadata Webhook** (`serving/metadata-webhook/`)
   - Provides defaults and validation for Knative resources
   - Ensures proper labeling and annotations

## Custom Resource Definitions (CRDs)

The operator manages these primary CRDs:

- **KnativeServing** - Manages Knative Serving installation
- **KnativeEventing** - Manages Knative Eventing installation
- **KnativeKafka** - Manages Kafka-based event sources and channels

## Dependencies

### Upstream Knative Components

- **Serving**: knative-v1.17
- **Eventing**: knative-v1.17
- **Eventing Kafka Broker**: knative-v1.17
- **Kourier** (networking): knative-v1.17
- **Istio** (networking): knative-v1.17

### OpenShift Components

- Minimum Kubernetes: 1.25.0
- Service Mesh integration support
- Istio Gateway and PeerAuthentication

## Development Workflow

### Prerequisites

- `podman` or `docker` (17.05+)
- `bash` (4.0.0+)
- `make`
- `helm`
- `go` 1.23
- Running OpenShift cluster (CRC recommended with 6 CPUs, 16GB RAM)

### Common Development Tasks

#### 1. Building and Testing Locally

```bash
# Set your docker repository
export DOCKER_REPO_OVERRIDE=quay.io/your-username

# Build images and run tests
make images test-operator
```

#### 2. Installing on Cluster

```bash
# Install operator + Serving + Eventing
make install

# Install with Kafka support
make install-all

# Install only Serving
make install-serving

# Install with Service Mesh
make install-mesh install
```

#### 3. Running Tests

```bash
# Unit tests
make test-unit

# E2E tests
make test-e2e

# E2E with Kafka
make test-e2e-with-kafka

# Upstream upgrade tests
make test-upstream-upgrade
```

### Making Changes

#### Updating Versions

1. **Update metadata**: Edit `olm-catalog/serverless-operator/project.yaml`
   - Bump `project.version`
   - Update `olm.replaces` and `olm.skipRange`
   - Update dependency versions in `dependencies` section

2. **Generate manifests**: Run `make generated-files`

3. **Update Go dependencies**:
   - Edit `hack/update-deps.sh` (update `KN_VERSION`)
   - Run `./hack/update-deps.sh --upgrade`

#### Adding/Modifying Operators

**Key Files:**
- Controllers: `knative-operator/pkg/controller/`
- APIs: `knative-operator/pkg/apis/operator/v1alpha1/`
- Common utilities: `pkg/common/`, `openshift-knative-operator/pkg/common/`

**Development Pattern:**
1. Add/modify CRD types in `pkg/apis/`
2. Implement controller logic in `pkg/controller/`
3. Add reconciliation logic
4. Update tests in `*_test.go` files
5. Run `make generated-files` to regenerate manifests

## Project Configuration

### project.yaml

The central configuration file: `olm-catalog/serverless-operator/project.yaml`

**Key Sections:**
- `project.version` - Current operator version
- `olm` - OLM metadata (channels, replaces, skipRange)
- `dependencies` - Upstream component versions
- `requirements` - Platform requirements (OCP, Kubernetes, Go versions)

### Image Management

Images are referenced in:
- `openshift-knative-operator/pkg/common/images.go`
- `knative-operator/pkg/controller/knativekafka/images.go`

Container image coordinates are templated from `project.yaml`.

## Monitoring and Observability

### Dashboards

Health and metrics dashboards:
- `knative-operator/pkg/monitoring/dashboards/`
- Grafana dashboards for Knative components
- Health metrics for operator status

### Service Monitors

Prometheus ServiceMonitors for event sources:
- `knative-operator/pkg/monitoring/sources/`

## Testing Strategy

### Test Hierarchy

1. **Unit Tests** (`*_test.go` in each package)
   - Test individual functions and components
   - Run with: `make test-unit`

2. **E2E Tests** (`test/e2e/`, `test/servinge2e/`, `test/eventinge2e/`)
   - Test full operator lifecycle
   - Test Knative Serving/Eventing functionality
   - Run with: `make test-e2e`

3. **Upgrade Tests**
   - Test upgrades from previous versions
   - Run with: `make test-upstream-upgrade`

4. **Kitchen Sink Tests** (`test/kitchensinke2e/`)
   - End-to-end feature tests
   - Tests complex scenarios

### Test Utilities

- `test/portforward.go` - Port forwarding helpers
- `knative-operator/pkg/webhook/testutil/` - Webhook testing utilities

## CI/CD Pipeline

### Konflux Integration

- `.konflux-release/` - Konflux release configurations
- `.tekton/` - Tekton pipeline definitions
- GitHub workflows: `.github/`

### Release Process

1. Create version branch
2. Update `project.yaml` with new version
3. Run `make generated-files`
4. Update Go dependencies via `hack/update-deps.sh`
5. Test upgrades from previous version
6. Create bundle and catalog images

## Key Patterns and Conventions

### Reconciliation Pattern

Controllers follow standard Kubernetes reconciliation:
1. Watch CRDs (KnativeServing, KnativeEventing, KnativeKafka)
2. Reconcile desired state from CR spec
3. Apply manifests and update status
4. Handle errors and requeue

### Manifest Management

Manifests are:
- Downloaded from upstream in `openshift-knative-operator/hack/update-manifests.sh`
- Stored in component directories
- Patched for OpenShift compatibility
- Applied during reconciliation

### Constants

Common constants defined in:
- `pkg/common/constants.go`
- `knative-operator/pkg/common/constants.go`
- `openshift-knative-operator/pkg/common/`

## Troubleshooting

### Common Issues

1. **Image pull errors**
   - Ensure `DOCKER_REPO_OVERRIDE` is set correctly
   - Check registry authentication

2. **Test failures**
   - Ensure cluster has sufficient resources (6 CPUs, 16GB RAM)
   - Check that previous test runs cleaned up properly

3. **Upgrade failures**
   - Verify `olm.replaces` and `olm.skipRange` are correct
   - Check that previous bundle images exist in catalog

### Debug Tools

- **Must-gather**: `must-gather/` - Collect debugging information
- **Port forwarding**: Use utilities in `test/portforward.go`
- **Logs**: Check operator logs via `kubectl logs -n openshift-serverless`

## Useful Make Targets

```bash
make images              # Build and push all images
make dev                 # Deploy operator only (no Knative components)
make install             # Deploy operator + Serving + Eventing
make install-all         # Deploy everything including Kafka
make install-mesh        # Install Service Mesh integration
make test-unit           # Run unit tests
make test-e2e            # Run E2E tests
make test-operator       # Run all tests
make generated-files     # Regenerate all generated files
make release-files       # Update release files from templates
make lint                # Run all linters
make fix-lint            # Auto-fix linting issues
```

## Code Navigation Tips

### Finding Controllers

Controllers are in `knative-operator/pkg/controller/`:
- `knativeserving/` - KnativeServing controller
- `knativeeventing/` - KnativeEventing controller
- `knativekafka/` - KnativeKafka controller

### Finding APIs

API definitions in `knative-operator/pkg/apis/operator/v1alpha1/`:
- `knativeserving_types.go`
- `knativeeventing_types.go`
- `knativekafka_types.go`

### Finding Common Utilities

- Image management: `openshift-knative-operator/pkg/common/images.go`
- Deployment helpers: `openshift-knative-operator/pkg/common/deployment_test.go`
- Label utilities: `openshift-knative-operator/pkg/common/label.go`
- Resource helpers: `openshift-knative-operator/pkg/common/resources.go`

## External Resources

- [Knative Documentation](https://knative.dev/docs/)
- [OpenShift Serverless Documentation](https://docs.openshift.com/serverless/)
- [Operator SDK](https://sdk.operatorframework.io/)
- [OLM Documentation](https://olm.operatorframework.io/)

## Contributing

See `CONTRIBUTING.md` and `DCO` for contribution guidelines.

### Linting Requirements

Required linters:
- `woke` - Non-inclusive language detection
- `golangci-lint` - Go code linting
- `shellcheck` - Shell script linting
- `operator-sdk` - Bundle validation
- `misspell` - Spell checking
- `prettier` - YAML formatting

Run all linters: `make lint`
Auto-fix issues: `make fix-lint`
