package webui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestHandlerServesIndexWithoutCanonicalRedirect(t *testing.T) {
	handler := Handler()
	for _, path := range []string{"/", "/library/library-movies", "/settings"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("path %s: expected 200, got %d", path, response.Code)
		}
		if location := response.Header().Get("Location"); location != "" {
			t.Fatalf("path %s: unexpected redirect to %s", path, location)
		}
		if !strings.Contains(response.Body.String(), `<div id="root"></div>`) {
			t.Fatalf("path %s: response is not the SPA index", path)
		}
	}
}

func TestEmbeddedUIFoundationAssets(t *testing.T) {
	index, err := embedded.ReadFile("dist/index.html")
	if err != nil {
		t.Fatalf("read embedded index: %v", err)
	}
	markup := string(index)
	for _, expected := range []string{`<meta name="theme-color" content="#171719" />`} {
		if !strings.Contains(markup, expected) {
			t.Fatalf("embedded index missing UI foundation asset %q", expected)
		}
	}

	scriptMatch := regexp.MustCompile(`src="/([^"]+\.js)"`).FindStringSubmatch(markup)
	stylesheetMatch := regexp.MustCompile(`href="/([^"]+\.css)"`).FindStringSubmatch(markup)
	if len(scriptMatch) != 2 || len(stylesheetMatch) != 2 {
		t.Fatalf("embedded index must reference production JavaScript and CSS: %s", markup)
	}
	for _, asset := range []string{scriptMatch[1], stylesheetMatch[1]} {
		if _, err := embedded.ReadFile("dist/" + asset); err != nil {
			t.Fatalf("embedded UI asset missing: %s: %v", asset, err)
		}
	}
	application, err := embedded.ReadFile("dist/" + scriptMatch[1])
	if err != nil {
		t.Fatalf("read embedded UI application: %v", err)
	}
	app := string(application)
	for _, expected := range []string{
		`kodumedia.trailer-failures.v1`,
		`kodumedia.trailer-failures.v2`,
	} {
		if !strings.Contains(app, expected) {
			t.Fatalf("embedded UI application missing corrected hero fallback marker %q", expected)
		}
	}
	stylesheet, err := embedded.ReadFile("dist/" + stylesheetMatch[1])
	if err != nil {
		t.Fatalf("read embedded UI stylesheet: %v", err)
	}
	css := string(stylesheet)
	for _, expected := range []string{
		`--motion-panel:`,
		`--ease-out:`,
		`.ui-dialog-layer`,
		`.ui-popover--bottom-start`,
		`[data-pressable=true]:active:not(:disabled)`,
		`.cast-list--cinematic::-webkit-scrollbar`,
		`.ambient-backdrop__layer--visible`,
		`.ui-dialog-layer.details-layer .ui-dialog-surface`,
		`grid-template-columns:minmax(260px,1.35fr)`,
		`content-visibility:auto`,
		`--focus-shadow-inset:`,
		`touch-action:pan-x pan-y pinch-zoom`,
		`.media-card:focus-visible{box-shadow:none}`,
	} {
		if !strings.Contains(css, expected) {
			t.Fatalf("embedded UI stylesheet missing %q", expected)
		}
	}
	if strings.Contains(css, "transition:all") {
		t.Fatal("embedded UI foundation must not use transition: all")
	}

	for _, legacy := range []string{"featured-hero-bootstrap.js", "featured-hero-bootstrap.css"} {
		if _, err := embedded.ReadFile("dist/" + legacy); err == nil {
			t.Fatalf("legacy embedded hero layer still exists: %s", legacy)
		}
		if strings.Contains(markup, legacy) {
			t.Fatalf("embedded index still loads legacy hero layer: %s", legacy)
		}
	}

	for _, deprecated := range []string{"home-featured", "initial-setup", "series-themes"} {
		if strings.Contains(markup, deprecated) {
			t.Fatalf("embedded index still loads deprecated runtime layer %q", deprecated)
		}
		for _, extension := range []string{".js", ".css"} {
			if _, err := embedded.ReadFile("dist/" + deprecated + extension); err == nil {
				t.Fatalf("deprecated embedded runtime asset still exists: %s%s", deprecated, extension)
			}
		}
	}
	for _, expected := range []string{
		`.featured-hero__arrow{width:34px;height:34px`,
		`.featured-hero__dot{width:7px;height:7px`,
		`.featured-hero__dot[aria-selected=true]{background:#fff;width:22px`,
	} {
		if !strings.Contains(css, expected) {
			t.Fatalf("embedded browsing stylesheet missing %q", expected)
		}
	}
	for _, forbidden := range []string{
		`.featured-hero__dot::before`,
		`.featured-hero__dot-indicator`,
		`.featured-hero__dot{width:32px`,
		`.featured-hero__dot{height:var(--tap-target)`,
	} {
		if strings.Contains(css, forbidden) {
			t.Fatalf("embedded hero carousel contains forbidden resized control %q", forbidden)
		}
	}
}
