package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiMaxBytes  = 25 * 1024 * 1024 // 25 MB — limite OpenAI
	apiEndpoint  = "https://api.openai.com/v1/audio/transcriptions"
	apiModel     = "whisper-1"
)

// ── Local backend ─────────────────────────────────────────────────────────────

type whisperBackend struct {
	path string
	kind string // "cpp" | "python"
}

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

func transcribeLocal(wavPath, modelPath, lang string) (string, error) {
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
			"-m", model, "-f", wavPath, "-l", lang, "-otxt", "-of", outBase,
		)
	case "python":
		model := pythonModelName(modelPath)
		fmt.Printf("[transcribe] backend: openai-whisper — modello: %s (cpu)\n", model)
		args := []string{wavPath,
			"--model", model, "--device", "cpu",
			"--output_dir", outDir, "--output_format", "txt",
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

	if fileExists(outBase + ".txt") {
		return outBase + ".txt", nil
	}
	alt := filepath.Join(outDir, filepath.Base(outBase)+".txt")
	if fileExists(alt) {
		return alt, nil
	}
	return "", fmt.Errorf("file trascrizione non trovato")
}

func pythonModelName(preferred string) string {
	if preferred == "" {
		return "small"
	}
	base := strings.TrimPrefix(filepath.Base(preferred), "ggml-")
	return strings.TrimSuffix(base, ".bin")
}

// ── API backend ───────────────────────────────────────────────────────────────

func transcribeAPI(wavPath, lang, apiKey string) (string, error) {
	uploadPath, isTemp, err := prepareForAPI(wavPath)
	if err != nil {
		return "", err
	}
	if isTemp {
		defer os.Remove(uploadPath)
	}

	info, _ := os.Stat(uploadPath)
	fmt.Printf("[transcribe] backend: OpenAI API — file: %s (%.1f MB)\n",
		filepath.Base(uploadPath), float64(info.Size())/1024/1024)

	text, err := callWhisperAPI(uploadPath, lang, apiKey)
	if err != nil {
		return "", err
	}

	outPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".txt"
	if err := os.WriteFile(outPath, []byte(text), 0644); err != nil {
		return "", fmt.Errorf("salvataggio trascrizione: %w", err)
	}
	return outPath, nil
}

// prepareForAPI returns a path ready to upload.
// If the WAV exceeds 25 MB it compresses to MP3 at 16kbps via ffmpeg (~3.5h max).
func prepareForAPI(wavPath string) (path string, isTemp bool, err error) {
	info, err := os.Stat(wavPath)
	if err != nil {
		return "", false, err
	}
	if info.Size() <= apiMaxBytes {
		return wavPath, false, nil
	}

	fmt.Printf("[transcribe] file troppo grande (%.1f MB), comprimo con ffmpeg...\n",
		float64(info.Size())/1024/1024)

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", false, fmt.Errorf(
			"file supera 25 MB e ffmpeg non è installato\n" +
				"  Arch:   sudo pacman -S ffmpeg\n" +
				"  Ubuntu: sudo apt install ffmpeg",
		)
	}

	mp3Path := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + "_upload.mp3"
	cmd := exec.Command("ffmpeg", "-y", "-i", wavPath,
		"-ac", "1", "-b:a", "16k", mp3Path,
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("ffmpeg: %w", err)
	}
	return mp3Path, true, nil
}

func callWhisperAPI(filePath, lang, apiKey string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	_ = w.WriteField("model", apiModel)
	_ = w.WriteField("response_format", "text")
	if lang != "auto" {
		_ = w.WriteField("language", lang)
	}
	w.Close()

	req, err := http.NewRequest("POST", apiEndpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("richiesta API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct{ Message string } `json:"error"`
		}
		json.Unmarshal(respBody, &apiErr)
		msg := apiErr.Error.Message
		if msg == "" {
			msg = string(respBody)
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
	}

	return strings.TrimSpace(string(respBody)), nil
}

// ── Entry point ───────────────────────────────────────────────────────────────

func transcribe(wavPath, modelPath, lang, backend, apiKey string) (string, error) {
	switch backend {
	case "api":
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return "", fmt.Errorf(
				"API key mancante: imposta OPENAI_API_KEY o usa -api-key <key>",
			)
		}
		return transcribeAPI(wavPath, lang, apiKey)
	default:
		return transcribeLocal(wavPath, modelPath, lang)
	}
}

// ── Help ──────────────────────────────────────────────────────────────────────

func printWhisperInstallHelp() {
	fmt.Fprintln(os.Stderr, "\n[transcribe] nessun backend locale trovato.")
	fmt.Fprintln(os.Stderr, "  openai-whisper: pip install openai-whisper")
	fmt.Fprintln(os.Stderr, "  whisper.cpp:    https://github.com/ggerganov/whisper.cpp")
	fmt.Fprintln(os.Stderr, "  oppure usa il backend API: -backend api -api-key <key>")
}

func printWhisperModelHelp() {
	fmt.Fprintln(os.Stderr, "\n[transcribe] nessun modello trovato.")
	fmt.Fprintln(os.Stderr, "  Scarica: https://huggingface.co/ggerganov/whisper.cpp")
	fmt.Fprintln(os.Stderr, "  Salva in: ~/.local/share/whisper/models/")
	fmt.Fprintln(os.Stderr, "  Consigliato: ggml-large-v3-turbo.bin")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
