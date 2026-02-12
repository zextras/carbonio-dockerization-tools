#!/bin/bash

set -e

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'none')}"
DATE="${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

LDFLAGS="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.date=$DATE'"

ARTIFACTS_DIR="artifacts"
APP_NAME="Carbonio Dockerization GUI"
APP_ID="com.zextras.carbonio-dockerization-gui"
ICON="carbonio-docker-logo.png"

usage() {
  echo "Usage: ./build.sh [command] [target]"
  echo ""
  echo "Commands:"
  echo "  build [target]  Build packages for a target OS (default: current OS)"
  echo "  install         Build + install to system (requires sudo)"
  echo "  clean           Remove build artifacts"
  echo ""
  echo "Targets:"
  echo "  linux           .deb and .rpm packages (requires nfpm)"
  echo "  macos           .app bundle + CLI binary (requires fyne CLI)"
  echo "  windows         .exe installer + CLI binary (requires fyne CLI)"
  echo "  all             Build for all targets"
  echo ""
  echo "Environment variables:"
  echo "  VERSION   Package version (default: dev)"
  echo "  GOARCH    Target architecture (default: auto-detected)"
  echo ""
  echo "Examples:"
  echo "  ./build.sh build              # Build for current OS"
  echo "  ./build.sh build linux        # Build .deb and .rpm"
  echo "  ./build.sh build macos        # Build macOS .app bundle"
  echo "  ./build.sh build all          # Build for all platforms"
  echo "  ./build.sh install            # Build and install locally"
}

print_header() {
  echo "Building Carbonio dockerization CLI & GUI..."
  echo "   Version: $VERSION"
  echo "   Commit:  $COMMIT"
  echo "   Date:    $DATE"
  echo "   Arch:    $GOARCH"
  echo ""
}

require_tool() {
  local tool="$1"
  local install_hint="$2"
  if ! command -v "$tool" &> /dev/null; then
    echo "Error: $tool not found. Install it with:"
    echo "  $install_hint"
    exit 1
  fi
}

compile_cli() {
  local goos="${1:-}"
  local suffix="${2:-}"

  echo "Compiling carbonio-dockerization-cli${suffix}..."
  if [ -n "$goos" ]; then
    CGO_ENABLED=0 GOOS="$goos" go build \
      -ldflags="$LDFLAGS" \
      -o "$ARTIFACTS_DIR/carbonio-dockerization-cli${suffix}" \
      ./cmd/carbonio-docker-cli
  else
    go build \
      -ldflags="$LDFLAGS" \
      -o "$ARTIFACTS_DIR/carbonio-dockerization-cli${suffix}" \
      ./cmd/carbonio-docker-cli
  fi
}

compile_gui() {
  echo "Compiling carbonio-dockerization-gui..."
  go build \
    -ldflags="$LDFLAGS" \
    -o "$ARTIFACTS_DIR/carbonio-dockerization-gui" \
    ./cmd/carbonio-docker-gui
}

# --- Linux: .deb + .rpm via nfpm ---

do_build_linux() {
  print_header
  mkdir -p "$ARTIFACTS_DIR"

  compile_cli
  compile_gui

  require_tool nfpm "go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"

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

  rm -f "$ARTIFACTS_DIR/carbonio-dockerization-cli" "$ARTIFACTS_DIR/carbonio-dockerization-gui"

  echo ""
  echo "Linux packages:"
  ls -lh "$ARTIFACTS_DIR/"*.deb "$ARTIFACTS_DIR/"*.rpm 2>/dev/null
}

# --- macOS: .app bundle via fyne package ---

do_build_macos() {
  print_header
  mkdir -p "$ARTIFACTS_DIR"

  require_tool fyne "go install fyne.io/fyne/v2/cmd/fyne@latest"

  compile_cli darwin ""
  mv "$ARTIFACTS_DIR/carbonio-dockerization-cli" "$ARTIFACTS_DIR/carbonio-dockerization-cli-darwin-${GOARCH}"

  echo "Creating macOS .app bundle..."
  fyne package \
    -os darwin \
    -icon "$ICON" \
    -appID "$APP_ID" \
    -name "$APP_NAME" \
    -src ./cmd/carbonio-docker-gui

  mv "Carbonio Dockerization GUI.app" "$ARTIFACTS_DIR/" 2>/dev/null || \
  mv *.app "$ARTIFACTS_DIR/" 2>/dev/null || true

  echo ""
  echo "macOS artifacts:"
  ls -lhd "$ARTIFACTS_DIR/"*.app "$ARTIFACTS_DIR/"*darwin* 2>/dev/null
}

