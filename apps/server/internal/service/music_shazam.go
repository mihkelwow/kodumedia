package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"math/cmplx"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
)

// The fingerprint format and peak detection are a Go implementation of the
// MIT-licensed shazamio-core algorithm. See THIRD-PARTY-NOTICES.txt.
const (
	shazamSampleRate      = 16000
	shazamSegmentSeconds  = 12
	shazamFFTSize         = 2048
	shazamFFTStep         = 128
	shazamFFTOutputLength = shazamFFTSize/2 + 1
	shazamRingLength      = 256
	shazamDataURIPrefix   = "data:audio/vnd.shazam.sig;base64,"
)

type shazamFrequencyPeak struct {
	FFTPassNumber             uint32
	PeakMagnitude             uint16
	CorrectedPeakFrequencyBin uint16
}

type shazamSignature struct {
	SampleRate    uint32
	NumberSamples uint32
	Bands         [4][]shazamFrequencyPeak
}

type shazamResponse struct {
	Matches []struct {
		ID string `json:"id"`
	} `json:"matches"`
	Track *struct {
		Key      string `json:"key"`
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		URL      string `json:"url"`
		Genres   struct {
			Primary string `json:"primary"`
		} `json:"genres"`
		Images struct {
			CoverArt   string `json:"coverart"`
			CoverArtHQ string `json:"coverarthq"`
			Background string `json:"background"`
		} `json:"images"`
		Sections []struct {
			Type     string `json:"type"`
			Metadata []struct {
				Title string `json:"title"`
				Text  string `json:"text"`
			} `json:"metadata"`
		} `json:"sections"`
	} `json:"track"`
}

func (p *MusicMetadataProvider) lookupShazam(ctx context.Context, item domain.MediaItem) (MusicMetadataMatch, bool, error) {
	offsets := shazamSampleOffsets(item)
	var lastErr error
	for _, offset := range offsets {
		samples, err := p.decodeShazamSample(ctx, item.File.Path, offset)
		if err != nil {
			lastErr = err
			continue
		}
		signature := buildShazamSignature(samples)
		if shazamPeakCount(signature) < 5 {
			lastErr = errors.New("Shazam fingerprint did not contain enough peaks")
			continue
		}
		match, found, err := p.sendShazamSignature(ctx, signature)
		if err != nil {
			lastErr = err
			continue
		}
		if found {
			return match, true, nil
		}
	}
	if lastErr != nil {
		return MusicMetadataMatch{}, false, lastErr
	}
	return MusicMetadataMatch{}, false, nil
}

func shazamSampleOffsets(item domain.MediaItem) []float64 {
	duration := float64(item.Technical.DurationMS) / 1000
	if duration <= shazamSegmentSeconds {
		return []float64{0}
	}
	middle := math.Max(0, duration/2-shazamSegmentSeconds/2)
	offsets := []float64{middle}
	// A second, different section materially improves recognition for long DJ
	// edits with intros while avoiding extra requests for normal radio tracks.
	if duration >= 75 {
		alternate := math.Max(0, duration*0.28-shazamSegmentSeconds/2)
		if math.Abs(alternate-middle) >= shazamSegmentSeconds {
			offsets = append(offsets, alternate)
		}
	}
	return offsets
}

func (p *MusicMetadataProvider) decodeShazamSample(ctx context.Context, path string, offsetSeconds float64) ([]int16, error) {
	binaryPath := "ffmpeg"
	if p.ffmpeg != nil && strings.TrimSpace(p.ffmpeg()) != "" {
		binaryPath = strings.TrimSpace(p.ffmpeg())
	}
	args := []string{"-nostdin", "-v", "error"}
	if offsetSeconds > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", offsetSeconds))
	}
	args = append(args,
		"-i", path,
		"-t", fmt.Sprintf("%d", shazamSegmentSeconds),
		"-vn", "-sn", "-dn",
		"-ac", "1", "-ar", fmt.Sprintf("%d", shazamSampleRate),
		"-f", "s16le", "pipe:1",
	)
	command := exec.CommandContext(ctx, binaryPath, args...)
	childprocess.Hide(command)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start Shazam sample decoding: %w", err)
	}
	maximum := shazamSampleRate*shazamSegmentSeconds*2 + 4096
	payload, readErr := io.ReadAll(io.LimitReader(stdout, int64(maximum)))
	waitErr := command.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return nil, fmt.Errorf("decode Shazam sample: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}
	if len(payload)%2 != 0 {
		payload = payload[:len(payload)-1]
	}
	if len(payload) < shazamSampleRate*3*2 {
		return nil, errors.New("Shazam audio sample was too short")
	}
	samples := make([]int16, len(payload)/2)
	for index := range samples {
		samples[index] = int16(binary.LittleEndian.Uint16(payload[index*2:]))
	}
	return samples, nil
}

