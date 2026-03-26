#!/bin/bash
# Initialize Consul KV store with application configuration.
# Run after consul-server is healthy.

set -euo pipefail
CONSUL_HTTP_ADDR="${CONSUL_HTTP_ADDR:-http://localhost:8500}"

DIR="$(dirname "$0")"
for script in "$DIR"/*.sh; do
  [ "$(basename "$script")" = "init.sh" ] && continue
  echo "Running $(basename "$script")..."
  bash "$script"
done

echo "Consul KV initialization complete."
