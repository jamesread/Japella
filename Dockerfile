FROM registry.fedoraproject.org/fedora-minimal:40-x86_64

LABEL org.opencontainers.image.source https://github.com/jamesread/japella

COPY japella /app/

ENTRYPOINT [ "/japella" ]
