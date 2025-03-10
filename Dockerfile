FROM registry.fedoraproject.org/fedora-minimal:40-x86_64

EXPOSE 8080/tcp

LABEL org.opencontainers.image.source https://github.com/jamesread/japella
LABEL org.opencontainers.image.title Japella

COPY webui /usr/share/Japella/
COPY var/config-skel/ /config/config.yaml
COPY japella /app/

RUN mkdir -p /config/exec/

VOLUME /config

ENTRYPOINT [ "/app/japella" ]
