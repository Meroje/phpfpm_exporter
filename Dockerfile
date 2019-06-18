ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/phpfpm_exporter /bin/phpfpm_exporter

ENTRYPOINT ["/bin/phpfpm_exporter"]
EXPOSE     9127
