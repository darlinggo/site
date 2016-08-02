FROM busybox

ADD readmesync /bin/readmesync
ADD hugo /bin/hugo
ADD ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

VOLUME /hugoSource

ENV HUGO_SOURCE=/hugoSource
ENV HUGO_CMD=/bin/hugo
ENV OUTPUT_DIR=/content/project

EXPOSE 9001

ENTRYPOINT /bin/readmesync
