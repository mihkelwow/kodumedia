package upnp

import (
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestUPnPTimeRoundTrip(t *testing.T) {
	const position = int64(7_445_000)
	formatted := formatUPnPTime(position)
	if formatted != "02:04:05" {
		t.Fatalf("formatUPnPTime = %q; want 02:04:05", formatted)
	}
	if parsed := parseUPnPTime(formatted + ".000"); parsed != position {
		t.Fatalf("parseUPnPTime = %d; want %d", parsed, position)
	}
}

func TestSOAPValuesReadsNamespacedResponse(t *testing.T) {
	payload := []byte(`<?xml version="1.0"?><s:Envelope xmlns:s="urn:soap"><s:Body><u:GetPositionInfoResponse xmlns:u="urn:upnp"><TrackDuration>01:30:00</TrackDuration><RelTime>00:12:34</RelTime></u:GetPositionInfoResponse></s:Body></s:Envelope>`)
	values := soapValues(payload)
	if values["TrackDuration"] != "01:30:00" || values["RelTime"] != "00:12:34" {
		t.Fatalf("unexpected SOAP values: %#v", values)
	}
}

func TestDIDLMetadataEscapesMediaAndLabels(t *testing.T) {
	metadata := didlMetadata("http://192.168.1.20/movie?a=1&b=2", domain.CastMediaMetadata{
		Title: "A < B", Subtitle: "Eesti & English", ContentType: "video/mp4",
	})
	for _, expected := range []string{"A &lt; B", "Eesti &amp; English", "a=1&amp;b=2", "video/mp4"} {
		if !strings.Contains(metadata, expected) {
			t.Fatalf("DIDL metadata does not contain %q: %s", expected, metadata)
		}
	}
}

func TestNormalizeTransportState(t *testing.T) {
	if state := normalizeTransportState("PAUSED_PLAYBACK"); state != domain.CastStatePaused {
		t.Fatalf("state = %q; want %q", state, domain.CastStatePaused)
	}
	if state := normalizeTransportState("TRANSITIONING"); state != domain.CastStateBuffering {
		t.Fatalf("state = %q; want %q", state, domain.CastStateBuffering)
	}
}
