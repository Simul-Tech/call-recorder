# call-recorder

Registra simultaneamente microfono e audio di sistema durante le chiamate, salva un file WAV mixato e lo trascrive automaticamente.

Usa **[miniaudio](https://miniaud.io/)** tramite `gen2brain/malgo` — libreria C embedded nel pacchetto, zero dipendenze di sistema per la sola registrazione.

## Installazione

### Script automatico (consigliato)

**Linux / macOS:**
```bash
# Con tray icon (richiede libappindicator su Linux)
bash <(curl -fsSL https://gitlab.simultech.it/simultech/call-recorder/-/raw/main/install.sh)

# CLI pura, zero dipendenze di sistema
bash <(curl -fsSL https://gitlab.simultech.it/simultech/call-recorder/-/raw/main/install.sh) --no-tray
```

**Windows (PowerShell):**
```powershell
irm https://gitlab.simultech.it/simultech/call-recorder/-/raw/main/install.ps1 | iex
```

Lo script rileva automaticamente OS e architettura, scarica il binario corretto e installa le dipendenze necessarie.

### Binari precompilati

Scarica dalla pagina **Releases** (`/-/releases`):

| Piattaforma | Con tray | Solo CLI |
|---|---|---|
| Linux x86_64 | `call-recorder-linux-amd64` | `call-recorder-linux-amd64-cli` |
| Linux ARM64 | `call-recorder-linux-arm64` | `call-recorder-linux-arm64-cli` |
| Windows x86_64 | `call-recorder-windows-amd64.exe` | — |
| macOS Apple Silicon | `call-recorder-darwin-arm64` | `call-recorder-darwin-arm64-cli` |

```bash
# Linux / macOS
chmod +x call-recorder-*
sudo mv call-recorder-* /usr/local/bin/call-recorder
```

La variante **CLI** (`-cli`) non include la tray icon e non ha dipendenze di sistema.

### Da sorgente

```bash
make install       # con tray — installa in ~/go/bin
make build-cli     # CLI pura — nessuna dipendenza di sistema
```

Aggiungi `~/go/bin` al PATH se non è già presente:
```bash
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc && source ~/.zshrc
```

### Script senza installazione

```bash
./rec.sh   # compila automaticamente se il binario non è in PATH
```

## Utilizzo

```bash
# Lista tutti i device disponibili
call-recorder list

# Registra e trascrivi (loopback rilevato automaticamente)
call-recorder record

# Forza la lingua italiana per la trascrizione
call-recorder record -lang it

# Device specifici
call-recorder record -mic "USB Microphone" -system "analog-stereo.monitor"

# File separati per mic e sistema (senza mix)
call-recorder record -mix=false

# Output in una cartella specifica
call-recorder record -output ~/Registrazioni

# Avvia la tray icon (accetta gli stessi flag di record)
call-recorder tray -lang it
```

I file vengono salvati in `~/recordings/` con nome `call_<timestamp>.wav` e `call_<timestamp>.txt`.

## Tray icon

La tray icon permette di gestire le registrazioni senza usare il terminale.

```bash
# Backend locale (default)
call-recorder tray

# Backend API OpenAI — rilevato automaticamente se OPENAI_API_KEY è impostata
export OPENAI_API_KEY=sk-...
call-recorder tray -lang it
```

Menu disponibile:
- **Avvia registrazione** — icona verde → rossa, registrazione in background
- **Ferma registrazione** — salva WAV e avvia trascrizione automatica
- **Trascrivi ultimo file** — ritrascrive l'ultimo WAV senza registrare di nuovo
- **Esci** — ferma eventuale registrazione in corso e chiude

**Dipendenza sistema Linux** (solo per la variante con tray):
```bash
sudo pacman -S libappindicator-gtk3   # Arch
sudo apt install libappindicator3-1   # Ubuntu/Debian
sudo dnf install libappindicator-gtk3 # Fedora
```

## Trascrizione automatica

Al termine di ogni registrazione la trascrizione parte automaticamente. Sono disponibili due backend:

### Backend locale (default)

Usa `whisper.cpp` o `openai-whisper` installati localmente. Nessun costo, nessun dato inviato fuori dalla macchina.

```bash
call-recorder record -lang it
call-recorder record -lang it -model /path/to/ggml-large-v3-turbo.bin
```

**Installazione (scegli uno):**
```bash
# openai-whisper (Python)
pip install openai-whisper

# whisper.cpp (Arch)
sudo pacman -S whisper.cpp
mkdir -p ~/.local/share/whisper/models
wget -O ~/.local/share/whisper/models/ggml-large-v3-turbo.bin \
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
```

### Backend API (OpenAI)

Invia il file audio all'API di OpenAI. Costo: ~$0.006/minuto.

Se `OPENAI_API_KEY` è impostata nell'ambiente, il backend API viene selezionato automaticamente senza bisogno di `-backend api`.

```bash
export OPENAI_API_KEY=sk-...
call-recorder record -lang it

# Per forzare il backend locale anche con la chiave impostata:
call-recorder record -lang it -backend local
```

I file sono registrati a 16kHz mono (~18 MB/10 min). Per call > ~13 minuti `ffmpeg` comprime automaticamente in MP3 16kbps prima dell'invio (supporta call fino a ~3.5 ore).

**Dipendenza opzionale** (solo per call > 13 min con backend API):
```bash
sudo pacman -S ffmpeg   # Arch
sudo apt install ffmpeg  # Ubuntu/Debian
```

## Setup loopback per OS

### Linux (PipeWire / PulseAudio)
Nessun setup — il monitor sink viene rilevato automaticamente.
Se il device corretto non viene trovato:
```bash
pactl list sources short
call-recorder record -system "analog-stereo.monitor"
```

### Windows
Abilita **Stereo Mix** in Impostazioni → Audio → Registrazione (tasto destro → Mostra dispositivi disabilitati).
Oppure installa [VB-Audio Virtual Cable](https://vb-audio.com/Cable/).

### macOS
Installa [BlackHole](https://github.com/ExistentialAudio/BlackHole), poi:
```bash
call-recorder record -system "BlackHole"
```
Per sentire l'audio mentre registri, crea un Multi-Output Device in Audio MIDI Setup che includa sia il tuo speaker che BlackHole.

## Opzioni complete

| Flag | Default | Descrizione |
|---|---|---|
| `-mic` | system default | Nome parziale del device microfono |
| `-system` | auto-detect | Nome parziale del device audio di sistema |
| `-output` | `~/recordings` | Cartella di output |
| `-mix` | `true` | Mixa mic e sistema in un unico file |
| `-lang` | `auto` | Lingua per la trascrizione (es. `it`, `en`) |
| `-backend` | `local` (auto `api` se `$OPENAI_API_KEY` è impostata) | Backend trascrizione: `local` \| `api` |
| `-model` | auto-detect | Percorso modello whisper.cpp |
| `-api-key` | `$OPENAI_API_KEY` | API key OpenAI |

## Struttura

| File | Ruolo |
|---|---|
| `main.go` | Entry point, routing comandi |
| `devices.go` | Enumerazione device, auto-detect loopback |
| `recorder.go` | Loop di registrazione, mix, gestione stop |
| `wav.go` | Scrittura WAV 16-bit PCM (zero dipendenze esterne) |
| `transcribe.go` | Trascrizione locale (whisper.cpp / openai-whisper) e API |
| `tray.go` | System tray icon e menu (build tag: `!notray`) |
| `tray_stub.go` | Stub per build senza tray (build tag: `notray`) |
| `icons.go` | Icone generate programmaticamente |
| `install.sh` | Script di installazione Linux/macOS |
| `install.ps1` | Script di installazione Windows |
| `rec.sh` | Avvio rapido senza installazione |
| `Makefile` | `build`, `build-cli`, `install`, `clean`, `dist`, `tag` |
| `.github/workflows/macos.yml` | CI GitHub Actions — build macOS (Intel + Apple Silicon) |
