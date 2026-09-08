package pocketbase

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const DefaultVersion = "0.40.3"

func AssetPlatform() (string, error) {
	return assetPlatform(runtime.GOOS, runtime.GOARCH)
}

func assetPlatform(goos, goarch string) (string, error) {
	if goos == "darwin" && (goarch == "amd64" || goarch == "arm64") {
		return goos + "_" + goarch, nil
	}
	if goos == "windows" && (goarch == "amd64" || goarch == "arm64") {
		return goos + "_" + goarch, nil
	}
	switch goos + "/" + goarch {
	case "linux/amd64", "linux/arm64", "linux/ppc64le", "linux/s390x":
		return goos + "_" + goarch, nil
	case "linux/arm":
		return "linux_armv7", nil
	default:
		return "", fmt.Errorf("unsupported PocketBase platform %s/%s", goos, goarch)
	}
}

func Ensure(root, version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return "", errors.New("PocketBase version is empty")
	}
	platform, err := AssetPlatform()
	if err != nil {
		return "", err
	}
	binaryName := "pocketbase"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(root, ".pb", "pocketbase", version, platform, binaryName)
	if info, err := os.Stat(binaryPath); err == nil && !info.IsDir() {
		return binaryPath, nil
	}

	assetName := fmt.Sprintf("pocketbase_%s_%s.zip", version, platform)
	baseURL := fmt.Sprintf("https://github.com/pocketbase/pocketbase/releases/download/v%s", version)
	archiveURL := baseURL + "/" + assetName
	checksumsURL := baseURL + "/checksums.txt"
	archive, err := download(archiveURL)
	if err != nil {
		return "", err
	}
	checksums, err := download(checksumsURL)
	if err != nil {
		return "", err
	}
	expected, err := checksumFor(string(checksums), assetName)
	if err != nil {
		return "", err
	}
	actualBytes := sha256.Sum256(archive)
	if actual := hex.EncodeToString(actualBytes[:]); !strings.EqualFold(expected, actual) {
		return "", fmt.Errorf("checksum mismatch for %s", assetName)
	}

	destination := filepath.Dir(binaryPath)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(destination, ".pocketbase-*.zip")
	if err != nil {
		return "", err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(archive); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := extractBinary(temporaryName, binaryPath, binaryName); err != nil {
		return "", err
	}
	return binaryPath, nil
}

func Validate(root, version string) error {
	binaryPath, err := Ensure(root, version)
	if err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "pocketbase-pockethost-validate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	dataDir := filepath.Join(temporary, "pb_data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	args := []string{
		"migrate", "up", "--dev",
		"--dir", dataDir,
		"--hooksDir", filepath.Join(root, "pb_hooks"),
		"--migrationsDir", filepath.Join(root, "pb_migrations"),
		"--publicDir", filepath.Join(root, "pb_public"),
	}
	command := exec.Command(binaryPath, args...)
	command.Dir = root
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("PocketBase migration validation failed: %w", err)
	}
	return nil
}

func download(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %s", url, response.Status)
	}
	return io.ReadAll(response.Body)
}

func checksumFor(contents, asset string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && filepath.Base(fields[len(fields)-1]) == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", asset)
}

func extractBinary(archivePath, binaryPath, binaryName string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, file := range archive.File {
		if filepath.Base(file.Name) != binaryName {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		temporary, err := os.CreateTemp(filepath.Dir(binaryPath), ".pocketbase-*")
		if err != nil {
			reader.Close()
			return err
		}
		_, copyErr := io.Copy(temporary, reader)
		readerErr := reader.Close()
		closeErr := temporary.Close()
		if copyErr != nil {
			return copyErr
		}
		if readerErr != nil {
			return readerErr
		}
		if closeErr != nil {
			return closeErr
		}
		if err := os.Chmod(temporary.Name(), 0o755); err != nil {
			return err
		}
		return os.Rename(temporary.Name(), binaryPath)
	}
	return fmt.Errorf("%s not found in archive", binaryName)
}
