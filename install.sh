#!/bin/bash
set -euo pipefail

REPO="joaobarroca93/hactl"
BIN="hactl"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "error: unsupported operating system: ${OS}" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "error: unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

# Fetch latest release tag from GitHub API
echo "Fetching latest release..."
LATEST_TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [ -z "${LATEST_TAG}" ]; then
  echo "error: could not determine latest release tag" >&2
  exit 1
fi

echo "Latest release: ${LATEST_TAG}"

# Build archive name and download URL
ARCHIVE="${BIN}_${LATEST_TAG#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE}"

# Download and extract
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${URL}..."
curl -fsSL "${URL}" -o "${TMP_DIR}/${ARCHIVE}"

echo "Extracting..."
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

# Install binary
echo "Installing ${BIN} to ${INSTALL_DIR}..."
if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP_DIR}/${BIN}" "${INSTALL_DIR}/${BIN}"
else
  sudo mv "${TMP_DIR}/${BIN}" "${INSTALL_DIR}/${BIN}"
fi

chmod +x "${INSTALL_DIR}/${BIN}"

echo "hactl ${LATEST_TAG} installed successfully to ${INSTALL_DIR}/${BIN}"
