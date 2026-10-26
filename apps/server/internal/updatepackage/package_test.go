package updatepackage

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAndExtractAcceptsValidRelease(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "release.zip")
	files := map[string][]byte{
		"kodumedia.exe":         []byte("new server"),
		"kodumedia-updater.exe": []byte("new updater"),
		"VERSION":               []byte("0.2.2\n"),
		"README.md":             []byte("notes"),
	}
	writeTestPackage(t, archive, "kodumedia-0.2.2-windows-x64/", "0.2.2", files, nil)
	stage := filepath.Join(t.TempDir(), "stage")

	prepared, err := ValidateAndExtract(context.Background(), archive, stage, "0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Manifest.Version != "0.2.2" {
		t.Fatalf("unexpected manifest: %+v", prepared.Manifest)
	}
	content, err := os.ReadFile(filepath.Join(stage, "kodumedia.exe"))
	if err != nil || string(content) != "new server" {
		t.Fatalf("payload was not extracted: %q, %v", content, err)
	}
}

func TestValidateAndExtractRejectsUnlistedPayload(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "release.zip")
	files := map[string][]byte{
		"kodumedia.exe":         []byte("new server"),
		"kodumedia-updater.exe": []byte("new updater"),
		"VERSION":               []byte("0.2.2\n"),
	}
	writeTestPackage(t, archive, "", "0.2.2", files, map[string][]byte{"data/catalog.json": []byte("bad")})
	_, err := ValidateAndExtract(context.Background(), archive, filepath.Join(t.TempDir(), "stage"), "0.2.1")
	if err == nil || !IsInvalid(err) {
		t.Fatalf("expected invalid package error, got %v", err)
	}
}

func TestValidateAndExtractRejectsSameOrOlderVersion(t *testing.T) {
	for _, version := range []string{"0.2.1", "0.2.0"} {
		t.Run(version, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "release.zip")
			files := map[string][]byte{
				"kodumedia.exe":         []byte("new server"),
				"kodumedia-updater.exe": []byte("new updater"),
				"VERSION":               []byte(version + "\n"),
			}
			writeTestPackage(t, archive, "", version, files, nil)
			_, err := ValidateAndExtract(context.Background(), archive, filepath.Join(t.TempDir(), "stage"), "0.2.1")
			if err == nil || !IsInvalid(err) {
				t.Fatalf("expected invalid version error, got %v", err)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"0.2.2", "0.2.1", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0", "1.0.0-rc.1", 1},
		{"2.0.0+build.5", "1.9.9", 1},
	}
	for _, test := range tests {
		got, err := CompareVersions(test.left, test.right)
		if err != nil {
			t.Fatalf("compare %s and %s: %v", test.left, test.right, err)
		}
		if got != test.want {
			t.Fatalf("compare %s and %s = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func writeTestPackage(t *testing.T, archivePath, prefix, version string, files, extras map[string][]byte) {
	t.Helper()
	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Product:       ProductName,
		Version:       version,
		Platform:      PlatformWindowsAMD64,
		EntryPoint:    "kodumedia.exe",
		Files:         make([]ManifestFile, 0, len(files)),
	}
	for name, content := range files {
		hash := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, ManifestFile{Path: name, SHA256: hex.EncodeToString(hash[:]), Size: int64(len(content))})
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	write := func(name string, content []byte) {
		entry, err := writer.Create(prefix + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	write(ManifestName, manifestBytes)
	for name, content := range files {
		write(name, content)
	}
	for name, content := range extras {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}
