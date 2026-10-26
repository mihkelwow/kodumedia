package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	upnpcastadapter "github.com/mihkelwow/kodumedia/internal/adapters/cast/upnp"
	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	filesystemadapter "github.com/mihkelwow/kodumedia/internal/adapters/filesystem/local"
	nativepickeradapter "github.com/mihkelwow/kodumedia/internal/adapters/filesystem/nativepicker"
	imdbadapter "github.com/mihkelwow/kodumedia/internal/adapters/metadata/imdb"
	tmdbadapter "github.com/mihkelwow/kodumedia/internal/adapters/metadata/tmdb"
	jsonnotificationsadapter "github.com/mihkelwow/kodumedia/internal/adapters/notifications/jsonnotifications"
	ffprobeadapter "github.com/mihkelwow/kodumedia/internal/adapters/probe/ffprobe"
	"github.com/mihkelwow/kodumedia/internal/adapters/settings/jsonsettings"
	localsubtitles "github.com/mihkelwow/kodumedia/internal/adapters/subtitles/local"
	opensubtitlesadapter "github.com/mihkelwow/kodumedia/internal/adapters/subtitles/opensubtitles"
	inspectionadapter "github.com/mihkelwow/kodumedia/internal/adapters/system/inspection"
	selfupdateadapter "github.com/mihkelwow/kodumedia/internal/adapters/system/selfupdate"
	themerrdbadapter "github.com/mihkelwow/kodumedia/internal/adapters/themes/themerrdb"
	ytdlpadapter "github.com/mihkelwow/kodumedia/internal/adapters/themes/ytdlp"
	ffmpegadapter "github.com/mihkelwow/kodumedia/internal/adapters/transcode/ffmpeg"
	jsonwatchadapter "github.com/mihkelwow/kodumedia/internal/adapters/watch/jsonwatch"
	"github.com/mihkelwow/kodumedia/internal/config"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
	"github.com/mihkelwow/kodumedia/internal/service"
	"github.com/mihkelwow/kodumedia/internal/transport/httpapi"
	"github.com/mihkelwow/kodumedia/internal/webui"
)

type App struct {
	server        *http.Server
	transcoder    ports.TranscodeSessionManager
	maintenance   *service.MetadataMaintenanceService
	libraries     *service.LibraryService
	scanner       *service.ScanService
	cancel        context.CancelFunc
	startOnce     sync.Once
	stopRequested <-chan struct{}
	background    context.Context
	storage       *service.StorageService
}

