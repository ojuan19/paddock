#!/usr/bin/env bash
set -euo pipefail

REPO="ojuan19/paddock"

# OS detect
case "$(uname -s)" in
  Darwin) OS=Darwin ;;
  Linux)  OS=Linux ;;
  *) echo "paddock install: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

# Arch detect
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "paddock install: unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

# Latest version (no jq dep)
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "paddock install: could not determine latest version" >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/paddock_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "paddock install: downloading ${VERSION} for ${OS}/${ARCH}..."
ARCHIVE="paddock_${OS}_${ARCH}.tar.gz"
curl -fsSL -o "$TMP/$ARCHIVE" "$URL"
curl -fsSL -o "$TMP/checksums.txt" \
  "https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  SHA_CMD="sha256sum"
else
  SHA_CMD="shasum -a 256"
fi
( cd "$TMP" && grep " ${ARCHIVE}$" checksums.txt | ${SHA_CMD} -c - ) \
  || { echo "paddock install: checksum verification failed" >&2; exit 1; }

tar -xz -C "$TMP" -f "$TMP/$ARCHIVE"

# Install target
SUDO=""
if [ -w /usr/local/bin ] 2>/dev/null; then
  DEST=/usr/local/bin
elif command -v sudo >/dev/null 2>&1 && [ -d /usr/local/bin ]; then
  SUDO=sudo
  DEST=/usr/local/bin
else
  mkdir -p "$HOME/.local/bin"
  DEST="$HOME/.local/bin"
fi

${SUDO} install -m 0755 "$TMP/paddock" "$DEST/paddock"
echo "✓ installed paddock $VERSION to $DEST/paddock"

case ":$PATH:" in
  *":$DEST:"*) ;;
  *) echo "⚠  $DEST is not in PATH — add it to your shell rc" ;;
esac

echo ""
echo "Next steps:"
echo "  1. Enable auto-switch:   eval \"\$(paddock shell-init zsh)\" >> ~/.zshrc"
echo "  2. Initialize:           paddock init"
echo "  3. Note: blank profiles created with 'paddock add' require '/login' on first use"
