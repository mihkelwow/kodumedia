package httpapi

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": a.dependencies.Version})
}
func (a *API) systemInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	ffmpegAvailable := a.dependencies.FFmpeg != nil && a.dependencies.FFmpeg.Available(ctx)
	ffprobeAvailable := a.dependencies.FFprobe != nil && a.dependencies.FFprobe.Available(ctx)
	metadataEnabled := a.dependencies.MetadataEnabled
	if a.dependencies.Metadata != nil {
		metadataEnabled = a.dependencies.Metadata.Available()
	}
	writeJSON(w, http.StatusOK, map[string]any{"version": a.dependencies.Version, "ffmpegAvailable": ffmpegAvailable, "ffprobeAvailable": ffprobeAvailable, "metadataEnabled": metadataEnabled})
}
func (a *API) getSystemSettings(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Settings == nil {
		writeError(w, fmt.Errorf("%w: settings service is not configured", domain.ErrUnavailable))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, a.dependencies.Settings.Configuration(ctx))
}
func (a *API) testSystemSettings(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Settings == nil {
		writeError(w, fmt.Errorf("%w: settings service is not configured", domain.ErrUnavailable))
		return
	}
	var input domain.ServerSettingsUpdate
	if !decodeJSON(w, r, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	configuration, err := a.dependencies.Settings.Test(ctx, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configuration)
}
func (a *API) updateSystemSettings(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Settings == nil {
		writeError(w, fmt.Errorf("%w: settings service is not configured", domain.ErrUnavailable))
		return
	}
	var input domain.ServerSettingsUpdate
	if !decodeJSON(w, r, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	configuration, err := a.dependencies.Settings.Update(ctx, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configuration)
}
func (a *API) browseFilesystem(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Filesystem == nil {
		writeError(w, fmt.Errorf("%w: filesystem browser is not configured", domain.ErrUnavailable))
		return
	}
	values := r.URL.Query()
	includeFiles, _ := strconv.ParseBool(strings.TrimSpace(values.Get("files")))
	extensions := make([]string, 0, 8)
	for _, value := range values["extension"] {
		extensions = append(extensions, strings.Split(value, ",")...)
	}
	if value := strings.TrimSpace(values.Get("extensions")); value != "" {
		extensions = append(extensions, strings.Split(value, ",")...)
	}
	listing, err := a.dependencies.Filesystem.Browse(r.Context(), strings.TrimSpace(values.Get("path")), includeFiles, extensions)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (a *API) pickFilesystem(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Picker == nil {
		writeError(w, fmt.Errorf("%w: native Windows picker is not configured", domain.ErrUnavailable))
		return
	}
	if !isLoopbackRequest(r) {
		writeError(w, fmt.Errorf("%w: native Windows picker can be opened only from a browser on the server computer", domain.ErrUnavailable))
		return
	}
	var request domain.FilePickerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.Title) > 160 || len(request.Extensions) > 16 {
		writeError(w, fmt.Errorf("%w: invalid picker request", domain.ErrInvalidInput))
		return
	}
	path, canceled, err := a.dependencies.Picker.Pick(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	if canceled {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, domain.FilePickerResult{Path: path})
}

func (a *API) refreshIMDbRatings(w http.ResponseWriter, _ *http.Request) {
	if a.dependencies.Maintenance == nil {
		writeError(w, fmt.Errorf("%w: metadata maintenance is not configured", domain.ErrUnavailable))
		return
	}
	writeJSON(w, http.StatusAccepted, a.dependencies.Maintenance.RefreshIMDb())
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

const maxUpdateRequestBytes = int64(130 << 20)

func (a *API) getUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Updates == nil {
		writeError(w, fmt.Errorf("%w: update service is not configured", domain.ErrUnavailable))
		return
	}
	if !isTrustedUpdateRequest(r, false) {
		writeError(w, fmt.Errorf("%w: rakenduse uuendamine on lubatud ainult serveriarvuti localhost-brauserist", domain.ErrForbidden))
		return
	}
	writeJSON(w, http.StatusOK, a.dependencies.Updates.Status())
}

func (a *API) installUpdate(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Updates == nil {
		writeError(w, fmt.Errorf("%w: update service is not configured", domain.ErrUnavailable))
		return
	}
	if !isTrustedUpdateRequest(r, true) {
		writeError(w, fmt.Errorf("%w: rakenduse uuendamine on lubatud ainult serveriarvuti localhost-brauserist", domain.ErrForbidden))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpdateRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, fmt.Errorf("%w: oodati multipart ZIP-paketti", domain.ErrInvalidInput))
		return
	}
	var packagePartFound bool
	for {
		part, nextErr := reader.NextPart()
		if nextErr != nil {
			if nextErr == io.EOF {
				break
			}
			writeError(w, fmt.Errorf("%w: uuenduspaketi lugemine ebaõnnestus", domain.ErrInvalidInput))
			return
		}
		if part.FormName() != "package" || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		if packagePartFound {
			_ = part.Close()
			writeError(w, fmt.Errorf("%w: korraga saab paigaldada ühe uuenduspaketi", domain.ErrInvalidInput))
			return
		}
		packagePartFound = true
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		status, installErr := a.dependencies.Updates.Install(ctx, part, part.FileName())
		cancel()
		_ = part.Close()
		if installErr != nil {
			writeError(w, installErr)
			return
		}
		writeJSON(w, http.StatusAccepted, status)
		return
	}
	if !packagePartFound {
		writeError(w, fmt.Errorf("%w: uuenduspakett puudub", domain.ErrInvalidInput))
	}
}

func isTrustedUpdateRequest(r *http.Request, requireIntentHeader bool) bool {
	if !isLoopbackRequest(r) || !isLoopbackHost(r.Host) {
		return false
	}
	if requireIntentHeader && r.Header.Get("X-KoduMedia-Update") != "install" {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || !isLoopbackHost(parsed.Host) {
		return false
	}
	requestHost, requestPort := splitHostAndPort(r.Host)
	originHost, originPort := splitHostAndPort(parsed.Host)
	if !isLoopbackHostname(requestHost) || !isLoopbackHostname(originHost) {
		return false
	}
	return effectivePort(parsed.Scheme, originPort) == effectivePort("http", requestPort)
}

func isLoopbackHost(value string) bool {
	host, _ := splitHostAndPort(value)
	return isLoopbackHostname(host)
}

func isLoopbackHostname(value string) bool {
	value = strings.Trim(strings.TrimSpace(value), "[]")
	if strings.EqualFold(value, "localhost") {
		return true
	}
	ip := net.ParseIP(value)
	return ip != nil && ip.IsLoopback()
}

func splitHostAndPort(value string) (string, string) {
	value = strings.TrimSpace(value)
	if host, port, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]"), port
	}
	return strings.Trim(value, "[]"), ""
}

func effectivePort(scheme, port string) string {
	if port != "" {
		return port
	}
	if strings.EqualFold(scheme, "https") {
		return "443"
	}
	return "80"
}
