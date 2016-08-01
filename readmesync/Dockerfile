FROM busybox

ADD readmesync /bin/readmesync
ADD hugo /bin/hugo

VOLUME /hugoSource

ENV HUGO_OUTPUT=/hugoSource/public
ENV HUGO_SOURCE=/hugoSource
ENV HUGO_CONFIG=/hugoSource/config.toml
ENV HUGO_CMD=/bin/hugo
ENV OUTPUT_DIR=/hugoSource/content/project

EXPOSE 9001

ENTRYPOINT /bin/readmesync
