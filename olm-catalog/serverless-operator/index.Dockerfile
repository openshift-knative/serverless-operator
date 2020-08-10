FROM quay.io/operator-framework/upstream-opm-builder

RUN echo "http://nl.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && \
    echo "http://nl.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk update && apk add podman

# Add internal registry to insecure registries.
RUN printf "[registries.insecure]\nregistries = ['image-registry.openshift-image-registry.svc:5000']" > /etc/containers/registries.conf

# Make /etc accessable by the entire root group.
# Required to avoid "Error: open /etc/nsswitch.conf: permission denied".
RUN chgrp -R 0 /etc && chmod -R g=u /etc

# Run as nobody and create a properly owned HOME directory for podman to be able to write
# its config files. Also set that directory as a workdir to be able to create more files.
ENV USER=nobody
ENV HOME /home/$USER
RUN mkdir -p $HOME && chgrp -R 0 $HOME && chmod -R g=u $HOME
USER $USER
WORKDIR $HOME
