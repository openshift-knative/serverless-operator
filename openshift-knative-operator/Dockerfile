FROM openshift/origin-release:golang-1.14 AS builder
COPY . .

WORKDIR openshift-knative-operator
ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ./cmd/operator

FROM openshift/origin-base
COPY --from=builder /tmp/operator /ko-app/operator

ENV KO_DATA_PATH="/var/run/ko"
COPY openshift-knative-operator/cmd/operator/kodata $KO_DATA_PATH

LABEL \
    com.redhat.component="openshift-serverless-1-tech-preview-knative-rhel8-operator-container" \
    name="openshift-serverless-1-tech-preview/knative-rhel8-operator" \
    version="1.11.0" \
    summary="Red Hat OpenShift Serverless 1 Openshift Knative Operator" \
    maintainer="serverless-support@redhat.com" \
    description="Red Hat OpenShift Serverless 1 Openshift Knative Operator" \
    io.k8s.display-name="Red Hat OpenShift Serverless 1 Openshift Knative Operator"

ENTRYPOINT ["/ko-app/operator"]
