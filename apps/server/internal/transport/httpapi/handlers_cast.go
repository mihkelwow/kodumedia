package httpapi

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

type castSessionRequest struct {
	DeviceID    string                     `json:"deviceId"`
	MediaID     string                     `json:"mediaId"`
	Title       string                     `json:"title"`
	Subtitle    string                     `json:"subtitle,omitempty"`
	Preferences domain.PlaybackPreferences `json:"preferences"`
}

type castSourceRequest struct {
	MediaID     string                     `json:"mediaId"`
	Preferences domain.PlaybackPreferences `json:"preferences"`
}

func (a *API) castDevices(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	devices, err := a.dependencies.Cast.Devices(ctx)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"items": devices})
}

func (a *API) startCastSession(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var request castSessionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, err := a.dependencies.Cast.Start(r.Context(), service.CastStartRequest{
		DeviceID: request.DeviceID, MediaID: request.MediaID, BaseURL: castRequestBaseURL(r),
		Title: request.Title, Subtitle: request.Subtitle, Preferences: request.Preferences,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (a *API) activeCastSession(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	session, found := a.dependencies.Cast.Active(r.URL.Query().Get("mediaId"))
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *API) castSessionStatus(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	session, err := a.dependencies.Cast.Status(ctx, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, session)
}

func (a *API) commandCastSession(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var command domain.CastCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	ctx, cancel := contextWithTimeout(r, 8*time.Second)
	defer cancel()
	session, err := a.dependencies.Cast.Command(ctx, r.PathValue("id"), command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *API) stopCastSession(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r, 8*time.Second)
	defer cancel()
	if err := a.dependencies.Cast.Stop(ctx, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createCastSource(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var request castSourceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	source, err := a.dependencies.Cast.PrepareBrowserSource(r.Context(), service.CastSourceRequest{
		MediaID: request.MediaID, BaseURL: castRequestBaseURL(r), Preferences: request.Preferences,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, source)
}

func (a *API) stopCastSource(w http.ResponseWriter, r *http.Request) {
	if a.dependencies.Cast == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	if err := a.dependencies.Cast.StopSource(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func castRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost"
	}
	if scheme == "http" {
		hostname, port, err := net.SplitHostPort(host)
		if err != nil {
			hostname = host
			port = ""
		}
		parsed := net.ParseIP(strings.Trim(hostname, "[]"))
		if parsed == nil || parsed.IsLoopback() || parsed.IsUnspecified() {
			if localIP := preferredCastIP(r.RemoteAddr); localIP != nil {
				if port != "" {
					host = net.JoinHostPort(localIP.String(), port)
				} else {
					host = localIP.String()
				}
			}
		}
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func preferredCastIP(remoteAddress string) net.IP {
	remoteHost, _, _ := net.SplitHostPort(remoteAddress)
	remoteIP := net.ParseIP(remoteHost)
	if remoteIP != nil && !remoteIP.IsLoopback() && !remoteIP.IsUnspecified() {
		if connection, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: remoteIP, Port: 9}); err == nil {
			local, _ := connection.LocalAddr().(*net.UDPAddr)
			_ = connection.Close()
			if local != nil && local.IP.To4() != nil && !local.IP.IsLoopback() {
				return local.IP
			}
		}
	}
	type candidate struct {
		ip   net.IP
		rank int
	}
	values := make([]candidate, 0)
	interfaces, _ := net.Interfaces()
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, _ := networkInterface.Addrs()
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil || ip.To4() == nil || !ip.IsPrivate() {
				continue
			}
			octets := ip.To4()
			rank := 3
			switch {
			case octets[0] == 192 && octets[1] == 168:
				rank = 0
			case octets[0] == 10:
				rank = 1
			case octets[0] == 172 && octets[1] >= 16 && octets[1] <= 31:
				rank = 2
			}
			values = append(values, candidate{ip: ip, rank: rank})
		}
	}
	sort.Slice(values, func(left, right int) bool { return values[left].rank < values[right].rank })
	if len(values) > 0 {
		return values[0].ip
	}
	return nil
}
