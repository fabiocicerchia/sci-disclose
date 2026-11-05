package grid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fabiocicerchia/sci-disclose/internal/fetch"
)

// source is everything the intensity lookup needs from outside the process:
// the API and the on-disk cache in front of it. ResolveIntensity uses
// liveSource; a test can hand resolveIntensity a fake and never touch the
// network or the disk.
type source interface {
	get(endpoint string) ([]byte, error)
	readCache(endpoint string) ([]byte, time.Duration, error)
	writeCache(endpoint string, body []byte)
}

// liveSource is the real API over fetch, cached under the user cache directory.
type liveSource struct{}

func (liveSource) get(endpoint string) ([]byte, error) {
	body, status, err := fetch.Get(endpoint)
	if err != nil {
		return nil, err
	}
	switch {
	case status == http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (1 request per 10s per IP)")
	case status == http.StatusNotFound:
		return nil, fmt.Errorf("no reading published for that country or zone")
	case status != http.StatusOK:
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return body, nil
}

func cacheFile(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "sci-disclose", hex.EncodeToString(sum[:8])+".json")
}

func (liveSource) readCache(endpoint string) ([]byte, time.Duration, error) {
	path := cacheFile(endpoint)
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	body, err := os.ReadFile(path) //nolint:gosec // this tool's own cache file, under its own cache directory
	if err != nil {
		return nil, 0, err
	}
	return body, time.Since(info.ModTime()), nil
}

// writeCache stores one API response for the cache window.
//
// 0700/0600, not 0755/0644: cacheFile falls back to os.TempDir() when
// os.UserCacheDir() fails, and /tmp is shared and world-writable. A
// group-readable directory there lets any other local user enumerate which
// grids this host has queried, and — the part that actually bites — a
// directory they can write to is one they can plant a symlink in, which
// os.WriteFile would then follow. Nothing but this process reads the cache, so
// there is nothing to trade away. gosec G301/G306.
func (liveSource) writeCache(endpoint string, body []byte) {
	path := cacheFile(endpoint)
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	// A cache that cannot be written is a cache miss next time, which is the
	// same outcome as no cache at all.
	_ = os.WriteFile(path, body, 0o600) //nolint:errcheck // see above
}
