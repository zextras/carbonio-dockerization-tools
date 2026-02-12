#!/bin/bash

set -e

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'none')}"
DATE="${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

LDFLAGS="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.date=$DATE'"

ARTIFACTS_DIR="artifacts"

usage() {
  echo "Usage: ./build.sh [command]"
  echo ""
  echo "Commands:"
  echo "  build     Build .deb and .rpm packages (default)"
  echo "  install   Build + install to system (requires sudo)"
  echo "  clean     Remove build artifacts"
  echo ""
  echo "Environment variables:"
  echo "  VERSION   Package version (default: dev)"
  echo "  GOARCH    Target architecture (default: auto-detected)"
  echo ""
  echo "Output: $ARTIFACTS_DIR/*.deb, $ARTIFACTS_DIR/*.rpm"
}

do_compile() {
  echo "Building Carbonio dockerization CLI & GUI..."
  echo "   Version: $VERSION"
  echo "   Commit:  $COMMIT"
  echo "   Date:    $DATE"
  echo "   Arch:    $GOARCH"
  echo ""

  mkdir -p "$ARTIFACTS_DIR"

  # CLI (headless, no GUI dependencies)
  echo "Compiling carbonio-dockerization-cli..."
  go build \
    -ldflags="$LDFLAGS" \
    -o "$ARTIFACTS_DIR/carbonio-dockerization-cli" \
    ./cmd/carbonio-docker-cli

  # GUI (requires: gcc, libgl1-mesa-dev, xorg-dev, pkg-config on Linux)
  echo "Compiling carbonio-dockerization-gui..."
  go build \
    -ldflags="$LDFLAGS" \
    -o "$ARTIFACTS_DIR/carbonio-dockerization-gui" \
    ./cmd/carbonio-docker-gui

  echo "Compilation done."
}

do_build() {
  do_compile

  if ! command -v nfpm &> /dev/null; then
    echo ""
    echo "Error: nfpm not found. Install it with:"
    echo "  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
    exit 1
  fi

  echo ""

  echo "Creating .deb package..."
  VERSION="$VERSION" GOARCH="$GOARCH" nfpm package \
    --config nfpm.yaml \
    --packager deb \
    --target "$ARTIFACTS_DIR/"

  echo "Creating .rpm package..."
  VERSION="$VERSION" GOARCH="$GOARCH" nfpm package \
    --config nfpm.yaml \
    --packager rpm \
    --target "$ARTIFACTS_DIR/"

  # Remove intermediate binaries, keep only packages
  rm -f "$ARTIFACTS_DIR/carbonio-dockerization-cli" "$ARTIFACTS_DIR/carbonio-dockerization-gui"

  echo ""
  echo "Packages:"
  ls -lh "$ARTIFACTS_DIR/"
}

do_install() {
  do_compile

  echo ""
  echo "Installing..."

  sudo install -m 755 "$ARTIFACTS_DIR/carbonio-dockerization-cli" /usr/local/bin/carbonio-dockerization-cli
  sudo install -m 755 "$ARTIFACTS_DIR/carbonio-dockerization-gui" /usr/local/bin/carbonio-dockerization-gui
  sudo install -m 644 carbonio-docker-logo.png /usr/share/pixmaps/carbonio-dockerization-gui.png
  sudo install -m 644 packaging/carbonio-dockerization-gui.desktop /usr/share/applications/carbonio-dockerization-gui.desktop

  # Remove intermediate binaries
  rm -f "$ARTIFACTS_DIR/carbonio-dockerization-cli" "$ARTIFACTS_DIR/carbonio-dockerization-gui"
  rmdir "$ARTIFACTS_DIR" 2>/dev/null || true

  echo "Installed:"
  echo "  /usr/local/bin/carbonio-dockerization-cli"
  echo "  /usr/local/bin/carbonio-dockerization-gui"
  echo "  /usr/share/applications/carbonio-dockerization-gui.desktop"
  echo "  /usr/share/pixmaps/carbonio-dockerization-gui.png"
}

do_clean() {
  echo "Cleaning build artifacts..."
  rm -rf "$ARTIFACTS_DIR/"
  echo "Done."
}

COMMAND="${1:-build}"

case "$COMMAND" in
  build)   do_build ;;
  install) do_install ;;
  clean)   do_clean ;;
  -h|--help|help) usage ;;
  *) echo "Unknown command: $COMMAND"; usage; exit 1 ;;
esac
