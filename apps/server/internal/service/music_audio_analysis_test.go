package service

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestClassifyPCMRecognizesSteadyDancePulse(t *testing.T) {
	const sampleRate = 8000
	const seconds = 24
	bpm := 124.0
	samples := sampleRate * seconds
	payload := make([]byte, samples*2)
	beatSamples := int(float64(sampleRate) * 60 / bpm)
	for index := 0; index < samples; index++ {
		phase := index % beatSamples
		value := 0.04 * math.Sin(2*math.Pi*220*float64(index)/sampleRate)
		if phase < sampleRate/50 { // 20 ms kick transient.
			value += 0.85 * math.Exp(-float64(phase)/(float64(sampleRate)*0.004)) * math.Sin(2*math.Pi*70*float64(phase)/sampleRate)
		}
		if value > 1 {
			value = 1
		}
		if value < -1 {
			value = -1
		}
		binary.LittleEndian.PutUint16(payload[index*2:], uint16(int16(value*32767)))
	}
	result, err := classifyPCM(payload, sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	if result.BPM < 118 || result.BPM > 130 {
		t.Fatalf("unexpected tempo estimate: %+v", result)
	}
	if result.Genre != "Electronic" || result.Style != "House" {
		t.Fatalf("expected a broad dance classification: %+v", result)
	}
}
