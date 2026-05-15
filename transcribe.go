package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// whisperBackend represents a working whisper binary.
type whisperBackend struct {
	path string
	kind string // "cpp" | "python"
}

// Candidates in preference order: whisper.cpp first, Python fallback.
var whisperCandidates = []struct{ name, kind string }{
	{"whisper-cli", "cpp"},
	{"whisper-cpp", "cpp"},
	{"whisper", "python"},
}

var modelPreference = []string{
	"ggml-large-v3-turbo.bin",
	"ggml-large-v3.bin",
	"ggml-large.bin",
	"ggml-medium.bin",
	"ggml-small.bin",
	"ggml-base.bin",
	"ggml-tiny.bin",
}

var modelSearchDirs = []string{
	filepath.Join(os.Getenv("HOME"), ".local", "share", "whisper", "models"),
	"/usr/share/whisper.cpp/models",
	"./models",
	".",
}

// findWhisper returns the first binary that actually loads (no missing shared libs).
func findWhisper() *whisperBackend {
	for _, c := range whisperCandidates {
		path, err := exec.LookPath(c.name)
		if err != nil {
			continue
		}
		if binaryWorks(path) {
			return &whisperBackend{path: path, kind: c.kind}
		}
	}
	return nil
}

// binaryWorks runs --help and checks there are no shared-library load errors.
func binaryWorks(path string) bool {
	var stderr bytes.Buffer
	cmd := exec.Command(path, "--help")
	cmd.Stderr = &stderr
	cmd.Stdout = &bytes.Buffer{}
	cmd.Run()
	return !strings.Contains(stderr.String(), "cannot open shared object file")
}

func findModel(preferred string) string {
	if preferred != "" {
		if fileExists(preferred) {
			return preferred
		}
		for _, dir := range modelSearchDirs {
			if p := filepath.Join(dir, preferred); fileExists(p) {
				return p
			}
		}
		return ""
	}
	for _, dir := range modelSearchDirs {
		for _, name := range modelPreference {
			if p := filepath.Join(dir, name); fileExists(p) {
				return p
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func transcribe(wavPath, modelPath, lang string) (string, error) {
	backend := findWhisper()
	if backend == nil {
		printWhisperInstallHelp()
		return "", nil
	}

	outDir := filepath.Dir(wavPath)
	outBase := strings.TrimSuffix(wavPath, filepath.Ext(wavPath))

	var cmd *exec.Cmd

	switch backend.kind {
	case "cpp":
		model := findModel(modelPath)
		if model == "" {
			printWhisperModelHelp()
			return "", nil
		}
		fmt.Printf("[transcribe] backend: whisper.cpp — modello: %s\n", filepath.Base(model))
		cmd = exec.Command(backend.path,
			"-m", model,
			"-f", wavPath,
			"-l", lang,
			"-otxt",
			"-of", outBase,
		)

	case "python":
		model := pythonModelName(modelPath)
		fmt.Printf("[transcribe] backend: openai-whisper — modello: %s (cpu)\n", model)
		args := []string{wavPath,
			"--model", model,
			"--device", "cpu",
			"--output_dir", outDir,
			"--output_format", "txt",
		}
		if lang != "auto" {
			args = append(args, "--language", lang)
		}
		cmd = exec.Command(backend.path, args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper: %w", err)
	}

	// whisper.cpp writes <outBase>.txt; openai-whisper writes <basename>.txt in outDir
	if fileExists(outBase + ".txt") {
		return outBase + ".txt", nil
	}
	alt := filepath.Join(outDir, filepath.Base(outBase)+".txt")
	if fileExists(alt) {
		return alt, nil
	}
	return "", fmt.Errorf("file trascrizione non trovato dopo l'esecuzione")
}

// pythonModelName converts a whisper.cpp model path/name to an openai-whisper model name.
func pythonModelName(preferred string) string {
	if preferred == "" {
		return "small" // buon compromesso velocità/qualità su CPU
	}
	base := strings.TrimPrefix(filepath.Base(preferred), "ggml-")
	base = strings.TrimSuffix(base, ".bin")
	return base
}

func printWhisperInstallHelp() {
	fmt.Fprintln(os.Stderr, "\n[transcribe] nessun backend whisper funzionante trovato.")
	fmt.Fprintln(os.Stderr, "  Opzione 1 — openai-whisper (Python):")
	fmt.Fprintln(os.Stderr, "    pip install openai-whisper")
	fmt.Fprintln(os.Stderr, "  Opzione 2 — whisper.cpp (CPU-only, senza CUDA):")
	fmt.Fprintln(os.Stderr, "    https://github.com/ggerganov/whisper.cpp")
}

func printWhisperModelHelp() {
	fmt.Fprintln(os.Stderr, "\n[transcribe] nessun modello trovato.")
	fmt.Fprintln(os.Stderr, "  Scarica un modello da: https://huggingface.co/ggerganov/whisper.cpp")
	fmt.Fprintln(os.Stderr, "  Salva in: ~/.local/share/whisper/models/")
	fmt.Fprintln(os.Stderr, "  Consigliato: ggml-large-v3-turbo.bin")
	fmt.Fprintln(os.Stderr, "  Oppure specifica il percorso con: -model /path/to/ggml-*.bin")
}
