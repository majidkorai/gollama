#!/bin/sh
# gollama — one-line install
#   curl -fsSL https://raw.githubusercontent.com/majidkorai/gollama/main/install.sh | sh
#   VERSION=v0.2.7-rc1 curl -fsSL https://raw.githubusercontent.com/majidkorai/gollama/main/install.sh | sh
set -e

REPO="majidkorai/gollama"
VERSION="${VERSION:-latest}"

# ── Detect platform ────────────────────────────────────────────────────
RAW_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
RAW_ARCH=$(uname -m)

case "$RAW_ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $RAW_ARCH"; exit 1 ;;
esac

EXE=""
case "$RAW_OS" in
    linux)          OS="linux"   ;;
    darwin)         OS="darwin"  ;;
    mingw*|msys*|cygwin*) OS="windows"; EXE=".exe" ;;
    *) echo "Unsupported OS: $RAW_OS"; exit 1 ;;
esac

# ── Install location ───────────────────────────────────────────────────
case "$OS" in
    windows)
        # Install next to the script or in USERPROFILE
        INSTALL_DIR="${GOLLAMA_HOME:-$HOME/gollama}"
        mkdir -p "$INSTALL_DIR"
        FINAL="$INSTALL_DIR/gollama$EXE"
        CMD_HINT="Add \"$INSTALL_DIR\" to your PATH or run: .\"$FINAL\""
        ;;
    *)
        INSTALL_DIR="/usr/local/bin"
        FINAL="$INSTALL_DIR/gollama"
        CMD_HINT="Run 'gollama' to start"
        ;;
esac

# ── Try pre-built binary ──────────────────────────────────────────────
echo "gollama — installing for $OS/$ARCH"

if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/gollama-$OS-$ARCH$EXE"
else
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/gollama-$OS-$ARCH$EXE"
fi
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if curl -sfL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
    chmod +x "$TMP_FILE" 2>/dev/null || true

    if [ "$OS" = "windows" ]; then
        mv "$TMP_FILE" "$FINAL"
    elif [ "$(id -u)" -eq 0 ]; then
        mv "$TMP_FILE" "$FINAL"
    else
        echo "Installing to $FINAL (may ask for sudo)..."
        sudo mv "$TMP_FILE" "$FINAL"
    fi

    echo "Installed $FINAL"
    echo "$CMD_HINT"
    exit 0
fi

# ── Fallback: build from source ───────────────────────────────────────
echo "No pre-built binary available for $OS/$ARCH."
echo "Building from source (requires Go)..."

if ! command -v go >/dev/null 2>&1; then
    echo
    echo "Go is required to build from source."
    echo "Install Go first: https://go.dev/doc/install"
    exit 1
fi

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

echo "Cloning $REPO${VERSION:+ (tag: $VERSION)}..."
git clone --depth 1 "https://github.com/$REPO.git" "$BUILD_DIR"
if [ "$VERSION" != "latest" ]; then
    cd "$BUILD_DIR" && git fetch --depth 1 origin "refs/tags/$VERSION" && git checkout -q "$VERSION" && cd - >/dev/null
fi
cd "$BUILD_DIR"

echo "Building..."
go build -o "gollama$EXE" .

if [ "$OS" = "windows" ]; then
    mv "gollama$EXE" "$FINAL"
elif [ "$(id -u)" -eq 0 ]; then
    mv "gollama$EXE" "$FINAL"
else
    echo "Installing to $FINAL (may ask for sudo)..."
    sudo mv "gollama$EXE" "$FINAL"
fi

echo "Installed $FINAL"
echo "$CMD_HINT"
