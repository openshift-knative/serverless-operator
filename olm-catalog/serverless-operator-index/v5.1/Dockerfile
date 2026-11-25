ARG OPM_IMAGE=registry.redhat.io/openshift5/ose-operator-registry-rhel9:v5.1

FROM $OPM_IMAGE

LABEL operators.operatorframework.io.index.configs.v1=/configs

COPY catalog/ /configs

RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]
