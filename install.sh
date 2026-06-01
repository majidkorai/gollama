#!/bin/sh
# gollama — one-line install
#   curl -fsSL https://raw.githubusercontent.com/majidkorai/gollama/main/install.sh | sh
set -e

REPO="majidkorai/gollama"
BINARY="gollama"
INSTALL_DIR="/usr/local/bin"

# ── Detect platform ────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# ── Try pre-built binary ──────────────────────────────────────────────
echo "gollama — installing for $OS/$ARCH"

DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/gollama-$OS-$ARCH"
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if curl -sfL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
    chmod +x "$TMP_FILE"

    if [ "$(id -u)" -eq 0 ]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
    else
        echo "Installing to $INSTALL_DIR/$BINARY (may ask for sudo)..."
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
    fi

    echo "Installed $INSTALL_DIR/$BINARY"
    echo "Run 'gollama' to start the setup wizard."
    exit 0
fi

# ── Fallback: build from source ───────────────────────────────────────
echo "No pre-built binary available for $OS/$ARCH."
echo "Building from source (requires Go)..."

if ! command -v go >/dev/null 2>&1; then
    echo
    echo "Go is required to build from source."
    echo "Install Go first: https://go.dev/doc/install"
    echo
    echo "Or use your package manager:"
    echo "  macOS: brew install go"
    echo "  Ubuntu: sudo apt install golang-go"
    echo "  Fedora: sudo dnf install golang"
    exit 1
fi

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

echo "Cloning $REPO..."
git clone --depth 1 "https://github.com/$REPO.git" "$BUILD_DIR"
cd "$BUILD_DIR"

echo "Building..."
go build -o "$BINARY" .

if [ "$(id -u)" -eq 0 ]; then
    mv "$BINARY" "$INSTALL_DIR/$BINARY"
else
    echo "Installing to $INSTALL_DIR/$BINARY (may ask for sudo)..."
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Installed $INSTALL_DIR/$BINARY"
echo "Run 'gollama' to start the setup wizard."
