package buildinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionIsReleaseSemanticVersion(t *testing.T) {
	version := Version()
	if version == "dev" {
		t.Fatal("embedded version must never be dev")
	}
	if !semanticVersion.MatchString(version) {
		t.Fatalf("embedded version is not valid SemVer: %q", version)
	}
}

func TestBuildMarkerMatchesVersion(t *testing.T) {
	want := "KODUMEDIA_VERSION=" + Version()
	if got := strings.TrimSpace(rawMarker); got != want {
		t.Fatalf("embedded build marker %q does not match %q", got, want)
	}
}

func TestEmbeddedVersionMatchesRepositoryVersion(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", ".."))
	contents, err := os.ReadFile(filepath.Join(repositoryRoot, "VERSION"))
	if err != nil {
		t.Fatalf("read repository VERSION: %v", err)
	}
	if got, want := Version(), strings.TrimSpace(string(contents)); got != want {
		t.Fatalf("embedded version %q does not match repository VERSION %q", got, want)
	}
}
