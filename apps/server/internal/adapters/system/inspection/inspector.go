package inspection

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const defaultTMDbBaseURL = "https://api.themoviedb.org/3"

type Inspector struct {
	client      *http.Client
	tmdbBaseURL string
	dataset     ports.RatingsDataset
}

func New(dataset ...ports.RatingsDataset) *Inspector {
	var ratings ports.RatingsDataset
	if len(dataset) > 0 {
		ratings = dataset[0]
	}
	return &Inspector{client: &http.Client{Timeout: 8 * time.Second}, tmdbBaseURL: defaultTMDbBaseURL, dataset: ratings}
}

func (i *Inspector) Inspect(ctx context.Context, settings domain.ServerSettings) domain.SystemStatus {
	type result struct {
		name   string
		status domain.ComponentStatus
	}
	results := make(chan result, 4)
	var group sync.WaitGroup
	group.Add(4)
	go func() {
		defer group.Done()
		results <- result{"ffmpeg", inspectExecutable(ctx, settings.FFmpegPath, "FFmpeg")}
	}()
	go func() {
		defer group.Done()
		results <- result{"ffprobe", inspectExecutable(ctx, settings.FFprobePath, "FFprobe")}
	}()
	go func() {
		defer group.Done()
		results <- result{"tmdb", i.inspectTMDb(ctx, settings.Metadata.TMDbToken)}
	}()
	go func() {
		defer group.Done()
		if i.dataset == nil {
			results <- result{"imdb", domain.ComponentStatus{Message: "IMDb andmestiku haldur pole saadaval."}}
			return
		}
		results <- result{"imdb", i.dataset.Inspect(ctx, settings.Metadata.IMDbRatingsSource)}
	}()
	go func() {
		group.Wait()
		close(results)
	}()

	var status domain.SystemStatus
	for item := range results {
		switch item.name {
		case "ffmpeg":
			status.FFmpeg = item.status
		case "ffprobe":
			status.FFprobe = item.status
		case "tmdb":
			status.Metadata.TMDb = item.status
		case "imdb":
			status.Metadata.IMDb = item.status
		}
	}
	return status
}

func inspectExecutable(parent context.Context, path, name string) domain.ComponentStatus {
	path = strings.TrimSpace(path)
	if path == "" {
		return domain.ComponentStatus{Message: name + " tee pole seadistatud."}
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, "-version")
	childprocess.Hide(command)
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return domain.ComponentStatus{Message: name + " kontroll aegus."}
		}
		return domain.ComponentStatus{Message: fmt.Sprintf("%s ei k\u00e4ivitunud: %v", name, err)}
	}
	return domain.ComponentStatus{Available: true, Version: firstLine(string(output)), Message: name + " t\u00f6\u00f6tab."}
}

func (i *Inspector) inspectTMDb(parent context.Context, token string) domain.ComponentStatus {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ComponentStatus{Message: "TMDb API Read Access Token pole seadistatud."}
	}
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(i.tmdbBaseURL, "/")+"/configuration", nil)
	if err != nil {
		return domain.ComponentStatus{Message: "TMDb kontrollp\u00e4ringut ei saanud luua."}
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	response, err := i.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return domain.ComponentStatus{Message: "TMDb \u00fchenduse kontroll aegus."}
		}
		return domain.ComponentStatus{Message: fmt.Sprintf("TMDb \u00fchendus eba\u00f5nnestus: %v", err)}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return domain.ComponentStatus{Message: "TMDb token ei kehti v\u00f5i sellel puudub ligip\u00e4\u00e4s."}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.ComponentStatus{Message: fmt.Sprintf("TMDb vastas HTTP %d.", response.StatusCode)}
	}
	return domain.ComponentStatus{Available: true, Message: "TMDb \u00fchendus ja token t\u00f6\u00f6tavad."}
}

func firstLine(value string) string {
	if index := strings.IndexAny(value, "\r\n"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

var _ ports.SettingsInspector = (*Inspector)(nil)
