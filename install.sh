#!/bin/sh
# tallyburn installer
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/joshsgoldstein/tallyburn/main/install.sh | sh

set -e

REPO="joshsgoldstein/tallyburn"
BINARY_NAME="tallyburn"

echo "Installing ${BINARY_NAME}..."

# prefer pipx, fall back to pip
if command -v pipx >/dev/null 2>&1; then
  pipx install "git+https://github.com/${REPO}.git"
elif command -v pip3 >/dev/null 2>&1; then
  pip3 install --user "git+https://github.com/${REPO}.git"
elif command -v pip >/dev/null 2>&1; then
  pip install --user "git+https://github.com/${REPO}.git"
else
  echo "Error: pip or pipx is required. Install Python first: https://python.org"
  exit 1
fi

echo ""
echo "${BINARY_NAME} installed. Run '${BINARY_NAME} --help' to get started."
echo ""
echo "Quick start:"
echo "  tallyburn --all          # all projects by client"
echo "  tallyburn solutionsguy   # drill into a folder"
echo "  tallyburn sessions       # session-level detail for current directory"
