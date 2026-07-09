package embed

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/version"
)

const dashboardReleaseBaseURL = "https://github.com/hatchet-dev/hatchet/releases/download"

const maxDashboardBundleBytes = 128 << 20

func dashboardTarballURL() string {
	if u := os.Getenv("HATCHET_EMBEDDED_DASHBOARD_URL"); u != "" {
		return u
	}

	v := version.Version
	return fmt.Sprintf("%s/%s/dashboard-%s.tar.gz", dashboardReleaseBaseURL, v, v)
}

func ensureDashboardAssets(ctx context.Context) (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve user cache dir: %w", err)
	}

	dir := filepath.Join(cacheRoot, "hatchet", "embedded-dashboard", version.Version)
	if _, statErr := os.Stat(filepath.Join(dir, "index.html")); statErr == nil {
		return dir, nil
	}

	url := dashboardTarballURL()

	tarball, err := downloadBytes(ctx, url)
	if err != nil {
		return "", fmt.Errorf("download dashboard bundle %s: %w", url, err)
	}

	if sum, sumErr := downloadBytes(ctx, url+".sha256"); sumErr == nil {
		if err := verifySHA256(tarball, sum); err != nil {
			return "", err
		}
	}

	if err := extractTarGz(tarball, dir); err != nil {
		return "", fmt.Errorf("extract dashboard bundle: %w", err)
	}

	return dir, nil
}

func downloadBytes(ctx context.Context, url string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxDashboardBundleBytes))
}

func verifySHA256(data, sumFile []byte) error {
	want := strings.TrimSpace(string(sumFile))
	if i := strings.IndexAny(want, " \t"); i >= 0 {
		want = want[:i]
	}

	got := sha256.Sum256(data)
	if !strings.EqualFold(want, hex.EncodeToString(got[:])) {
		return fmt.Errorf("dashboard bundle checksum mismatch: got %s, want %s", hex.EncodeToString(got[:]), want)
	}

	return nil
}

func extractTarGz(data []byte, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp(filepath.Dir(destDir), ".download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp) //nolint:errcheck

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gz.Close() //nolint:errcheck

	tr := tar.NewReader(io.LimitReader(gz, maxDashboardBundleBytes))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := path.Clean(strings.TrimPrefix(hdr.Name, "./"))
		if name == "." || name == "" {
			continue
		}
		if strings.HasPrefix(name, "..") || strings.Contains(name, "../") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe path in dashboard bundle: %q", hdr.Name)
		}

		target := filepath.Join(tmp, filepath.FromSlash(name))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, io.LimitReader(tr, maxDashboardBundleBytes)); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}

	if err := os.Rename(tmp, destDir); err != nil {
		if _, statErr := os.Stat(filepath.Join(destDir, "index.html")); statErr == nil {
			return nil
		}
		return err
	}

	return nil
}

func dashboardHandler(fsys fs.FS) (http.Handler, error) {
	indexBytes, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, fmt.Errorf("dashboard assets are missing index.html: %w", err)
	}

	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if reqPath != "" {
			if _, statErr := fs.Stat(fsys, reqPath); statErr == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	}), nil
}
