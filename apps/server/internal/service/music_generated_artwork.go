package service

import (
	"crypto/sha256"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func generateMusicArtworkSVG(item domain.MediaItem, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	key := strings.ToLower(strings.TrimSpace(item.Artist + "\x00" + item.Title + "\x00" + item.Album))
	digest := sha256.Sum256([]byte(key))
	hueA := int(digest[0]) * 360 / 255
	hueB := (hueA + 55 + int(digest[1])%95) % 360
	accent := (hueA + 180) % 360

	titleLines := wrapArtworkText(firstNonEmptyMusicText(item.Title, strings.TrimSuffix(item.File.Name, filepath.Ext(item.File.Name))), 24, 2)
	artistLines := wrapArtworkText(firstNonEmptyMusicText(item.Artist, item.Album, "KoduMedia"), 30, 1)
	version := musicVersionLabel(item.Title + " " + item.Album)

	var titleText strings.Builder
	for index, line := range titleLines {
		y := 610 + index*54
		fmt.Fprintf(&titleText, `<text x="64" y="%d" class="title">%s</text>`, y, html.EscapeString(line))
	}
	var artistText strings.Builder
	for index, line := range artistLines {
		y := 710 + index*36
		fmt.Fprintf(&artistText, `<text x="64" y="%d" class="artist">%s</text>`, y, html.EscapeString(line))
	}
	badge := ""
	if version != "" {
		badge = fmt.Sprintf(`<g transform="translate(64 770)"><rect width="150" height="42" rx="21" fill="hsla(%d,90%%,88%%,.18)" stroke="hsla(%d,90%%,92%%,.42)"/><text x="75" y="28" text-anchor="middle" class="badge">%s</text></g>`, accent, accent, html.EscapeString(strings.ToUpper(version)))
	}

	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="900" height="900" viewBox="0 0 900 900" role="img" aria-label="%s">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="hsl(%d,72%%,23%%)"/>
      <stop offset="1" stop-color="hsl(%d,76%%,11%%)"/>
    </linearGradient>
    <radialGradient id="glow" cx="70%%" cy="20%%" r="70%%">
      <stop offset="0" stop-color="hsla(%d,90%%,70%%,.42)"/>
      <stop offset="1" stop-color="hsla(%d,90%%,55%%,0)"/>
    </radialGradient>
    <filter id="blur"><feGaussianBlur stdDeviation="28"/></filter>
  </defs>
  <rect width="900" height="900" fill="url(#bg)"/>
  <rect width="900" height="900" fill="url(#glow)"/>
  <circle cx="720" cy="175" r="205" fill="hsla(%d,85%%,65%%,.13)" filter="url(#blur)"/>
  <circle cx="720" cy="175" r="142" fill="none" stroke="hsla(%d,90%%,90%%,.26)" stroke-width="3"/>
  <circle cx="720" cy="175" r="54" fill="hsla(%d,95%%,92%%,.76)"/>
  <path d="M106 410 C178 350 214 472 286 410 S394 350 466 410 S574 472 646 410 S754 350 826 410" fill="none" stroke="hsla(%d,95%%,93%%,.72)" stroke-width="14" stroke-linecap="round"/>
  <path d="M106 452 C178 392 214 514 286 452 S394 392 466 452 S574 514 646 452 S754 392 826 452" fill="none" stroke="hsla(%d,95%%,80%%,.24)" stroke-width="8" stroke-linecap="round"/>
  <style>
    .title { fill:#fff; font:700 46px system-ui,-apple-system,"Segoe UI",sans-serif; letter-spacing:-1px; }
    .artist { fill:rgba(255,255,255,.72); font:500 26px system-ui,-apple-system,"Segoe UI",sans-serif; }
    .badge { fill:rgba(255,255,255,.86); font:700 17px system-ui,-apple-system,"Segoe UI",sans-serif; letter-spacing:2px; }
  </style>
  %s
  %s
  %s
  <text x="836" y="838" text-anchor="end" fill="rgba(255,255,255,.32)" font-family="system-ui,-apple-system,'Segoe UI',sans-serif" font-size="18" letter-spacing="3">KODU</text>
</svg>
`, html.EscapeString(item.Artist+" — "+item.Title), hueA, hueB, accent, accent, accent, accent, accent, accent, hueB, titleText.String(), artistText.String(), badge)

	temporary := target + ".tmp"
	if err := os.WriteFile(temporary, []byte(svg), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func wrapArtworkText(value string, maxRunes, maxLines int) []string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return []string{"Tundmatu lugu"}
	}
	words := strings.Fields(value)
	lines := make([]string, 0, maxLines)
	current := ""
	for _, word := range words {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if utf8.RuneCountInString(candidate) <= maxRunes || current == "" {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
		if len(lines) == maxLines-1 {
			break
		}
	}
	if current != "" && len(lines) < maxLines {
		lines = append(lines, current)
	}
	consumed := strings.Join(lines, " ")
	if len(lines) == maxLines && utf8.RuneCountInString(consumed) < utf8.RuneCountInString(value) {
		runes := []rune(lines[len(lines)-1])
		if len(runes) > maxRunes-1 {
			runes = runes[:maxRunes-1]
		}
		lines[len(lines)-1] = strings.TrimSpace(string(runes)) + "…"
	}
	return lines
}

func musicVersionLabel(value string) string {
	tokens := musicVersionTokens(value)
	for _, candidate := range []string{"bootleg", "mashup", "remix", "rework", "vip", "edit", "live", "acoustic"} {
		if _, ok := tokens[candidate]; ok {
			return candidate
		}
	}
	return ""
}
