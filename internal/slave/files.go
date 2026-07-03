package slave

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

// EnsureTestFiles creates test files in dir if they do not exist.
// Only sizes listed in sizes are created; an empty slice uses all defaults.
func EnsureTestFiles(dir string, sizes []shared.FileSize) error {
	if len(sizes) == 0 {
		sizes = shared.DefaultFileSizes
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, fs := range sizes {
		path := filepath.Join(dir, fs.Name+".bin")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := createFile(path, fs.Size); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
	}
	return nil
}

func createFile(path string, size int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Write repeating zero-ish pattern to avoid sparse-file confusion on all filesystems.
	const blockSize = 64 * 1024
	block := make([]byte, blockSize)
	for i := range block {
		block[i] = byte(i % 256)
	}
	var written int64
	for written < size {
		n := blockSize
		if size-written < int64(blockSize) {
			n = int(size - written)
		}
		if _, err := f.Write(block[:n]); err != nil {
			return err
		}
		written += int64(n)
	}
	return nil
}

// ParseFileSize parses a file size name into bytes using the allowed sizes.
// An empty allowed slice uses all default sizes.
func ParseFileSize(s string, allowed []shared.FileSize) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".BIN")
	if len(allowed) == 0 {
		allowed = shared.DefaultFileSizes
	}
	for _, fs := range allowed {
		if s == fs.Name {
			return fs.Size, nil
		}
	}
	return 0, fmt.Errorf("unknown file size: %s", s)
}

// ServeFile streams a test file of the requested size.
// Only sizes listed in allowed are served; an empty slice allows all defaults.
func ServeFile(w http.ResponseWriter, r *http.Request, sizeName, dir string, allowed []shared.FileSize) {
	size, err := ParseFileSize(sizeName, allowed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.bin\"", sizeName))
	w.WriteHeader(http.StatusOK)
	// Stream repeating block so we do not need the actual file on disk.
	const blockSize = 64 * 1024
	block := make([]byte, blockSize)
	for i := range block {
		block[i] = byte(i % 256)
	}
	var sent int64
	for sent < size {
		n := blockSize
		if size-sent < blockSize {
			n = int(size - sent)
		}
		if _, err := w.Write(block[:n]); err != nil {
			return
		}
		sent += int64(n)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	// Suppress unused dir in this implementation.
	_ = dir
}
