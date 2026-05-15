#!/usr/bin/env bash
set -e

REPO_URL="https://gitlab.simultech.it/simultech/call-recorder"
BINARY="call-recorder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── Detect OS and arch ────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)         ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *) echo "Architettura non supportata: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)  EXT="" ;;
  darwin) EXT="" ;;
  *)      echo "OS non supportato: $OS. Su Windows usa install.ps1"; exit 1 ;;
esac

# ── Choose variant ────────────────────────────────────────────────────────────

TRAY=true
if [[ "$*" == *--no-tray* ]]; then
  TRAY=false
fi

if [ "$TRAY" = true ] && [ "$OS" = "darwin" ]; then
  # macOS tray is native, no extra deps
  ASSET="${BINARY}-${OS}-${ARCH}"
elif [ "$TRAY" = true ]; then
  ASSET="${BINARY}-${OS}-${ARCH}"
else
  ASSET="${BINARY}-${OS}-${ARCH}-cli"
fi

# ── Find latest release ───────────────────────────────────────────────────────

echo "Recupero ultima release da ${REPO_URL}..."
LATEST=$(curl -fsSL "${REPO_URL}/-/releases/permalink/latest" \
  | grep -oP 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)

if [ -z "$LATEST" ]; then
  echo "Impossibile recuperare la versione. Specifica manualmente:"
  echo "  RELEASE=v1.0.0 bash install.sh"
  LATEST="${RELEASE:-}"
fi

if [ -z "$LATEST" ]; then
  exit 1
fi

DOWNLOAD_URL="${REPO_URL}/-/releases/${LATEST}/downloads/${ASSET}"

# ── Download ──────────────────────────────────────────────────────────────────

TMP=$(mktemp)
echo "Scaricando ${ASSET} (${LATEST})..."
curl -fL "$DOWNLOAD_URL" -o "$TMP"
chmod +x "$TMP"

# ── Install system deps (Linux tray) ─────────────────────────────────────────

if [ "$OS" = "linux" ] && [ "$TRAY" = true ]; then
  echo "Installazione dipendenze di sistema per la tray icon..."
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

# ── Install binary ────────────────────────────────────────────────────────────

echo "Installando in ${INSTALL_DIR}/${BINARY}..."
sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"

echo ""
echo "✓ Installato: $(${INSTALL_DIR}/${BINARY} --help 2>&1 | head -1 || echo call-recorder)"
echo ""
echo "Utilizzo:"
echo "  call-recorder list"
echo "  call-recorder record -lang it"
echo "  call-recorder tray -lang it"
