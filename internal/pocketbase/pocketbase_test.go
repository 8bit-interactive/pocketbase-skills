package pocketbase

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetPlatform(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"darwin", "arm64", "darwin_arm64"},
		{"darwin", "amd64", "darwin_amd64"},
		{"linux", "amd64", "linux_amd64"},
		{"linux", "arm", "linux_armv7"},
		{"windows", "arm64", "windows_arm64"},
	}
	for _, test := range tests {
		got, err := assetPlatform(test.goos, test.goarch)
		if err != nil {
			t.Errorf("assetPlatform(%s, %s): %v", test.goos, test.goarch, err)
			continue
		}
		if got != test.want {
			t.Errorf("assetPlatform(%s, %s) = %q, want %q", test.goos, test.goarch, got, test.want)
		}
	}
	if _, err := assetPlatform("freebsd", "amd64"); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

func TestChecksumFor(t *testing.T) {
	contents := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.zip\n" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  pocketbase_0.40.3_linux_amd64.zip\n"
	got, err := checksumFor(contents, "pocketbase_0.40.3_linux_amd64.zip")
	if err != nil {
		t.Fatal(err)
	}
	if got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("checksum = %q", got)
	}
	if _, err := checksumFor(contents, "missing.zip"); err == nil {
		t.Fatal("expected missing checksum error")
	}
}

func TestExtractBinary(t *testing.T) {
	temporary := t.TempDir()
	archivePath := filepath.Join(temporary, "pocketbase.zip")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(archiveFile)
	entry, err := archive.Create("pocketbase")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("binary")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(temporary, "out", "pocketbase")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractBinary(archivePath, binaryPath, "pocketbase"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary" {
		t.Fatalf("extracted data = %q", data)
	}
}
