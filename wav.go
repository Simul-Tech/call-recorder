package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// writeWav writes float32 samples to a 16-bit PCM WAV file.
// No external dependencies — pure stdlib.
func writeWav(path string, samples []float32, sampleRate int, channels int) error {
	if len(samples) == 0 {
		return fmt.Errorf("writeWav: no samples to write")
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writeWav: cannot create %s: %w", path, err)
	}
	defer f.Close()

	numSamples := len(samples)
	bitsPerSample := 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := numSamples * bitsPerSample / 8
	chunkSize := 36 + dataSize

	le := binary.LittleEndian

	// RIFF header
	f.Write([]byte("RIFF"))
	writeU32(f, le, uint32(chunkSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	writeU32(f, le, 16)                      // chunk size
	writeU16(f, le, 1)                        // PCM
	writeU16(f, le, uint16(channels))
	writeU32(f, le, uint32(sampleRate))
	writeU32(f, le, uint32(byteRate))
	writeU16(f, le, uint16(blockAlign))
	writeU16(f, le, uint16(bitsPerSample))

	// data chunk
	f.Write([]byte("data"))
	writeU32(f, le, uint32(dataSize))

	// samples: convert float32 [-1,1] → int16
	buf := make([]byte, 2)
	for _, s := range samples {
		if s > 1.0 { s = 1.0 }
		if s < -1.0 { s = -1.0 }
		v := int16(s * 32767)
		le.PutUint16(buf, uint16(v))
		f.Write(buf)
	}

	return nil
}

func writeU16(f *os.File, order binary.ByteOrder, v uint16) {
	b := make([]byte, 2)
	order.PutUint16(b, v)
	f.Write(b)
}

func writeU32(f *os.File, order binary.ByteOrder, v uint32) {
	b := make([]byte, 4)
	order.PutUint32(b, v)
	f.Write(b)
}
