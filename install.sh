#!/usr/bin/env bash
set -e

GITLAB="https://gitlab.simultech.it"
PROJECT="simultech/call-recorder"
BINARY="call-recorder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── Auth ──────────────────────────────────────────────────────────────────────

TOKEN="${GITLAB_TOKEN:-}"

curl_auth() {
  if [ -n "$TOKEN" ]; then
    curl -fsSL -H "PRIVATE-TOKEN: $TOKEN" "$@"
  else
    curl -fsSL "$@"
  fi
}

# ── Detect OS and arch ────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Architettura non supportata: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "OS non supportato: $OS — su Windows usa install.ps1"; exit 1 ;;
esac

# ── Choose variant ────────────────────────────────────────────────────────────

if [[ "$*" == *--no-tray* ]]; then
  ASSET="${BINARY}-${OS}-${ARCH}-cli"
else
  ASSET="${BINARY}-${OS}-${ARCH}"
fi

# ── Find latest release via API ───────────────────────────────────────────────

API="${GITLAB}/api/v4/projects/$(python3 -c "import urllib.parse; print(urllib.parse.quote('${PROJECT}', safe=''))")/releases"
LATEST=$(curl_auth "$API" 2>/dev/null | grep -oP '"tag_name":"\K[^"]+' | head -1)

if [ -z "$LATEST" ]; then
  echo ""
  echo "Impossibile recuperare la lista release."
  if [ -z "$TOKEN" ]; then
    echo ""
    echo "Il repository è privato. Genera un Personal Access Token su:"
    echo "  ${GITLAB}/-/user_settings/personal_access_tokens"
    echo "(scope: read_api)"
    echo ""
    echo "Poi esegui:"
    echo "  GITLAB_TOKEN=<token> bash install.sh"
  fi
  exit 1
fi

echo "Ultima release: $LATEST"

# ── Download binary ───────────────────────────────────────────────────────────

PKG_URL="${GITLAB}/api/v4/projects/$(python3 -c "import urllib.parse; print(urllib.parse.quote('${PROJECT}', safe=''))")/packages/generic/${BINARY}/${LATEST}/${ASSET}"

TMP=$(mktemp)
echo "Scaricando ${ASSET}..."
if ! curl_auth "$PKG_URL" -o "$TMP" 2>/dev/null || [ ! -s "$TMP" ]; then
  echo "Download fallito. Verifica che il token abbia lo scope 'read_api'."
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