# --- Windows: .exe via fyne package ---

do_build_windows() {
  print_header
  mkdir -p "$ARTIFACTS_DIR"

  require_tool fyne "go install fyne.io/fyne/v2/cmd/fyne@latest"

  compile_cli windows .exe
  mv "$ARTIFACTS_DIR/carbonio-dockerization-cli.exe" "$ARTIFACTS_DIR/carbonio-dockerization-cli-windows-${GOARCH}.exe"

  echo "Creating Windows .exe..."
  fyne package \
    -os windows \
    -icon "$ICON" \
    -appID "$APP_ID" \
    -name "$APP_NAME" \
    -src ./cmd/carbonio-docker-gui

  mv *.exe "$ARTIFACTS_DIR/" 2>/dev/null || true

  echo ""
  echo "Windows artifacts:"
  ls -lh "$ARTIFACTS_DIR/"*.exe 2>/dev/null
}

# --- Install locally (current OS) ---

do_install() {
  print_header
  mkdir -p "$ARTIFACTS_DIR"

  compile_cli
  compile_gui

  echo ""
  echo "Installing..."

  OS="$(uname -s)"

  sudo install -m 755 "$ARTIFACTS_DIR/carbonio-dockerization-cli" /usr/local/bin/carbonio-dockerization-cli
  sudo install -m 755 "$ARTIFACTS_DIR/carbonio-dockerization-gui" /usr/local/bin/carbonio-dockerization-gui

  echo "Installed:"
  echo "  /usr/local/bin/carbonio-dockerization-cli"
  echo "  /usr/local/bin/carbonio-dockerization-gui"

  if [ "$OS" = "Linux" ]; then
    sudo install -m 644 "$ICON" /usr/share/pixmaps/carbonio-dockerization-gui.png
    sudo install -m 644 packaging/carbonio-dockerization-gui.desktop /usr/share/applications/carbonio-dockerization-gui.desktop
    echo "  /usr/share/applications/carbonio-dockerization-gui.desktop"
    echo "  /usr/share/pixmaps/carbonio-dockerization-gui.png"
  elif [ "$OS" = "Darwin" ]; then
    echo ""
    echo "Note: On macOS, launch the GUI directly from /usr/local/bin/carbonio-dockerization-gui"
    echo "      For a .app bundle, use: ./build.sh build macos"
  fi

  rm -f "$ARTIFACTS_DIR/carbonio-dockerization-cli" "$ARTIFACTS_DIR/carbonio-dockerization-gui"
  rmdir "$ARTIFACTS_DIR" 2>/dev/null || true
}

do_clean() {
  echo "Cleaning build artifacts..."
  rm -rf "$ARTIFACTS_DIR/"
  echo "Done."
}

# --- Detect current OS for default build target ---

detect_os() {
  case "$(uname -s)" in
    Linux)  echo "linux" ;;
    Darwin) echo "macos" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)      echo "linux" ;;
  esac
}

# --- Main ---

COMMAND="${1:-build}"
TARGET="${2:-}"

case "$COMMAND" in
  build)
    if [ -z "$TARGET" ]; then
      TARGET="$(detect_os)"
    fi
    case "$TARGET" in
      linux)           do_build_linux ;;
      macos|darwin)    do_build_macos ;;
      windows|win)     do_build_windows ;;
      all)
        do_build_linux
        echo ""
        echo "========================================="
        echo ""
        do_build_macos
        echo ""
        echo "========================================="
        echo ""
        do_build_windows
        ;;
      *) echo "Unknown target: $TARGET"; usage; exit 1 ;;
    esac
    ;;
  install) do_install ;;
  clean)   do_clean ;;
  -h|--help|help) usage ;;
  *) echo "Unknown command: $COMMAND"; usage; exit 1 ;;
esac
