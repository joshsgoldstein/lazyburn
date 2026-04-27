#!/bin/sh
# lazyburn installer
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/joshsgoldstein/lazyburn/main/install.sh | sh

set -e

REPO="joshsgoldstein/lazyburn"
BINARY="lazyburn"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Install manually: go install github.com/${REPO}@latest"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)
    echo "Unsupported OS: $OS"
    echo "On Windows, download lazyburn_windows_amd64.exe from:"
    echo "  https://github.com/${REPO}/releases/latest"
    exit 1
    ;;
esac

ASSET="${BINARY}_${OS}_${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "Downloading ${BINARY} (${OS}/${ARCH})..."

TMP=$(mktemp)
if command -v curl >/dev/null 2>&1; then
  curl -sSfL "$URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP" "$URL"
else
  echo "Error: curl or wget is required"
  exit 1
fi

chmod +x "$TMP"

# Try /usr/local/bin first; fall back to ~/bin if no write access
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
  echo "${BINARY} installed to ${INSTALL_DIR}/${BINARY}"
else
  mkdir -p "$HOME/bin"
  mv "$TMP" "$HOME/bin/${BINARY}"
  echo "${BINARY} installed to $HOME/bin/${BINARY}"
  echo ""
  echo "Make sure \$HOME/bin is in your PATH:"
  echo "  export PATH=\"\$HOME/bin:\$PATH\""
fi

echo ""
echo "Quick start:"
echo "  lazyburn --all            # all projects, grouped by folder"
echo "  lazyburn --path acme      # drill into a specific folder"
echo "  lazyburn sessions         # per-session breakdown"
echo "  lazyburn --help           # full usage"
