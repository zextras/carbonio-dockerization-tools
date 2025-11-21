#!/bin/bash

export COMPOSE_FILE="docker-compose.yaml:docker-compose-advanced.yaml"
exec docker compose down