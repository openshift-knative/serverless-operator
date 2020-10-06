FROM quay.io/operator-framework/upstream-opm-builder:v1.14.3

RUN echo "http://nl.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && \
    echo "http://nl.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk update && apk add podman

# Add internal registry to insecure registries.
RUN printf "[registries.insecure]\nregistries = ['image-registry.openshift-image-registry.svc:5000']" > /etc/containers/registries.conf

# Create a barebones /etc/nsswitch.conf file.
# Required to avoid "Error: open /etc/nsswitch.conf: permission denied".
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

# Run as nobody and create a properly owned HOME directory for podman to be able to write
# its config files. Also set that directory as a workdir to be able to create more files.
ENV USER=nobody
ENV HOME /home/$USER
RUN addgroup $USER root && mkdir -p $HOME && chown -R $USER:root $HOME && chmod -R g=u $HOME
USER $USER
WORKDIR $HOME
