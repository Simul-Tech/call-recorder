package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

const sampleRate = 16000

type RecordConfig struct {
	MicName    string
	SystemName string
	OutputDir  string
	Mix        bool
	Lang       string
	ModelPath  string
	Backend    string // "local" | "api"
	APIKey     string
}

func parseRecordFlags(args []string) (*RecordConfig, error) {
	home, _ := os.UserHomeDir()
	defaultOutput := filepath.Join(home, "recordings")

	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	mic := fs.String("mic", "", "Mic device name (partial match)")
	system := fs.String("system", "", "System audio device name (partial match)")
	output := fs.String("output", defaultOutput, "Output directory")
	mix := fs.Bool("mix", true, "Merge into single WAV file")
	lang := fs.String("lang", "auto", "Transcription language (e.g. it, en, auto)")
	model := fs.String("model", "", "Path to whisper.cpp model file (auto-detected if empty)")
	backend := fs.String("backend", "local", "Transcription backend: local | api")
	apiKey := fs.String("api-key", "", "OpenAI API key (default: $OPENAI_API_KEY)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	resolvedBackend := *backend
	if resolvedBackend == "local" && os.Getenv("OPENAI_API_KEY") != "" {
		resolvedBackend = "api"
	}

	return &RecordConfig{
		MicName:    *mic,
		SystemName: *system,
		OutputDir:  *output,
		Mix:        *mix,
		Lang:       *lang,
		ModelPath:  *model,
		Backend:    resolvedBackend,
		APIKey:     *apiKey,
	}, nil
}

type captureStream struct {
	mu       sync.Mutex
	samples  []float32
	device   *malgo.Device
	channels uint32
}

func newCaptureStream(ctx *malgo.AllocatedContext, info *malgo.DeviceInfo) (*captureStream, error) {
	cs := &captureStream{}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = 0 // device native channel count
	cfg.SampleRate = sampleRate
	if info != nil {
		cfg.Capture.DeviceID = info.ID.Pointer()
	}

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, _ uint32) {
			floats := bytesToFloat32(input)
			cs.mu.Lock()
			cs.samples = append(cs.samples, floats...)
			cs.mu.Unlock()
		},
	}

	device, err := malgo.InitDevice(ctx.Context, cfg, callbacks)
	if err != nil {
		return nil, fmt.Errorf("cannot open device: %w", err)
	}

	cs.device = device
	cs.channels = device.CaptureChannels()
	if cs.channels == 0 {
		cs.channels = 1
	}
	if cs.channels > 2 {
		cs.channels = 2
	}
	return cs, nil
}

func bytesToFloat32(data []byte) []float32 {
	n := len(data) / 4
	out := make([]float32, n)
	for i := range out {
		bits := binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		out[i] = math.Float32frombits(bits)
	}
	return out
}

func (cs *captureStream) start() error { return cs.device.Start() }
func (cs *captureStream) stop()        { _ = cs.device.Stop() }
func (cs *captureStream) close()       { cs.device.Uninit() }

func (cs *captureStream) drain() []float32 {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	out := cs.samples
	cs.samples = nil
	return out
}