func (p *MusicMetadataProvider) sendShazamSignature(ctx context.Context, signature shazamSignature) (MusicMetadataMatch, bool, error) {
	uri, err := encodeShazamSignatureURI(signature)
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}

	p.shazamMu.Lock()
	defer p.shazamMu.Unlock()
	if wait := time.Until(p.lastShazam.Add(850 * time.Millisecond)); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return MusicMetadataMatch{}, false, ctx.Err()
		case <-timer.C:
		}
	}

	firstUUID, err := randomShazamUUID()
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}
	secondUUID, err := randomShazamUUID()
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}
	timestamp := p.now().UnixMilli()
	body := map[string]any{
		"timezone": "Europe/Tallinn",
		"signature": map[string]any{
			"uri":      uri,
			"samplems": int64(signature.NumberSamples) * 1000 / int64(signature.SampleRate),
		},
		"timestamp":   timestamp,
		"context":     map[string]any{},
		"geolocation": map[string]any{},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}
	baseURL := strings.TrimRight(p.shazamURL, "/")
	endpoint := fmt.Sprintf("%s/discovery/v5/en-US/US/iphone/-/tag/%s/%s?sync=true&webv3=true&sampling=true&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3", baseURL, firstUUID, secondUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-Shazam-Platform", "IPHONE")
	req.Header.Set("X-Shazam-AppVersion", "14.1.0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148")
	resp, err := p.client.Do(req)
	p.lastShazam = p.now()
	if err != nil {
		return MusicMetadataMatch{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return MusicMetadataMatch{}, false, fmt.Errorf("Shazam recognition was rate limited")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MusicMetadataMatch{}, false, fmt.Errorf("Shazam recognition returned %s", resp.Status)
	}
	var payload shazamResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return MusicMetadataMatch{}, false, err
	}
	return musicMatchFromShazamResponse(payload)
}

func musicMatchFromShazamResponse(payload shazamResponse) (MusicMetadataMatch, bool, error) {
	if payload.Track == nil || strings.TrimSpace(payload.Track.Title) == "" || len(payload.Matches) == 0 {
		return MusicMetadataMatch{}, false, nil
	}
	track := payload.Track
	album := ""
	for _, section := range track.Sections {
		for _, metadata := range section.Metadata {
			if strings.EqualFold(strings.TrimSpace(metadata.Title), "Album") {
				album = strings.TrimSpace(metadata.Text)
				break
			}
		}
		if album != "" {
			break
		}
	}
	genres := normalizeCatalogGenres(optionalMusicValue(track.Genres.Primary))
	artwork := firstNonEmptyMusicURL(track.Images.CoverArtHQ, track.Images.CoverArt, track.Images.Background)
	sourceURL := strings.TrimSpace(track.URL)
	if sourceURL == "" && strings.TrimSpace(track.Key) != "" {
		sourceURL = "https://www.shazam.com/track/" + strings.TrimSpace(track.Key)
	}
	return MusicMetadataMatch{
		Title:         strings.TrimSpace(track.Title),
		Artist:        strings.TrimSpace(track.Subtitle),
		Album:         album,
		Genres:        genres,
		ArtworkURL:    artwork,
		ArtworkSource: "shazam",
		Source:        "shazam",
		SourceURL:     sourceURL,
		Method:        "shazam-fingerprint",
		Score:         100,
	}, true, nil
}

func buildShazamSignature(samples []int16) shazamSignature {
	generator := newShazamSignatureGenerator(uint32(len(samples)))
	for start := 0; start+shazamFFTStep <= len(samples); start += shazamFFTStep {
		generator.process(samples[start : start+shazamFFTStep])
	}
	return generator.signature
}

type shazamSignatureGenerator struct {
	ring          []int16
	ringIndex     int
	window        []float64
	fftBuffer     []complex128
	fftOutputs    [][]float64
	fftIndex      int
	spreadOutputs [][]float64
	spreadIndex   int
	numSpreadFFTs uint32
	signature     shazamSignature
}

