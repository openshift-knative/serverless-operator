FROM quay.io/operator-framework/upstream-opm-builder:v1.14.3

RUN echo "http://nl.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && \
    echo "http://nl.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk update && apk add tzdata podman

# Add internal registry to insecure registries.
RUN printf "[registries.insecure]\nregistries = ['image-registry.openshift-image-registry.svc:5000']" > /etc/containers/registries.conf

# Create a barebones /etc/nsswitch.conf file.
# Required to avoid "Error: open /etc/nsswitch.conf: permission denied".
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf
