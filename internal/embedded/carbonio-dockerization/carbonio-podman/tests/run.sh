#!/usr/bin/env bash
# Run the Hurl smoke tests against the running infrastructure.
set -euo pipefail
cd "$(dirname "$0")"

HOST="${HOST:-localhost}"
DOMAIN="${DOMAIN:-carbonio.localhost}"

podman run -it --rm \
  --network=host \
  -v "$(pwd)/hurl:/tests:ro" \
  ghcr.io/orange-opensource/hurl:latest \
  -k --test --variable "host=$HOST" --variable "domain=$DOMAIN" --glob "/tests/**/*.hurl"
