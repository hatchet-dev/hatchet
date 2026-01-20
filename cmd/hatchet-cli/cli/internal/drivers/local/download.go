package local

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

const (
	GitHubReleaseURL = "https://github.com/hatchet-dev/hatchet/releases/download"
	DownloadTimeout  = 10 * time.Minute
)

// BinaryDownloader handles downloading and caching Hatchet binaries
type BinaryDownloader struct {
	cacheDir string // {cacheDir}/hatchet/bin/
}

// NewBinaryDownloader creates a new binary downloader
func NewBinaryDownloader() (*BinaryDownloader, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	binCacheDir := filepath.Join(cacheDir, "hatchet", "bin")
	if err := os.MkdirAll(binCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create binary cache directory: %w", err)
	}

	return &BinaryDownloader{
		cacheDir: binCacheDir,
	}, nil
}

// EnsureBinary ensures the specified binary is available, downloading if necessary.
// Returns the path to the binary.
func (bd *BinaryDownloader) EnsureBinary(ctx context.Context, name, version string) (string, error) {
	// Normalize version to include 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Path where the binary will be cached
	versionDir := filepath.Join(bd.cacheDir, version)
	binaryPath := filepath.Join(versionDir, name)

	// Check if binary already exists and is executable
	if info, err := os.Stat(binaryPath); err == nil && !info.IsDir() {
		fmt.Println(styles.SuccessMessage(fmt.Sprintf("Using cached %s", name)))
		return binaryPath, nil
	}

	// Create version directory
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create version directory: %w", err)
	}

	// Download the binary
	if err := bd.downloadBinary(ctx, name, version, binaryPath); err != nil {
		return "", err
	}

	return binaryPath, nil
}

// downloadBinary downloads a specific binary from GitHub releases
func (bd *BinaryDownloader) downloadBinary(ctx context.Context, name, version, destPath string) error {
	archiveName := bd.getArchiveName(name, version)
	url := fmt.Sprintf("%s/%s/%s", GitHubReleaseURL, version, archiveName)

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Downloading %s %s...", name, version)))
	fmt.Println(styles.Muted.Render("  " + url))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: DownloadTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// Create a temporary file for the download
	tmpDir := filepath.Dir(destPath)
	tmpFile, err := os.CreateTemp(tmpDir, "download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download binary: %w", err)
	}
	tmpFile.Close()

	// Extract binary from archive
	if err := bd.extractBinary(tmpPath, name, destPath); err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	fmt.Println(styles.SuccessMessage(fmt.Sprintf("Downloaded %s", name)))
	return nil
}

// extractBinary extracts a binary from a tar.gz archive
func (bd *BinaryDownloader) extractBinary(archivePath, binaryName, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the binary file (might be at root or in a subdirectory)
		baseName := filepath.Base(header.Name)
		if baseName == binaryName && header.Typeflag == tar.TypeReg {
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract binary: %w", err)
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}

// getArchiveName returns the archive name for a binary based on OS/arch
func (bd *BinaryDownloader) getArchiveName(name, version string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch names to release arch names
	archName := goarch
	if goarch == "amd64" {
		archName = "x86_64"
	}

	// Map Go OS names to release OS names (capitalize first letter)
	osName := capitalizeFirst(goos)

	// Format: hatchet-api_v0.73.10_Darwin_arm64.tar.gz
	return fmt.Sprintf("%s_%s_%s_%s.tar.gz", name, version, osName, archName)
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// VerifyChecksum verifies the SHA256 checksum of a file
func (bd *BinaryDownloader) VerifyChecksum(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// CacheDir returns the binary cache directory
func (bd *BinaryDownloader) CacheDir() string {
	return bd.cacheDir
}

// CleanOldVersions removes cached binaries for versions other than the current one
func (bd *BinaryDownloader) CleanOldVersions(currentVersion string) error {
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}

	entries, err := os.ReadDir(bd.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != currentVersion {
			oldPath := filepath.Join(bd.cacheDir, entry.Name())
			fmt.Println(styles.Muted.Render(fmt.Sprintf("Removing old cached binaries: %s", oldPath)))
			if err := os.RemoveAll(oldPath); err != nil {
				fmt.Println(styles.Muted.Render(fmt.Sprintf("Warning: failed to remove %s: %v", oldPath, err)))
			}
		}
	}

	return nil
}
