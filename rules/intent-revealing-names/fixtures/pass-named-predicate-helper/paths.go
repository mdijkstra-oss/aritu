package archive

import (
	"path/filepath"
	"strings"
)

func isUnderRoot(root, path string) bool {
	return strings.HasPrefix(path, root+string(filepath.Separator))
}

func hasArchiveSuffix(path string) bool {
	return strings.HasSuffix(path, ".tar") || strings.HasSuffix(path, ".zip")
}

func Extractable(root string, paths []string) []string {
	extractable := make([]string, 0, len(paths))
	for _, path := range paths {
		if !isUnderRoot(root, path) {
			continue
		}
		if !hasArchiveSuffix(path) {
			continue
		}
		extractable = append(extractable, path)
	}
	return extractable
}
