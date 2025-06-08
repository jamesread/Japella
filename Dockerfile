FROM registry.fedoraproject.org/fedora-minimal:40

EXPOSE 8080/tcp

LABEL org.opencontainers.image.source="https://github.com/jamesread/japella"
LABEL org.opencontainers.image.documentation="https://jamesread.github.io/Japella/"
LABEL org.opencontainers.image.title="Japella"
LABEL org.opencontainers.image.vendor="Japella's Community of Contributors"
LABEL org.opencontainers.image.version={{.Version}}

COPY webui /usr/share/Japella/webui/
COPY var/config-skel/ /config/config.yaml
COPY japella /app/

RUN mkdir -p /config/exec/

VOLUME /config
VOLUME /usr/libexec/japella/

ENTRYPOINT [ "/app/japella" ]
