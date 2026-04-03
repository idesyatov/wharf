#!/bin/bash
# Wharf — install script
# Usage: curl -sL https://raw.githubusercontent.com/idesyatov/wharf/master/scripts/install.sh | bash

set -e

REPO="idesyatov/wharf"
if [ -n "$DIR" ]; then
    INSTALL_DIR="$DIR"
elif [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect ARCH
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Installing wharf ${VERSION} (${OS}/${ARCH})..."

# Download and extract
URL="https://github.com/${REPO}/releases/download/${VERSION}/wharf-${VERSION}-${OS}-${ARCH}.tar.gz"
TMPDIR=$(mktemp -d)
curl -sL "$URL" -o "${TMPDIR}/wharf.tar.gz" || { echo "Download failed"; exit 1; }
tar xzf "${TMPDIR}/wharf.tar.gz" -C "${TMPDIR}" || { echo "Extract failed"; exit 1; }

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/wharf" "${INSTALL_DIR}/wharf"
chmod +x "${INSTALL_DIR}/wharf"
rm -rf "$TMPDIR"

echo "Installed wharf ${VERSION} to ${INSTALL_DIR}/wharf"

# Check if INSTALL_DIR is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
