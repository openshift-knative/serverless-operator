ARG OPM_IMAGE=registry.ci.openshift.org/origin/__OCP_VERSION__:operator-registry

FROM $OPM_IMAGE

LABEL operators.operatorframework.io.index.configs.v1=/configs

COPY catalog/ /configs

RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]
