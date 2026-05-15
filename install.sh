#!/usr/bin/env bash
set -e

GITHUB="https://github.com"
REPO="Simul-Tech/call-recorder"
BINARY="call-recorder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── Detect OS and arch ────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Architettura non supportata: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)  OS_TAG="linux" ;;
  darwin) OS_TAG="macos" ;;
  *) echo "OS non supportato: $OS — su Windows usa install.ps1"; exit 1 ;;
esac

if [ "$OS_TAG" = "macos" ] && [ "$ARCH" = "amd64" ]; then
  echo "Binari precompilati per macOS Intel non disponibili."
  echo "Compila da sorgente: git clone https://github.com/${REPO} && make build"
  exit 1
fi

# ── Choose variant ────────────────────────────────────────────────────────────

if [[ "$*" == *--no-tray* ]]; then
  ASSET="${BINARY}-${OS_TAG}-${ARCH}-cli"
else
  ASSET="${BINARY}-${OS_TAG}-${ARCH}"
fi

# ── Find latest release ───────────────────────────────────────────────────────

RESPONSE=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
LATEST=$(echo "$RESPONSE" | grep -oP '"tag_name":\s*"\K[^"]+' || true)

if [ -z "$LATEST" ]; then
  echo "Impossibile recuperare l'ultima release da GitHub."
  echo "Verifica che esista almeno una release su: https://github.com/${REPO}/releases"
  exit 1
fi

echo "Ultima release: $LATEST"

# ── Download binary ───────────────────────────────────────────────────────────

DOWNLOAD_URL="${GITHUB}/${REPO}/releases/download/${LATEST}/${ASSET}"

TMP=$(mktemp)
echo "Scaricando ${ASSET}..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP" 2>/dev/null || [ ! -s "$TMP" ]; then
  echo "Download fallito."
  rm -f "$TMP"
  exit 1
fi
chmod +x "$TMP"

# ── Install system deps (Linux tray) ─────────────────────────────────────────

if [ "$OS" = "linux" ] && [[ "$*" != *--no-tray* ]]; then
  echo "Installazione dipendenze tray..."
  if command -v pacman &>/dev/null; then
    sudo pacman -S --noconfirm --needed libappindicator-gtk3
  elif command -v apt-get &>/dev/null; then
    sudo apt-get install -y libappindicator3-1
  elif command -v dnf &>/dev/null; then
    sudo dnf install -y libappindicator-gtk3
  else
    echo "⚠  Installa manualmente: libappindicator-gtk3"
  fi
fi

# ── Install ───────────────────────────────────────────────────────────────────

echo "Installando in ${INSTALL_DIR}/${BINARY}..."
sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"

echo ""
echo "✓ call-recorder ${LATEST} installato"
echo ""
echo "Utilizzo:"
echo "  call-recorder list"
echo "  call-recorder record -lang it"
echo "  call-recorder tray -lang it"
