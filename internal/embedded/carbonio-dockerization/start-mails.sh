#!/bin/bash

set -e
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

# On macOS, append the platform override (Zextras repo ships no arm64 build)
if [[ "$(uname)" == "Darwin" ]]; then
    COMPOSE_FILE="${COMPOSE_FILE}:docker-compose.macos.override.yaml"
fi

# Remaining arguments are additional services
ALL_SERVICES+=("$@")

echo "=========================================="
echo "=========================================="
echo "Starting with compose file: $COMPOSE_FILE"
echo "Services: ${ALL_SERVICES[*]}"
echo "=========================================="
echo "=========================================="

# Pulls latest images in the built container
docker compose build --pull
# Pulls latest images in the compose files (ignores failures for local-only images)
docker compose pull --ignore-pull-failures
exec docker compose up -d "${ALL_SERVICES[@]}"