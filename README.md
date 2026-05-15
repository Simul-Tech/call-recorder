# call-recorder

Registra simultaneamente microfono e audio di sistema durante le chiamate, salva un file WAV mixato e lo trascrive automaticamente.

Usa **[miniaudio](https://miniaud.io/)** tramite `gen2brain/malgo` — libreria C embedded nel pacchetto, **zero dipendenze di sistema** su qualsiasi piattaforma.

## Download

Scarica il binario precompilato dalla pagina **Releases** del repository (`/-/releases`):

| Piattaforma | File |
|---|---|
| Linux x86_64 | `call-recorder-linux-amd64` |
| Linux ARM64 | `call-recorder-linux-arm64` |
| Windows x86_64 | `call-recorder-windows-amd64.exe` |
| macOS Intel | `call-recorder-macos-amd64` |
| macOS Apple Silicon | `call-recorder-macos-arm64` |

```bash
# Linux / macOS — rendi eseguibile e sposta in PATH
chmod +x call-recorder-*
sudo mv call-recorder-* /usr/local/bin/call-recorder
```

## Installazione da sorgente

```bash
make install   # compila e installa in ~/go/bin
make build     # solo binario locale
```

Aggiungi `~/go/bin` al PATH se non è già presente:

```bash
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc && source ~/.zshrc
```

### Alias rapido

```bash
alias rec='call-recorder record'
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
```

I file vengono salvati in `./recordings/` con nome `call_<timestamp>.wav` e `call_<timestamp>.txt`.

## Trascrizione automatica

Al termine di ogni registrazione viene avviata automaticamente la trascrizione. Sono disponibili due backend:

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
# modello consigliato (~800 MB):
mkdir -p ~/.local/share/whisper/models
wget -O ~/.local/share/whisper/models/ggml-large-v3-turbo.bin \
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
```

### Backend API (OpenAI)

Invia il file audio all'API di OpenAI. Richiede una API key e connessione internet. Costo: ~$0.006/minuto.
I file più grandi di 25 MB vengono compressi automaticamente con `ffmpeg` prima dell'invio.

```bash
export OPENAI_API_KEY=sk-...
call-recorder record -lang it -backend api

# oppure inline
call-recorder record -lang it -backend api -api-key sk-...
```

**Dipendenza opzionale:** `ffmpeg` (per la compressione automatica dei file grandi):
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
| `-output` | `./recordings` | Cartella di output |
| `-mix` | `true` | Mixa mic e sistema in un unico file |
| `-lang` | `auto` | Lingua per la trascrizione (es. `it`, `en`) |
| `-backend` | `local` | Backend trascrizione: `local` \| `api` |
| `-model` | auto-detect | Percorso modello whisper.cpp |
| `-api-key` | `$OPENAI_API_KEY` | API key OpenAI |

## Struttura

| File | Ruolo |
|---|---|
| `main.go` | Entry point, routing comandi |
| `devices.go` | Enumerazione device, auto-detect loopback |
| `recorder.go` | Loop di registrazione, mix, gestione Ctrl+C |
| `wav.go` | Scrittura WAV 16-bit PCM (zero dipendenze esterne) |
| `transcribe.go` | Trascrizione locale (whisper.cpp / openai-whisper) e API |
| `rec.sh` | Script di avvio senza installazione |
| `Makefile` | `build`, `install`, `clean`, `dist`, `tag` |
