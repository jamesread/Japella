FROM registry.fedoraproject.org/fedora-minimal:40-x86_64

LABEL org.opencontainers.image.source https://github.com/jamesread/japella

COPY var/config-skel/ /config/config.yaml
COPY japella /app/

RUN mkdir -p /config/exec/

ENTRYPOINT [ "/app/japella" ]
