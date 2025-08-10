FROM scratch

LABEL org.opencontainers.image.source=https://github.com/scholzj/strimzi-backup
LABEL org.opencontainers.image.title="Strimzi Backup"
LABEL org.opencontainers.image.description="Simple utility to backup and restore your Strimzi-based Apache Kafka cluster"

ARG TARGETOS
ARG TARGETARCH

ADD strimzi-backup-*-${TARGETOS}-${TARGETARCH} /strimzi-backup

CMD ["/strimzi-backup"]