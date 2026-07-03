package shared

import (
	"fmt"
	"strings"
)

// FileSize describes a single test file size.
type FileSize struct {
	Name string
	Size int64
}

// DefaultFileSizes is the full list of available test file sizes.
var DefaultFileSizes = []FileSize{
	{"1M", 1 << 20},
	{"10M", 10 << 20},
	{"100M", 100 << 20},
	{"1G", 1 << 30},
	{"10G", 10 << 30},
	{"100G", 100 << 30},
}

// FileSizes is kept for backwards compatibility. Use DefaultFileSizes or
// FilterFileSizes in new code.
var FileSizes = DefaultFileSizes

// ParseFileSizesList parses a comma-separated list of file size names.
// An empty string returns all default sizes.
func ParseFileSizesList(s string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		names := make([]string, 0, len(DefaultFileSizes))
		for _, fs := range DefaultFileSizes {
			names = append(names, fs.Name)
		}
		return names, nil
	}
	parts := strings.Split(s, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(strings.ToUpper(p))
		found := false
		for _, fs := range DefaultFileSizes {
			if fs.Name == name {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown file size: %s", name)
		}
		names = append(names, name)
	}
	return names, nil
}

// FilterFileSizes returns the FileSize entries matching the requested names.
// An empty names slice returns all default sizes.
func FilterFileSizes(names []string) []FileSize {
	if len(names) == 0 {
		return DefaultFileSizes
	}
	allowed := make(map[string]struct{}, len(names))
	for _, n := range names {
		allowed[strings.ToUpper(strings.TrimSpace(n))] = struct{}{}
	}
	filtered := make([]FileSize, 0, len(names))
	for _, fs := range DefaultFileSizes {
		if _, ok := allowed[fs.Name]; ok {
			filtered = append(filtered, fs)
		}
	}
	return filtered
}

// TestFiles returns metadata for the available test files served by a slave.
// If names is empty, all default sizes are returned.
func TestFiles(baseURL string, names []string) []TestFile {
	sizes := FilterFileSizes(names)
	files := make([]TestFile, 0, len(sizes))
	for _, fs := range sizes {
		files = append(files, TestFile{
			Name: fs.Name + ".bin",
			Size: fs.Name,
			URL:  baseURL + "/files/" + fs.Name,
		})
	}
	return files
}
