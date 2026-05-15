# call-recorder

Registra simultaneamente microfono e audio di sistema durante le chiamate e salva un unico file WAV mixato.

Usa **[miniaudio](https://miniaud.io/)** tramite il binding Go `gen2brain/malgo` — libreria C embedded nel pacchetto, **zero dipendenze di sistema** su qualsiasi piattaforma.

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
# Installa il binario in ~/go/bin (deve essere in PATH)
make install

# Oppure compila solo il binario locale
make build
```

### Alias rapido (opzionale)

Aggiungi al tuo `.zshrc` / `.bashrc`:

```bash
alias rec='call-recorder record'
```

Poi `source ~/.zshrc` per attivarlo.

### Script senza installazione

```bash
./rec.sh        # compila automaticamente se il binario non è in PATH
```

## Utilizzo

```bash
# Lista tutti i device disponibili
call-recorder list

# Registra (salva in ./recordings/, rileva automaticamente il loopback)
call-recorder record

# Con device specifici
call-recorder record -mic "USB Microphone" -system "analog-stereo.monitor"

# File separati per mic e sistema
call-recorder record -mix=false

# Output personalizzato
call-recorder record -output ~/Registrazioni
```

I file vengono salvati in `./recordings/` con nome `call_<timestamp>.wav`.

## Setup loopback per OS

### Linux (PipeWire / PulseAudio)
Nessun setup — il monitor sink viene rilevato automaticamente.
Se il device corretto non viene trovato:
```bash
pactl list sources short
# trova il nome del monitor, es: alsa_output.pci-0000_00_1f.3.analog-stereo.monitor
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

## Struttura

| File          | Ruolo                                              |
|---------------|----------------------------------------------------|
| `main.go`     | Entry point, routing comandi                       |
| `devices.go`  | Enumerazione device, auto-detect loopback          |
| `recorder.go` | Loop di registrazione, mix, gestione Ctrl+C        |
| `wav.go`      | Scrittura WAV 16-bit PCM (zero dipendenze esterne) |
| `rec.sh`      | Script di avvio senza installazione                |
| `Makefile`    | Build, install, clean                              |