func record(ctx *malgo.AllocatedContext, cfg *RecordConfig, stopCh <-chan struct{}) ([]string, error) {
	// Resolve mic
	var micInfo *malgo.DeviceInfo
	if cfg.MicName != "" {
		info, err := findCaptureDevice(ctx, cfg.MicName)
		if err != nil {
			return nil, fmt.Errorf("mic: %w", err)
		}
		micInfo = info
		fmt.Printf("[mic]    Using: %s\n", micInfo.Name())
	} else {
		fmt.Println("[mic]    Using: system default")
	}

	// Resolve system audio
	var sysInfo *malgo.DeviceInfo
	if cfg.SystemName != "" {
		info, err := findCaptureDevice(ctx, cfg.SystemName)
		if err != nil {
			return nil, fmt.Errorf("system: %w", err)
		}
		sysInfo = info
	} else {
		info, err := autoDetectLoopback(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil {
			printLoopbackHelp()
			return nil, fmt.Errorf("no loopback device found — use -system to specify one manually")
		}
		sysInfo = info
	}
	fmt.Printf("[system] Using: %s\n", sysInfo.Name())

	// Open streams
	micStream, err := newCaptureStream(ctx, micInfo)
	if err != nil {
		return nil, fmt.Errorf("mic stream: %w", err)
	}
	defer micStream.close()

	sysStream, err := newCaptureStream(ctx, sysInfo)
	if err != nil {
		return nil, fmt.Errorf("system stream: %w", err)
	}
	defer sysStream.close()

	if err := micStream.start(); err != nil {
		return nil, fmt.Errorf("mic start: %w", err)
	}
	if err := sysStream.start(); err != nil {
		return nil, fmt.Errorf("system start: %w", err)
	}

	fmt.Println("\n[call-recorder] Recording... Press Ctrl+C to stop.\n")
	<-stopCh
	fmt.Println("\n[call-recorder] Stopping...")

	micStream.stop()
	sysStream.stop()

	micSamples := micStream.drain()
	sysSamples := sysStream.drain()
	micCh := int(micStream.channels)
	sysCh := int(sysStream.channels)

	fmt.Printf("[call-recorder] mic=%d samples, system=%d samples\n", len(micSamples), len(sysSamples))

	if len(micSamples) == 0 && len(sysSamples) == 0 {
		return nil, fmt.Errorf("no audio captured — check device permissions")
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("20060102_150405")
	var saved []string

	if cfg.Mix {
		mixed := mixBuffers(toMono(micSamples, micCh), toMono(sysSamples, sysCh))
		path := fmt.Sprintf("%s/call_%s.wav", cfg.OutputDir, timestamp)
		if err := writeWav(path, mixed, sampleRate, 1); err != nil {
			return nil, err
		}
		saved = append(saved, path)
	} else {
		if len(micSamples) > 0 {
			path := fmt.Sprintf("%s/call_%s_mic.wav", cfg.OutputDir, timestamp)
			if err := writeWav(path, micSamples, sampleRate, micCh); err != nil {
				return nil, err
			}
			saved = append(saved, path)
		}
		if len(sysSamples) > 0 {
			path := fmt.Sprintf("%s/call_%s_system.wav", cfg.OutputDir, timestamp)
			if err := writeWav(path, sysSamples, sampleRate, sysCh); err != nil {
				return nil, err
			}
			saved = append(saved, path)
		}
	}

	notify("Registrazione completata", fmt.Sprintf("%d file salvati in %s", len(saved), cfg.OutputDir))

	// Transcribe the mixed file (or the mic file as fallback)
	var toTranscribe string
	if cfg.Mix && len(saved) > 0 {
		toTranscribe = saved[0]
	} else {
		for _, f := range saved {
			if strings.HasSuffix(f, "_mic.wav") {
				toTranscribe = f
				break
			}
		}
	}

	if toTranscribe != "" {
		fmt.Println("\n[transcribe] Avvio trascrizione...")
		txt, err := transcribe(toTranscribe, cfg.ModelPath, cfg.Lang, cfg.Backend, cfg.APIKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[transcribe] errore: %v\n", err)
		} else if txt != "" {
			saved = append(saved, txt)
			notify("Trascrizione completata", txt)
		}
	}

	return saved, nil
}

func toMono(samples []float32, channels int) []float32 {
	if channels <= 1 {
		return samples
	}
	out := make([]float32, len(samples)/channels)
	for i := range out {
		var sum float32
		for c := 0; c < channels; c++ {
			sum += samples[i*channels+c]
		}
		out[i] = sum / float32(channels)
	}
	return out
}

func mixBuffers(a, b []float32) []float32 {
	length := len(a)
	if len(b) > length {
		length = len(b)
	}
	out := make([]float32, length)
	for i := range out {
		var sa, sb float32
		if i < len(a) {
			sa = a[i]
		}
		if i < len(b) {
			sb = b[i]
		}
		v := (sa + sb) * 0.5
		if v > 1.0 {
			v = 1.0
		}
		if v < -1.0 {
			v = -1.0
		}
		out[i] = v
	}
	return out
}

func printLoopbackHelp() {
	fmt.Fprintln(os.Stderr)
	switch runtime.GOOS {
	case "linux":
		fmt.Fprintln(os.Stderr, "  Linux: nessun monitor sink trovato.")
		fmt.Fprintln(os.Stderr, "  Esegui: pactl list sources short")
		fmt.Fprintln(os.Stderr, "  Cerca una sorgente che finisce in '.monitor', poi:")
		fmt.Fprintln(os.Stderr, "    call-recorder record -system '<nome>'")
	case "windows":
		fmt.Fprintln(os.Stderr, "  Windows: abilita 'Stereo Mix' in Impostazioni > Audio > Registrazione")
		fmt.Fprintln(os.Stderr, "  Oppure installa VB-Audio Virtual Cable: https://vb-audio.com/Cable/")
	case "darwin":
		fmt.Fprintln(os.Stderr, "  macOS: installa BlackHole: https://github.com/ExistentialAudio/BlackHole")
		fmt.Fprintln(os.Stderr, "  Poi: call-recorder record -system 'BlackHole'")
	}
}
