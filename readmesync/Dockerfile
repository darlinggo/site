FROM busybox

ADD readmesync /bin/readmesync

VOLUME /output

ENV OUTPUT_DIR=/output

EXPOSE 9001

ENTRYPOINT /bin/readmesync
