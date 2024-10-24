FROM quay.io/openshift/origin-must-gather:4.11.0

# Save original gather script
RUN mv /usr/bin/gather /usr/bin/gather_original

# Copy all collection scripts to /usr/bin
COPY bin/* /usr/bin/

ENTRYPOINT /usr/bin/gather
