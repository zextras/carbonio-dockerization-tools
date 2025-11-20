#!/bin/bash

# Script di build per Carbonio Docker CLI
# Compila la CLI con tutte le dipendenze embedded

set -e

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'none')}"
DATE="${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

echo "🔨 Building Carbonio Docker CLI..."
echo "   Version: $VERSION"
echo "   Commit:  $COMMIT"
echo "   Date:    $DATE"
echo ""

# Pulisci build precedenti
rm -f carbonio-docker-cli

# Compila con ldflags per versioning
go build \
  -ldflags="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.date=$DATE'" \
  -o carbonio-docker-cli \
  ./cmd/carbonio-docker-cli

echo "✅ Build completed: carbonio-docker-cli"
echo ""
echo "Run with:"
echo "  ./carbonio-docker-cli"
echo ""
echo "Or with config:"
echo "  ./carbonio-docker-cli --config carbonio-config.yaml"