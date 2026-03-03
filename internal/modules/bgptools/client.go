package bgptools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *service) fetchIfNeeded(ctx context.Context, url, destination string, maxAge time.Duration) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return false, fmt.Errorf("failed to create cache directory: %w", err)
	}

	existing := fileExists(destination)
	if existing && !isFileOlderThan(destination, maxAge) {
		return false, nil
	}

	if err := s.downloadFile(ctx, url, destination); err != nil {
		if existing {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *service) downloadFile(ctx context.Context, url, destination string) error {
	tmp := destination + ".tmp"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request %q: %w", url, err)
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed for %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected HTTP %d for %q (%s)", resp.StatusCode, url, string(body))
	}

	file, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to create temp cache file: %w", err)
	}

	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to write temp cache file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to close temp cache file: %w", closeErr)
	}

	if err := os.Rename(tmp, destination); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to finalize cache file: %w", err)
	}

	return nil
}

func isFileOlderThan(path string, maxAge time.Duration) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return true
	}

	return time.Since(stat.ModTime()) > maxAge
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
