#!/bin/sh
set -e
TOKEN_DIR="/etc/carbonio/${SERVICE_NAME}/service-discover"
mkdir -p "$TOKEN_DIR"
service-discover > "$TOKEN_DIR/token"
echo "[setup] Token written to $TOKEN_DIR/token"
