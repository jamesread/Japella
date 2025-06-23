FROM registry.fedoraproject.org/fedora-minimal:40

EXPOSE 8080/tcp

LABEL org.opencontainers.image.source="https://github.com/jamesread/Japella"
LABEL org.opencontainers.image.description="A social media & chat powertool."
LABEL org.opencontainers.image.documentation="https://jamesread.github.io/Japella/"
LABEL org.opencontainers.image.title="Japella"
LABEL org.opencontainers.image.vendor="Japella's Community of Contributors"
LABEL org.opencontainers.image.version={{.Version}}

COPY webui /usr/share/Japella/webui/
COPY var/config-skel/ /config/
COPY japella /app/
COPY lang/ /app/lang/

RUN mkdir -p /config/exec/

VOLUME /config
VOLUME /usr/libexec/japella/

ENTRYPOINT [ "/app/japella" ]
