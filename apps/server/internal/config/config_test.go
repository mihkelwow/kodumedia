package config

import "testing"

func TestLoadUsesEnglishFirstRunLanguageDefaults(t *testing.T) {
	t.Setenv("MEDIA_SERVER_METADATA_LANGUAGE", "")
	t.Setenv("MEDIA_SERVER_SUBTITLE_LANGUAGES", "")

	loaded := Load()
	if loaded.InitialSettings.Metadata.Language != "en-US" {
		t.Fatalf("metadata language = %q, want en-US", loaded.InitialSettings.Metadata.Language)
	}
	preferred := loaded.InitialSettings.Subtitles.PreferredLanguages
	if len(preferred) != 1 || preferred[0] != "en" {
		t.Fatalf("subtitle languages = %#v, want []string{\"en\"}", preferred)
	}
}
