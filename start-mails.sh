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

# On macOS, append the override pinning the images that still have no working
# arm64 variant. Note: the default service set starts fine without it (none of
# its dependencies are amd64-only); it matters only when extra services are
# requested (e.g. carbonio-tasks, ws-collaboration, files/docs).
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

docker compose build --pull --with-dependencies "${ALL_SERVICES[@]}"
docker compose pull --ignore-pull-failures --include-deps "${ALL_SERVICES[@]}"
exec docker compose up -d "${ALL_SERVICES[@]}"
