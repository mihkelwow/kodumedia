package updatepackage

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

const (
	ManifestName          = "release-manifest.json"
	ManifestSchemaVersion = 1
	ProductName           = "KoduMedia"
	PlatformWindowsAMD64  = "windows-amd64"

	maxManifestBytes = 1 << 20
	maxFiles         = 2048
	maxExtracted     = int64(512 << 20)
	maxSingleFile    = int64(256 << 20)
)

type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	Product       string         `json:"product"`
	Version       string         `json:"version"`
	Platform      string         `json:"platform"`
	EntryPoint    string         `json:"entryPoint"`
	Files         []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type PreparedPackage struct {
	Manifest Manifest
	StageDir string
}

func IsInvalid(err error) bool { return errors.Is(err, domain.ErrInvalidInput) }

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidInput, fmt.Sprintf(format, args...))
}

// ValidateAndExtract validates a complete release package before exposing any
// payload in the staging directory. The staging directory must not already
// contain files.
func ValidateAndExtract(ctx context.Context, archivePath, stageDir, currentVersion string) (PreparedPackage, error) {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return PreparedPackage{}, invalid("uuenduspaketti ei saa ZIP-failina avada: %v", err)
	}
	defer archive.Close()
	if len(archive.File) == 0 || len(archive.File) > maxFiles+64 {
		return PreparedPackage{}, invalid("uuenduspaketi failide arv ei ole lubatud")
	}

	manifestFile, prefix, err := locateManifest(archive.File)
	if err != nil {
		return PreparedPackage{}, err
	}
	manifest, err := readManifest(manifestFile)
	if err != nil {
		return PreparedPackage{}, err
	}
	if err := validateManifest(manifest, currentVersion); err != nil {
		return PreparedPackage{}, err
	}

	listed := make(map[string]ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		key := strings.ToLower(file.Path)
		if _, exists := listed[key]; exists {
			return PreparedPackage{}, invalid("manifest sisaldab sama faili mitu korda: %s", file.Path)
		}
		listed[key] = file
	}

	zipFiles := make(map[string]*zip.File, len(manifest.Files))
	for _, file := range archive.File {
		if err := ctx.Err(); err != nil {
			return PreparedPackage{}, err
		}
		name, err := normalizeZipName(file.Name)
		if err != nil {
			return PreparedPackage{}, err
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return PreparedPackage{}, invalid("sümbollink ei ole paketis lubatud: %s", name)
		}
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				return PreparedPackage{}, invalid("fail asub paketijuurest väljaspool: %s", name)
			}
			name = strings.TrimPrefix(name, prefix)
		}
		if name == ManifestName {
			continue
		}
		if name == "" {
			return PreparedPackage{}, invalid("pakett sisaldab tühja failinime")
		}
		entry, ok := listed[strings.ToLower(name)]
		if !ok || entry.Path != name {
			return PreparedPackage{}, invalid("pakett sisaldab manifestis kirjeldamata faili: %s", name)
		}
		key := strings.ToLower(name)
		if _, exists := zipFiles[key]; exists {
			return PreparedPackage{}, invalid("pakett sisaldab sama faili mitu korda: %s", name)
		}
		zipFiles[key] = file
	}
	if len(zipFiles) != len(manifest.Files) {
		for _, entry := range manifest.Files {
			if _, ok := zipFiles[strings.ToLower(entry.Path)]; !ok {
				return PreparedPackage{}, invalid("manifestis kirjeldatud fail puudub paketist: %s", entry.Path)
			}
		}
	}

	if err := os.RemoveAll(stageDir); err != nil {
		return PreparedPackage{}, fmt.Errorf("clear update stage: %w", err)
	}
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return PreparedPackage{}, fmt.Errorf("create update stage: %w", err)
	}

	ordered := append([]ManifestFile(nil), manifest.Files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, entry := range ordered {
		if err := ctx.Err(); err != nil {
			return PreparedPackage{}, err
		}
		file := zipFiles[strings.ToLower(entry.Path)]
		if err := extractVerified(file, filepath.Join(stageDir, filepath.FromSlash(entry.Path)), entry); err != nil {
			return PreparedPackage{}, err
		}
	}
	versionBytes, err := os.ReadFile(filepath.Join(stageDir, "VERSION"))
	if err != nil {
		return PreparedPackage{}, invalid("VERSION faili ei saanud lugeda: %v", err)
	}
	if strings.TrimSpace(string(versionBytes)) != manifest.Version {
		return PreparedPackage{}, invalid("VERSION faili ja manifesti versioonid ei ühti")
	}
	return PreparedPackage{Manifest: manifest, StageDir: stageDir}, nil
}

