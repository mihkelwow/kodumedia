package buildinfo

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

// rawVersion is generated from the repository VERSION file before every build.
// Keeping the value in an embedded resource makes the executable version
// independent from linker flags and the process working directory.
//
//go:embed version.txt
var rawVersion string

//go:embed marker.txt
var rawMarker string

var semanticVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// Version returns the immutable version compiled into the executable.
// It panics on invalid build metadata so a broken release fails during startup
// instead of silently reporting "dev" and triggering an updater rollback.
func Version() string {
	version := strings.TrimSpace(rawVersion)
	if !semanticVersion.MatchString(version) {
		panic(fmt.Sprintf("invalid embedded KoduMedia version %q", version))
	}
	marker := strings.TrimSpace(rawMarker)
	expectedMarker := "KODUMEDIA_VERSION=" + version
	if marker != expectedMarker {
		panic(fmt.Sprintf("invalid embedded KoduMedia build marker %q, expected %q", marker, expectedMarker))
	}
	return version
}
