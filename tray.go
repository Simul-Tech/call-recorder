//go:build !notray

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"fyne.io/systray"
	"github.com/gen2brain/malgo"
)

var traryCfg *RecordConfig

func runTray(cfg *RecordConfig) {
	traryCfg = cfg
	systray.Run(trayReady, func() {})
}

func trayReady() {
	systray.SetIcon(iconIdle())
	systray.SetTooltip("call-recorder — inattivo")

	mStart := systray.AddMenuItem("Avvia registrazione", "")
	mStop := systray.AddMenuItem("Ferma registrazione", "")
	mStop.Disable()
	systray.AddSeparator()
	mTranscribe := systray.AddMenuItem("Trascrivi ultimo file", "")
	mTranscribe.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Esci", "")

	var (
		mu      sync.Mutex
		stopCh  chan struct{}
		lastWAV string
	)

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				mu.Lock()
				if stopCh != nil {
					mu.Unlock()
					continue
				}
				ch := make(chan struct{})
				stopCh = ch
				mu.Unlock()

				mStart.Disable()
				mStop.Enable()
				systray.SetIcon(iconRecording())
				systray.SetTooltip("call-recorder — registrazione in corso...")

				go func() {
					ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(string) {})
					if err != nil {
						fmt.Fprintf(os.Stderr, "[tray] errore contesto audio: %v\n", err)
						resetTrayState(&mu, &stopCh, mStart, mStop)
						return
					}
					defer func() { _ = ctx.Uninit(); ctx.Free() }()

					files, err := record(ctx, traryCfg, ch)

					mu.Lock()
					stopCh = nil
					for _, f := range files {
						if strings.HasSuffix(f, ".wav") {
							lastWAV = f
							break
						}
					}
					mu.Unlock()

					mStop.Disable()
					mStart.Enable()
					systray.SetIcon(iconIdle())

					if err != nil {
						systray.SetTooltip("call-recorder — errore: " + err.Error())
						fmt.Fprintf(os.Stderr, "[tray] errore registrazione: %v\n", err)
					} else {
						systray.SetTooltip("call-recorder — inattivo")
						mTranscribe.Enable()
					}
				}()

			case <-mStop.ClickedCh:
				mu.Lock()
				if stopCh != nil {
					close(stopCh)
					stopCh = nil
				}
				mu.Unlock()

			case <-mTranscribe.ClickedCh:
				mu.Lock()
				wav := lastWAV
				mu.Unlock()
				if wav == "" {
					continue
				}
				mTranscribe.Disable()
				go func() {
					txt, err := transcribe(wav, traryCfg.ModelPath, traryCfg.Lang, traryCfg.Backend, traryCfg.APIKey)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[tray] errore trascrizione: %v\n", err)
					} else if txt != "" {
						fmt.Println("[tray] trascrizione salvata:", txt)
					}
					mTranscribe.Enable()
				}()

			case <-mQuit.ClickedCh:
				mu.Lock()
				if stopCh != nil {
					close(stopCh)
					stopCh = nil
				}
				mu.Unlock()
				systray.Quit()
				return
			}
		}
	}()
}

func resetTrayState(mu *sync.Mutex, stopCh *chan struct{}, mStart, mStop *systray.MenuItem) {
	mu.Lock()
	*stopCh = nil
	mu.Unlock()
	mStop.Disable()
	mStart.Enable()
	systray.SetIcon(iconIdle())
	systray.SetTooltip("call-recorder — inattivo")
}

