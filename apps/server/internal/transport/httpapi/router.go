package httpapi

import (
	"context"
	"net/http"

	"github.com/mihkelwow/kodumedia/internal/ports"
	"github.com/mihkelwow/kodumedia/internal/service"
)

type Availability interface {
	Available(ctx context.Context) bool
}
type Dependencies struct {
	Libraries       *service.LibraryService
	Media           *service.MediaService
	PostPlay        *service.PostPlayService
	Scanner         *service.ScanService
	Playback        *service.PlaybackService
	Subtitles       *service.SubtitleService
	Metadata        *service.MetadataService
	Settings        *service.SettingsService
	Filesystem      ports.DirectoryBrowser
	Picker          ports.FilePicker
	Maintenance     *service.MetadataMaintenanceService
	Jobs            ports.JobRunner
	Transcoder      ports.TranscodeSessionManager
	FFmpeg          Availability
	FFprobe         Availability
	MetadataEnabled bool
	AllowedOrigins  []string
	UI              http.Handler
	Version         string
	Updates         *service.UpdateService
	Watch           *service.WatchService
	Notifications   *service.NotificationService
	Music           *service.MusicService
	Audio           *service.AudioService
	MusicArtwork    *service.MusicArtworkService
	MusicEnrichment *service.MusicEnrichmentService
	Themes          *service.ThemeService
	Storage         *service.StorageService
	Cast            *service.CastService
}
type API struct{ dependencies Dependencies }

func NewRouter(dependencies Dependencies) http.Handler {
	api := &API{dependencies: dependencies}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", api.health)
	mux.HandleFunc("GET /api/v1/system/info", api.systemInfo)
	mux.HandleFunc("GET /api/v1/system/settings", api.getSystemSettings)
	mux.HandleFunc("PUT /api/v1/system/settings", api.updateSystemSettings)
	mux.HandleFunc("POST /api/v1/system/settings/test", api.testSystemSettings)
	mux.HandleFunc("GET /api/v1/filesystem/browse", api.browseFilesystem)
	mux.HandleFunc("POST /api/v1/filesystem/pick", api.pickFilesystem)
	mux.HandleFunc("POST /api/v1/system/metadata/imdb/refresh", api.refreshIMDbRatings)
	mux.HandleFunc("GET /api/v1/system/update", api.getUpdateStatus)
	mux.HandleFunc("POST /api/v1/system/update", api.installUpdate)
	mux.HandleFunc("GET /api/v1/system/storage", api.getStorage)
	mux.HandleFunc("POST /api/v1/system/storage", api.manageStorage)
	mux.HandleFunc("GET /api/v1/libraries", api.listLibraries)
	mux.HandleFunc("POST /api/v1/libraries", api.createLibrary)
	mux.HandleFunc("GET /api/v1/libraries/{id}", api.getLibrary)
	mux.HandleFunc("PATCH /api/v1/libraries/{id}", api.updateLibrary)
	mux.HandleFunc("DELETE /api/v1/libraries/{id}", api.deleteLibrary)
	mux.HandleFunc("POST /api/v1/libraries/{id}/sources", api.addLibrarySource)
	mux.HandleFunc("DELETE /api/v1/libraries/{id}/sources/{sourceId}", api.removeLibrarySource)
	mux.HandleFunc("POST /api/v1/libraries/{id}/scan", api.scanLibrary)
	mux.HandleFunc("GET /api/v1/jobs/{id}", api.getJob)
	mux.HandleFunc("GET /api/v1/media", api.listMedia)
	mux.HandleFunc("GET /api/v1/watch/continue", api.continueWatching)
	mux.HandleFunc("GET /api/v1/notifications", api.listNotifications)
	mux.HandleFunc("POST /api/v1/notifications/read", api.markNotificationsRead)
	mux.HandleFunc("PUT /api/v1/media/{id}/watch", api.updateWatchState)
	mux.HandleFunc("GET /api/v1/media/{id}", api.getMedia)
	mux.HandleFunc("GET /api/v1/media/{id}/post-play", api.getPostPlay)
	mux.HandleFunc("GET /api/v1/media/{id}/stream", api.streamMedia)
	mux.HandleFunc("GET /api/v1/media/{id}/audio/lossless", api.streamLosslessAudio)
	mux.HandleFunc("GET /api/v1/media/{id}/artwork", api.serveMusicArtwork)
	mux.HandleFunc("GET /api/v1/music", api.getMusicLanding)
	mux.HandleFunc("GET /api/v1/music/tracks", api.getMusicTracks)
	mux.HandleFunc("POST /api/v1/music/enrich", api.enrichMusic)
	mux.HandleFunc("GET /api/v1/music/artists/{artist}", api.getMusicArtist)
	mux.HandleFunc("GET /api/v1/music/albums/{albumId}", api.getMusicAlbum)
	mux.HandleFunc("GET /api/v1/media/{id}/playback-options", api.playbackOptions)
	mux.HandleFunc("GET /api/v1/media/{id}/subtitles/search", api.searchSubtitles)
	mux.HandleFunc("POST /api/v1/media/{id}/subtitles/download", api.downloadSubtitle)
	mux.HandleFunc("GET /api/v1/media/{id}/subtitles/{subtitleId}", api.serveSubtitle)
	mux.HandleFunc("POST /api/v1/media/{id}/metadata/refresh", api.refreshMetadata)
	mux.HandleFunc("GET /api/v1/series/{id}", api.getSeriesDetails)
	mux.HandleFunc("GET /api/v1/series/{id}/theme", api.serveSeriesTheme)
	mux.HandleFunc("GET /api/v1/themes/settings", api.getThemeSettings)
	mux.HandleFunc("PUT /api/v1/themes/settings", api.updateThemeSettings)
	mux.HandleFunc("POST /api/v1/playback/plan", api.planPlayback)
	mux.HandleFunc("POST /api/v1/playback/sessions", api.startPlayback)
	mux.HandleFunc("GET /api/v1/playback/sessions/{id}/stats", api.playbackStats)
	mux.HandleFunc("DELETE /api/v1/playback/sessions/{id}", api.stopPlayback)
	mux.HandleFunc("GET /api/v1/playback/sessions/{id}/{file...}", api.servePlaybackFile)
	mux.HandleFunc("GET /api/v1/cast/devices", api.castDevices)
	mux.HandleFunc("GET /api/v1/cast/sessions/active", api.activeCastSession)
	mux.HandleFunc("POST /api/v1/cast/sessions", api.startCastSession)
	mux.HandleFunc("GET /api/v1/cast/sessions/{id}", api.castSessionStatus)
	mux.HandleFunc("POST /api/v1/cast/sessions/{id}/commands", api.commandCastSession)
	mux.HandleFunc("DELETE /api/v1/cast/sessions/{id}", api.stopCastSession)
	mux.HandleFunc("POST /api/v1/cast/sources", api.createCastSource)
	mux.HandleFunc("DELETE /api/v1/cast/sources/{id}", api.stopCastSource)
	if dependencies.UI != nil {
		mux.Handle("/", dependencies.UI)
	}
	return withMiddleware(mux, dependencies.AllowedOrigins)
}