func locateManifest(files []*zip.File) (*zip.File, string, error) {
	var found *zip.File
	prefix := ""
	for _, file := range files {
		name, err := normalizeZipName(file.Name)
		if err != nil {
			return nil, "", err
		}
		if file.FileInfo().IsDir() || path.Base(name) != ManifestName {
			continue
		}
		if found != nil {
			return nil, "", invalid("pakett sisaldab mitut %s faili", ManifestName)
		}
		found = file
		prefix = strings.TrimSuffix(name, ManifestName)
	}
	if found == nil {
		return nil, "", invalid("paketist puudub %s", ManifestName)
	}
	return found, prefix, nil
}

func readManifest(file *zip.File) (Manifest, error) {
	if file.UncompressedSize64 > maxManifestBytes {
		return Manifest{}, invalid("manifest on liiga suur")
	}
	reader, err := file.Open()
	if err != nil {
		return Manifest{}, invalid("manifesti ei saa avada: %v", err)
	}
	defer reader.Close()
	limited := io.LimitReader(reader, maxManifestBytes+1)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, invalid("manifest ei ole korrektne JSON: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Manifest{}, invalid("manifest sisaldab üleliigseid andmeid")
	}
	return manifest, nil
}

func validateManifest(manifest Manifest, currentVersion string) error {
	if manifest.SchemaVersion != ManifestSchemaVersion {
		return invalid("manifesti skeemi versioon %d ei ole toetatud", manifest.SchemaVersion)
	}
	if manifest.Product != ProductName {
		return invalid("pakett ei ole KoduMedia väljalase")
	}
	if manifest.Platform != PlatformWindowsAMD64 {
		return invalid("pakett ei ole Windows x64 platvormile")
	}
	if manifest.EntryPoint != "kodumedia.exe" {
		return invalid("paketi käivitusfail ei ole lubatud")
	}
	comparison, err := CompareVersions(manifest.Version, currentVersion)
	if err != nil {
		return invalid("vigane versioon: %v", err)
	}
	if comparison <= 0 {
		return invalid("uuendusversioon %s peab olema uuem kui %s", manifest.Version, currentVersion)
	}
	if len(manifest.Files) == 0 || len(manifest.Files) > maxFiles {
		return invalid("manifesti failide arv ei ole lubatud")
	}
	required := map[string]bool{"kodumedia.exe": false, "kodumedia-updater.exe": false, "VERSION": false}
	var total int64
	for index := range manifest.Files {
		entry := &manifest.Files[index]
		normalized, err := normalizePayloadPath(entry.Path)
		if err != nil {
			return err
		}
		if normalized != entry.Path {
			return invalid("manifesti failitee peab olema normaliseeritud: %s", entry.Path)
		}
		if entry.Size < 0 || entry.Size > maxSingleFile {
			return invalid("faili suurus ei ole lubatud: %s", entry.Path)
		}
		total += entry.Size
		if total > maxExtracted {
			return invalid("paketi kogumaht on liiga suur")
		}
		hash := strings.ToLower(strings.TrimSpace(entry.SHA256))
		decoded, decodeErr := hex.DecodeString(hash)
		if decodeErr != nil || len(decoded) != sha256.Size || hash != entry.SHA256 {
			return invalid("vigane SHA-256 väärtus failil %s", entry.Path)
		}
		if _, ok := required[entry.Path]; ok {
			required[entry.Path] = true
		}
	}
	for file, present := range required {
		if !present {
			return invalid("paketist puudub kohustuslik fail: %s", file)
		}
	}
	return nil
}

func extractVerified(file *zip.File, destination string, expected ManifestFile) error {
	if int64(file.UncompressedSize64) != expected.Size {
		return invalid("faili suurus ei ühti manifestiga: %s", expected.Path)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create update directory: %w", err)
	}
	reader, err := file.Open()
	if err != nil {
		return invalid("faili ei saa paketist avada: %s", expected.Path)
	}
	defer reader.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create staged update file: %w", err)
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hasher), io.LimitReader(reader, expected.Size+1))
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("extract update file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close update file: %w", closeErr)
	}
	if written != expected.Size {
		return invalid("faili lahtipakitud suurus ei ühti manifestiga: %s", expected.Path)
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expected.SHA256 {
		return invalid("faili SHA-256 ei ühti manifestiga: %s", expected.Path)
	}
	return nil
}

func normalizeZipName(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || strings.ContainsRune(value, '\x00') {
		return "", invalid("vigane ZIP-failitee")
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", invalid("absoluutne ZIP-failitee ei ole lubatud: %s", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", invalid("ZIP-failitee väljub paketist: %s", value)
	}
	return strings.TrimSuffix(cleaned, "/"), nil
}

func normalizePayloadPath(value string) (string, error) {
	cleaned, err := normalizeZipName(value)
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(cleaned)
	if lower == ".env" || lower == "settings.json" || lower == "catalog.json" || lower == "data" || strings.HasPrefix(lower, "data/") {
		return "", invalid("pakett ei tohi muuta kasutajaandmeid: %s", value)
	}
	return cleaned, nil
}
