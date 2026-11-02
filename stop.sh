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
#export COMPOSE_FILE="docker-compose.yaml:docker-compose-advanced.yaml:monitoring.yaml"
echo "Stopping services defined in $COMPOSE_FILE"
exec docker compose down