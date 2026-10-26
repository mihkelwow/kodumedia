package service

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

// AudioService owns the lossless compatibility path for audio-only media. It
// never uses a lossy encoder. Browsers that can decode the source should use
// /stream directly; this service is only the fallback for unsupported codecs.
type AudioService struct {
	catalog ports.Catalog
	ffmpeg  func() string
}

func NewAudioService(catalog ports.Catalog, ffmpeg func() string) *AudioService {
	return &AudioService{catalog: catalog, ffmpeg: ffmpeg}
}

func (s *AudioService) StreamLossless(ctx context.Context, mediaID string, output io.Writer) error {
	item, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return err
	}
	if item.Type != domain.MediaTypeMusic || strings.TrimSpace(item.File.Path) == "" {
		return fmt.Errorf("%w: media item is not an audio track", domain.ErrUnsupported)
	}
	binary := "ffmpeg"
	if s.ffmpeg != nil && strings.TrimSpace(s.ffmpeg()) != "" {
		binary = strings.TrimSpace(s.ffmpeg())
	}
	codec := losslessPCMCodec(item)
	command := exec.CommandContext(ctx, binary,
		"-v", "error",
		"-i", item.File.Path,
		"-map", "0:a:0",
		"-vn",
		"-c:a", codec,
		"-f", "wav",
		"pipe:1",
	)
	childprocess.Hide(command)
	command.Stdout = output
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("lossless audio conversion failed: %s", message)
	}
	return nil
}

func losslessPCMCodec(item domain.MediaItem) string {
	codec := strings.ToLower(item.Technical.AudioCodec)
	if strings.Contains(codec, "f64") {
		return "pcm_f64le"
	}
	if strings.Contains(codec, "f32") {
		return "pcm_f32le"
	}
	depth := item.Technical.AudioBitDepth
	if depth <= 0 {
		for _, track := range item.Technical.Tracks {
			if track.Kind == "audio" && track.BitDepth > depth {
				depth = track.BitDepth
			}
		}
	}
	switch {
	case depth > 24:
		return "pcm_s32le"
	case depth > 16:
		return "pcm_s24le"
	default:
		return "pcm_s16le"
	}
}
