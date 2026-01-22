#!/bin/bash

export COMPOSE_FILE="docker-compose.yaml"
ALL_SERVICES=(event-listener carbonio-composed-ui carbonio-provisioner consul-register carbonio-preview carbonio-storages)

# Parse flags in any order
while [[ "$1" == --* ]]; do
    case "$1" in
        --advanced)
            COMPOSE_FILE="${COMPOSE_FILE}:docker-compose-advanced.yaml"
            shift
            ;;
        --monitoring)
            COMPOSE_FILE="${COMPOSE_FILE}:monitoring.yaml"
            shift
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Remaining arguments are additional services
ALL_SERVICES+=("$@")

echo "=========================================="
echo "=========================================="
echo "Starting with compose file: $COMPOSE_FILE"
echo "Services: ${ALL_SERVICES[*]}"
echo "=========================================="
echo "=========================================="
exec docker compose up -d --build --pull always "${ALL_SERVICES[@]}"