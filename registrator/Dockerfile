FROM docker.io/alpine/curl

WORKDIR /app

COPY start.sh start.sh

ENV CONSUL_HTTP_ADDR="http://consul:8500"
ENV SERVICE_NAME="myService"
ENV SERVICE_PORT=8080
ENTRYPOINT ["/bin/sh", "-c", "/app/start.sh"]