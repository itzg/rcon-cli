FROM scratch
COPY rcon-cli /usr/bin/
ENTRYPOINT /usr/bin/rcon-cli