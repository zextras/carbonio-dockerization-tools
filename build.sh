#!/bin/bash

set -e

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'none')}"
DATE="${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

echo "🔨 Building Carbonio Docker CLI..."
echo "   Version: $VERSION"
echo "   Commit:  $COMMIT"
echo "   Date:    $DATE"
echo ""

rm -f carbonio-docker-cli

go build \
  -ldflags="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.date=$DATE'" \
  -o carbonio-docker-cli \
  ./cmd/carbonio-docker-cli

echo "✅ Build completed: carbonio-docker-cli"
echo ""
echo "Usage:"
echo "  ./carbonio-docker-cli"
echo "  ./carbonio-docker-cli --config-file <file>"
echo "  ./carbonio-docker-cli --config-file <file> --headless"
echo "  ./carbonio-docker-cli --config-file <file> --with-clean-database"
echo "  ./carbonio-docker-cli --save-logs"
echo ""
echo "Flags:"
echo "  --config-file <file>  Load configuration from YAML file"
echo "  --headless            Run without TUI (for scripts/automation)"
echo "  --with-clean-database Start with a fresh database (removes all data)"
echo "  --save-logs           Enable logging to carbonio-docker-cli.log"