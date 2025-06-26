FROM registry.fedoraproject.org/fedora-minimal:40

ARG BIN_DIR=./service/

EXPOSE 8080/tcp

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD [ "sh", "-c", "curl -f http://localhost:8080/healthz || exit 1" ]

LABEL org.opencontainers.image.source="https://github.com/jamesread/Japella"
LABEL org.opencontainers.image.description="A social media & chat powertool."
LABEL org.opencontainers.image.documentation="https://jamesread.github.io/Japella/"
LABEL org.opencontainers.image.title="Japella"
LABEL org.opencontainers.image.vendor="Japella's Community of Contributors"
LABEL org.opencontainers.image.version={{.Version}}

COPY frontend/dist /usr/share/Japella/webui/
COPY var/config-skel/ /config/
COPY var/app-skel/ /app/
COPY ${BIN_DIR}japella /app/

RUN mkdir -p /config/exec/

VOLUME /config
VOLUME /usr/libexec/japella/

ENTRYPOINT [ "/app/japella" ]
