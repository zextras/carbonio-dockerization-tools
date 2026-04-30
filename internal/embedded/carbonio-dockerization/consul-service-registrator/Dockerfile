FROM docker:latest

RUN apk add --no-cache jq curl
WORKDIR /app

COPY entrypoint.sh entrypoint.sh
RUN chmod +x /app/entrypoint.sh
ENTRYPOINT ["/bin/sh", "-c", "/app/entrypoint.sh"]
