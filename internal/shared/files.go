package shared

// FileSizes maps human-readable file sizes to bytes.
var FileSizes = []struct {
	Name string
	Size int64
}{
	{"1M", 1 << 20},
	{"10M", 10 << 20},
	{"100M", 100 << 20},
	{"1G", 1 << 30},
	{"10G", 10 << 30},
	{"100G", 100 << 30},
}

// TestFiles returns metadata for the available test files served by a slave.
func TestFiles(baseURL string) []TestFile {
	files := make([]TestFile, 0, len(FileSizes))
	for _, fs := range FileSizes {
		files = append(files, TestFile{
			Name: fs.Name + ".bin",
			Size: fs.Name,
			URL:  baseURL + "/files/" + fs.Name,
		})
	}
	return files
}
