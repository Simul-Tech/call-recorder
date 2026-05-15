package main

import (
	"fmt"
	"os"

	"github.com/gen2brain/malgo"
)

func main() {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {})
	if err != nil {
		fmt.Fprintf(os.Stderr, "audio context init error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		if err := listDevices(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "record":
		cfg, err := parseRecordFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printUsage()
			os.Exit(1)
		}
		files, err := record(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Recording error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Saved:")
		for _, f := range files {
			fmt.Println(" →", f)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`call-recorder — records mic + system audio during calls

USAGE:
  call-recorder list
      List all available audio devices

  call-recorder record [OPTIONS]
      Start recording (Ctrl+C to stop)

OPTIONS:
  -mic     <name>   Mic device (partial name match, default: system default)
  -system  <name>   System audio device (partial name match, auto-detected on Linux/Windows)
  -output  <dir>    Output directory (default: ./recordings)
  -mix=false        Save mic and system as separate WAV files

EXAMPLES:
  call-recorder list
  call-recorder record
  call-recorder record -system "analog-stereo.monitor" -output ~/Recordings
  call-recorder record -mix -output ~/Recordings

SETUP PER OS:
  Linux   → nessun setup, rileva automaticamente il sink .monitor di PipeWire/PulseAudio
  Windows → abilita "Stereo Mix" in Impostazioni audio, oppure installa VB-Audio Cable
  macOS   → installa BlackHole: https://github.com/ExistentialAudio/BlackHole`)
}