func newShazamSignatureGenerator(numberSamples uint32) *shazamSignatureGenerator {
	fftOutputs := make([][]float64, shazamRingLength)
	spreadOutputs := make([][]float64, shazamRingLength)
	for index := 0; index < shazamRingLength; index++ {
		fftOutputs[index] = make([]float64, shazamFFTOutputLength)
		spreadOutputs[index] = make([]float64, shazamFFTOutputLength)
	}
	window := make([]float64, shazamFFTSize)
	for index := range window {
		// shazamio-core uses the 2048 non-zero points of a 2050-point Hann
		// window (the leading and trailing zeroes are omitted).
		window[index] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(index+1)/float64(shazamFFTSize+1))
	}
	return &shazamSignatureGenerator{
		ring:          make([]int16, shazamFFTSize),
		window:        window,
		fftBuffer:     make([]complex128, shazamFFTSize),
		fftOutputs:    fftOutputs,
		spreadOutputs: spreadOutputs,
		signature: shazamSignature{
			SampleRate:    shazamSampleRate,
			NumberSamples: numberSamples,
		},
	}
}

func (g *shazamSignatureGenerator) process(chunk []int16) {
	copy(g.ring[g.ringIndex:g.ringIndex+shazamFFTStep], chunk)
	g.ringIndex = (g.ringIndex + shazamFFTStep) & (shazamFFTSize - 1)

	for index := 0; index < shazamFFTSize; index++ {
		value := float64(g.ring[(index+g.ringIndex)&(shazamFFTSize-1)]) * g.window[index]
		g.fftBuffer[index] = complex(value, 0)
	}
	fftComplexInPlace(g.fftBuffer)
	output := g.fftOutputs[g.fftIndex]
	for index := 0; index < shazamFFTOutputLength; index++ {
		magnitude := cmplx.Abs(g.fftBuffer[index])
		power := magnitude * magnitude / float64(1<<17)
		if power < 1e-10 {
			power = 1e-10
		}
		output[index] = power
	}
	g.fftIndex = (g.fftIndex + 1) & (shazamRingLength - 1)

	latest := g.fftOutputs[(g.fftIndex-1)&(shazamRingLength-1)]
	spread := g.spreadOutputs[g.spreadIndex]
	copy(spread, latest)
	for position := 0; position <= 1022; position++ {
		spread[position] = math.Max(spread[position], math.Max(spread[position+1], spread[position+2]))
	}
	for position := 0; position < shazamFFTOutputLength; position++ {
		value := spread[position]
		for _, former := range []int{1, 3, 6} {
			formerOutput := g.spreadOutputs[(g.spreadIndex-former)&(shazamRingLength-1)]
			if value > formerOutput[position] {
				formerOutput[position] = value
			}
		}
	}
	g.spreadIndex = (g.spreadIndex + 1) & (shazamRingLength - 1)
	g.numSpreadFFTs++
	if g.numSpreadFFTs >= 46 {
		g.recognizePeaks()
	}
}

func (g *shazamSignatureGenerator) recognizePeaks() {
	fftMinus46 := g.fftOutputs[(g.fftIndex-46)&(shazamRingLength-1)]
	fftMinus49 := g.spreadOutputs[(g.spreadIndex-49)&(shazamRingLength-1)]
	for binPosition := 10; binPosition <= 1014; binPosition++ {
		peak := fftMinus46[binPosition]
		if peak < 1.0/64.0 || peak < fftMinus49[binPosition-1] {
			continue
		}
		maxNeighbor := 0.0
		for _, offset := range []int{-10, -7, -4, -3, 1, 2, 5, 8} {
			maxNeighbor = math.Max(maxNeighbor, fftMinus49[binPosition+offset])
		}
		if peak <= maxNeighbor {
			continue
		}
		for _, offset := range []int{-53, -45, 165, 172, 179, 186, 193, 200, 214, 221, 228, 235, 242, 249} {
			other := g.spreadOutputs[(g.spreadIndex+offset)&(shazamRingLength-1)]
			maxNeighbor = math.Max(maxNeighbor, other[binPosition-1])
		}
		if peak <= maxNeighbor {
			continue
		}

		magnitude := math.Max(math.Log(peak), 1.0/64.0)*1477.3 + 6144.0
		magnitudeBefore := math.Max(math.Log(fftMinus46[binPosition-1]), 1.0/64.0)*1477.3 + 6144.0
		magnitudeAfter := math.Max(math.Log(fftMinus46[binPosition+1]), 1.0/64.0)*1477.3 + 6144.0
		variation1 := magnitude*2 - magnitudeBefore - magnitudeAfter
		if variation1 <= 0 || math.IsNaN(variation1) || math.IsInf(variation1, 0) {
			continue
		}
		variation2 := (magnitudeAfter - magnitudeBefore) * 32 / variation1
		corrected := binPosition*64 + int(variation2)
		if corrected < 0 || corrected > math.MaxUint16 {
			continue
		}
		frequencyHz := float64(corrected) * (float64(shazamSampleRate) / 2 / 1024 / 64)
		band := -1
		switch {
		case frequencyHz >= 250 && frequencyHz < 520:
			band = 0
		case frequencyHz >= 520 && frequencyHz < 1450:
			band = 1
		case frequencyHz >= 1450 && frequencyHz < 3500:
			band = 2
		case frequencyHz >= 3500 && frequencyHz <= 5500:
			band = 3
		}
		if band < 0 {
			continue
		}
		if magnitude < 0 {
			magnitude = 0
		}
		if magnitude > math.MaxUint16 {
			magnitude = math.MaxUint16
		}
		g.signature.Bands[band] = append(g.signature.Bands[band], shazamFrequencyPeak{
			FFTPassNumber:             g.numSpreadFFTs - 46,
			PeakMagnitude:             uint16(magnitude),
			CorrectedPeakFrequencyBin: uint16(corrected),
		})
	}
}

