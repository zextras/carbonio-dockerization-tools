#!/bin/bash

export COMPOSE_FILE="docker-compose.yaml"
ALL_SERVICES=(carbonio-composed-ui carbonio-provisioner consul-register carbonio-preview preview-registrator)
if [[ "$1" == "--advanced" ]]; then
    export COMPOSE_FILE="docker-compose.yaml:docker-compose-advanced.yaml"
    shift
    ALL_SERVICES+=("$@")
else
    ALL_SERVICES+=("$@")
fi

echo "=========================================="
echo "=========================================="
echo "Starting with compose file: $COMPOSE_FILE"
echo "Services: ${ALL_SERVICES[*]}"
echo "=========================================="
echo "=========================================="
exec docker compose up -d --build --pull missing "${ALL_SERVICES[@]}"