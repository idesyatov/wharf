#!/bin/bash
# Wharf — install script
# Usage: curl -sL https://raw.githubusercontent.com/idesyatov/wharf/master/scripts/install.sh | bash

set -e

REPO="idesyatov/wharf"
CURL="curl -fsSL --retry 2 --retry-delay 1 --connect-timeout 15 --max-time 180"
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
VERSION=$(curl -sI --retry 2 --connect-timeout 15 "https://github.com/${REPO}/releases/latest" | grep -i "location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Installing wharf ${VERSION} (${OS}/${ARCH})..."

# Download
ARCHIVE="wharf-${VERSION}-${OS}-${ARCH}.tar.gz"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
$CURL "${BASE}/${ARCHIVE}" -o "${TMPDIR}/${ARCHIVE}" || { echo "Download failed"; exit 1; }

# Verify checksum
if $CURL "${BASE}/checksums.txt" -o "${TMPDIR}/checksums.txt" 2>/dev/null; then
    EXPECTED=$(awk -v f="$ARCHIVE" '$2 == f {print $1}' "${TMPDIR}/checksums.txt")
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL=$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
    fi
    if [ -n "$EXPECTED" ] && [ -n "$ACTUAL" ]; then
        if [ "$EXPECTED" != "$ACTUAL" ]; then
            echo "Checksum mismatch! Aborting."
            exit 1
        fi
        echo "Checksum verified."
    else
        echo "Warning: could not verify checksum (no sha256 tool or entry missing)"
    fi
else
    echo "Warning: checksums.txt unavailable, skipping verification"
fi

# Extract
tar xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}" || { echo "Extract failed"; exit 1; }

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/wharf" "${INSTALL_DIR}/wharf"
chmod +x "${INSTALL_DIR}/wharf"

echo "Installed wharf ${VERSION} to ${INSTALL_DIR}/wharf"

# Check if INSTALL_DIR is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