func fftComplexInPlace(values []complex128) {
	n := len(values)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			values[i], values[j] = values[j], values[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		step := cmplx.Rect(1, angle)
		half := length >> 1
		for start := 0; start < n; start += length {
			factor := complex(1.0, 0)
			for offset := 0; offset < half; offset++ {
				even := values[start+offset]
				odd := values[start+offset+half] * factor
				values[start+offset] = even + odd
				values[start+offset+half] = even - odd
				factor *= step
			}
		}
	}
}

func encodeShazamSignatureURI(signature shazamSignature) (string, error) {
	buffer := &bytes.Buffer{}
	write32 := func(value uint32) error { return binary.Write(buffer, binary.LittleEndian, value) }
	for _, value := range []uint32{
		0xcafe2580, 0, 0, 0x94119c00,
		0, 0, 0, shazamSampleRateID(signature.SampleRate) << 27,
		0, 0, signature.NumberSamples + uint32(float64(signature.SampleRate)*0.24),
		(15 << 19) + 0x40000,
		0x40000000, 0,
	} {
		if err := write32(value); err != nil {
			return "", err
		}
	}
	for bandIndex, peaks := range signature.Bands {
		if len(peaks) == 0 {
			continue
		}
		peakBuffer := &bytes.Buffer{}
		var previous uint32
		for _, peak := range peaks {
			if peak.FFTPassNumber < previous {
				return "", errors.New("Shazam peaks were not ordered")
			}
			if peak.FFTPassNumber-previous >= 255 {
				_ = peakBuffer.WriteByte(0xff)
				_ = binary.Write(peakBuffer, binary.LittleEndian, peak.FFTPassNumber)
				previous = peak.FFTPassNumber
			}
			_ = peakBuffer.WriteByte(byte(peak.FFTPassNumber - previous))
			_ = binary.Write(peakBuffer, binary.LittleEndian, peak.PeakMagnitude)
			_ = binary.Write(peakBuffer, binary.LittleEndian, peak.CorrectedPeakFrequencyBin)
			previous = peak.FFTPassNumber
		}
		if err := write32(0x60030040 + uint32(bandIndex)); err != nil {
			return "", err
		}
		if err := write32(uint32(peakBuffer.Len())); err != nil {
			return "", err
		}
		if _, err := buffer.Write(peakBuffer.Bytes()); err != nil {
			return "", err
		}
		for padding := (4 - peakBuffer.Len()%4) % 4; padding > 0; padding-- {
			_ = buffer.WriteByte(0)
		}
	}
	payload := buffer.Bytes()
	if len(payload) < 56 {
		return "", errors.New("Shazam signature was incomplete")
	}
	sizeMinusHeader := uint32(len(payload) - 48)
	binary.LittleEndian.PutUint32(payload[8:12], sizeMinusHeader)
	binary.LittleEndian.PutUint32(payload[52:56], sizeMinusHeader)
	binary.LittleEndian.PutUint32(payload[4:8], crc32.ChecksumIEEE(payload[8:]))
	return shazamDataURIPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func shazamSampleRateID(sampleRate uint32) uint32 {
	switch sampleRate {
	case 8000:
		return 1
	case 11025:
		return 2
	case 16000:
		return 3
	case 32000:
		return 4
	case 44100:
		return 5
	case 48000:
		return 6
	default:
		return 0
	}
}

func shazamPeakCount(signature shazamSignature) int {
	count := 0
	for _, peaks := range signature.Bands {
		count += len(peaks)
	}
	return count
}

func randomShazamUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return strings.ToUpper(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(value[0:4]),
		binary.BigEndian.Uint16(value[4:6]),
		binary.BigEndian.Uint16(value[6:8]),
		binary.BigEndian.Uint16(value[8:10]),
		value[10:16],
	)), nil
}