func New(ctx context.Context, cfg config.Config, version string) (*App, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	catalog, err := jsoncatalog.Open(filepath.Join(cfg.DataDir, "catalog.json"))
	if err != nil {
		return nil, err
	}
	settingsRepository, err := jsonsettings.Open(filepath.Join(cfg.DataDir, "settings.json"), cfg.InitialSettings)
	if err != nil {
		return nil, err
	}
	currentSettings := settingsRepository.Current()
	preferredTools := config.PreferBundledTools(currentSettings)
	if preferredTools.FFmpegPath != currentSettings.FFmpegPath || preferredTools.FFprobePath != currentSettings.FFprobePath {
		if err := settingsRepository.Update(ctx, preferredTools); err != nil {
			return nil, fmt.Errorf("migrate bundled media tools: %w", err)
		}
	}
	dataset, err := imdbadapter.NewDatasetManager(
		func() string { return settingsRepository.Current().Metadata.IMDbRatingsSource },
		filepath.Join(cfg.DataDir, "metadata", "imdb"),
	)
	if err != nil {
		return nil, err
	}
	settings := service.NewSettingsService(settingsRepository, inspectionadapter.New(dataset), version)
	libraries := service.NewLibraryService(catalog)
	if err := libraries.EnsureSystemLibraries(ctx); err != nil {
		return nil, err
	}
	watchStore, err := jsonwatchadapter.Open(filepath.Join(cfg.DataDir, "watch-state.json"))
	if err != nil {
		return nil, err
	}
	watch := service.NewWatchService(catalog, watchStore)
	notificationStore, err := jsonnotificationsadapter.Open(filepath.Join(cfg.DataDir, "notifications.json"))
	if err != nil {
		return nil, err
	}
	notifications := service.NewNotificationService(catalog, notificationStore)
	media := service.NewMediaService(catalog, service.WithMediaWatch(watch))
	music := service.NewMusicService(catalog, watch)
	audio := service.NewAudioService(catalog, func() string { return settingsRepository.Current().FFmpegPath })
	musicMetadata := service.NewMusicMetadataProvider(func() domain.MusicSettings { return settingsRepository.Current().Music }, filepath.Join(cfg.DataDir, "music-metadata"), version, service.WithMusicMetadataFFmpeg(func() string { return settingsRepository.Current().FFmpegPath }))
	musicArtwork := service.NewMusicArtworkService(catalog, func() string { return settingsRepository.Current().FFmpegPath }, filepath.Join(cfg.DataDir, "music-artwork"), version, service.WithMusicMetadataArtwork(musicMetadata))
	postPlay := service.NewPostPlayService(catalog, watch)
	appContext, cancel := context.WithCancel(ctx)
	themes := service.NewThemeService(
		catalog,
		themerrdbadapter.New(version),
		ytdlpadapter.NewManaged(
			func() string { return cfg.YTDLPPath },
			func() string { return settingsRepository.Current().FFmpegPath },
			filepath.Join(cfg.DataDir, "tools", "yt-dlp"),
			version,
		),
		filepath.Join(cfg.DataDir, "themes", "series"),
		func() domain.ThemeMusicSettings { return settingsRepository.Current().ThemeMusic },
		service.WithThemeBackgroundContext(appContext),
	)
	jobs := service.NewJobManager()
	musicEnrichment := service.NewMusicEnrichmentService(catalog, musicMetadata, musicArtwork, jobs)
	probe := ffprobeadapter.NewDynamic(func() string { return settingsRepository.Current().FFprobePath })
	transcoder, err := ffmpegadapter.NewManagerDynamic(
		func() string { return settingsRepository.Current().FFmpegPath },
		filepath.Join(cfg.DataDir, "transcodes"),
		cfg.TranscodeIdleTTL,
		ffmpegadapter.SoftwareH264AACProfile{},
	)
	if err != nil {
		cancel()
		return nil, err
	}
	tmdbProvider := tmdbadapter.NewDynamic(func() (string, string) {
		value := settingsRepository.Current()
		return value.Metadata.TMDbToken, value.Metadata.Language
	})
	imdbRatings := imdbadapter.NewDynamicRatingsProvider(dataset.LocalPath)
	metadata := service.NewDynamicMetadataService(
		catalog,
		func() string { return settingsRepository.Current().Metadata.Language },
		[]ports.MetadataProvider{tmdbProvider},
		[]ports.RatingsProvider{imdbRatings},
	)
	maintenance := service.NewMetadataMaintenanceService(appContext, catalog, metadata, dataset, imdbRatings, jobs)
	scanner := service.NewScanService(catalog, probe, jobs, service.WithScanMetadata(metadata), service.WithScanMusicMetadata(musicMetadata))
	if repairedLibraries, repairErr := scanner.RepairSeriesGrouping(ctx); repairErr != nil {
		slog.Warn("repair legacy series grouping", "error", repairErr)
	} else if repairedLibraries > 0 {
		slog.Info("repaired legacy series grouping", "libraries", repairedLibraries)
	}
	subtitleStore, err := localsubtitles.New(func() string { return settingsRepository.Current().FFmpegPath }, cfg.DataDir)
	if err != nil {
		cancel()
		_ = transcoder.Close()
		return nil, fmt.Errorf("initialize subtitle store: %w", err)
	}
	openSubtitles := opensubtitlesadapter.NewDynamic(func() domain.SubtitleSettings { return settingsRepository.Current().Subtitles }, version)
	subtitles := service.NewSubtitleService(catalog, subtitleStore, openSubtitles)
	playback := service.NewPlaybackService(catalog, service.NewPlaybackPlanner(func() domain.QualitySettings { return settingsRepository.Current().Quality }), transcoder, service.WithPlaybackProbe(probe), service.WithPlaybackSubtitles(subtitles))
	casting := service.NewCastService(playback, upnpcastadapter.New())
	stopRequested := make(chan struct{})
	var stopOnce sync.Once
	requestStop := func() { stopOnce.Do(func() { close(stopRequested) }) }
	updateInstaller, err := selfupdateadapter.New(cfg.DataDir, version, cfg.Address)
	if err != nil {
		cancel()
		_ = transcoder.Close()
		return nil, fmt.Errorf("initialize self updater: %w", err)
	}
	storage := service.NewStorageService(
		cfg.DataDir,
		catalog,
		func() domain.ServerSettings { return settingsRepository.Current() },
		transcoder,
		musicArtwork,
		musicMetadata,
		themes,
		subtitleStore,
		watch,
	)
	settings.SetChangeHandler(func(previous, current domain.ServerSettings) {
		maintenance.OnSettingsChanged(previous, current)
		if previous.Quality != current.Quality {
			transcoder.ApplyStoragePolicy(
				current.Quality.CachePolicy,
				int64(current.Quality.CacheMaxGB)<<30,
				int64(current.Quality.CacheFreeReserveGB)<<30,
				current.Quality.CacheFreeReservePercent,
			)
		}
	})
	transcoder.ApplyStoragePolicy(
		currentSettings.Quality.CachePolicy,
		int64(currentSettings.Quality.CacheMaxGB)<<30,
		int64(currentSettings.Quality.CacheFreeReserveGB)<<30,
		currentSettings.Quality.CacheFreeReservePercent,
	)
	updates := service.NewUpdateService(updateInstaller, requestStop)
	handler := httpapi.NewRouter(httpapi.Dependencies{
		Libraries:       libraries,
		Media:           media,
		PostPlay:        postPlay,
		Scanner:         scanner,
		Playback:        playback,
		Subtitles:       subtitles,
		Metadata:        metadata,
		Settings:        settings,
		Filesystem:      filesystemadapter.New(),
		Picker:          nativepickeradapter.New(),
		Maintenance:     maintenance,
		Jobs:            jobs,
		Transcoder:      transcoder,
		FFmpeg:          transcoder,
		FFprobe:         probe,
		AllowedOrigins:  cfg.AllowedOrigins,
		UI:              webui.Handler(),
		Version:         version,
		Updates:         updates,
		Watch:           watch,
		Notifications:   notifications,
		Music:           music,
		Audio:           audio,
		MusicArtwork:    musicArtwork,
		MusicEnrichment: musicEnrichment,
		Themes:          themes,
		Storage:         storage,
		Cast:            casting,
	})
	server := &http.Server{Addr: cfg.Address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	return &App{
		server:        server,
		transcoder:    transcoder,
		maintenance:   maintenance,
		libraries:     libraries,
		scanner:       scanner,
		cancel:        cancel,
		stopRequested: stopRequested,
		background:    appContext,
		storage:       storage,
	}, nil
}

func (a *App) Run() error {
	a.startOnce.Do(func() {
		a.maintenance.Start()
		go a.maintenance.RunPeriodicRefresh()
		if a.storage != nil {
			go a.storage.Run(a.background)
		}
	})
	slog.Info("mediaserver started", "address", a.server.Addr)
	err := a.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.cancel != nil {
		a.cancel()
	}
	serverErr := a.server.Shutdown(ctx)
	transcodeErr := a.transcoder.Close()
	if serverErr != nil {
		return serverErr
	}
	return transcodeErr
}

func (a *App) StopRequested() <-chan struct{} {
	return a.stopRequested
}

// ScanAllLibraries queues one background scan for every configured media
// library. Empty libraries are skipped so the tray action remains safe on a
// fresh installation.
func (a *App) ScanAllLibraries(ctx context.Context) (started int, skipped int, err error) {
	libraries, listErr := a.libraries.List(ctx)
	if listErr != nil {
		return 0, 0, listErr
	}
	var failures []error
	for _, library := range libraries {
		if len(library.Sources) == 0 {
			skipped++
			continue
		}
		if _, startErr := a.scanner.Start(ctx, library.ID); startErr != nil {
			failures = append(failures, fmt.Errorf("scan %s: %w", library.Name, startErr))
			continue
		}
		started++
	}
	return started, skipped, errors.Join(failures...)
}
