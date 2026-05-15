#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"

if ! command -v call-recorder &>/dev/null; then
  echo "[rec] call-recorder non trovato in PATH, compilo dal sorgente..."
  (cd "$DIR" && go build -o "$DIR/call-recorder" .)
  export PATH="$DIR:$PATH"
fi

exec call-recorder record -output "$DIR/recordings" "$@"
