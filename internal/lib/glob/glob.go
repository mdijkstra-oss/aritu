package glob

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

// Expand resolves patterns into the deduplicated, sorted set of existing files they
// match. A pattern carrying no metacharacters is taken as a literal path.
//
// Matching does not depend on the calling shell. zsh expands ** and bash without
// globstar does not, so a quoted pattern and a list of paths the shell already
// expanded have to reach the same set. Wildcards skip hidden files and directories
// for that same reason: every shell does.
//
// Directories never contribute, and an unreadable one fails the whole expansion
// rather than quietly narrowing it.
//
// A pattern matching nothing is an error naming that pattern: silently succeeding
// over an empty set is how a hook reports green because its path was wrong.
func Expand(patterns []string) ([]string, error) {
	files := make([]string, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))

	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern, expandOptions...)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("pattern %q: %w", pattern, errNoMatch)
		}
		for _, match := range matches {
			if _, isKnown := seen[match]; isKnown {
				continue
			}
			seen[match] = struct{}{}
			files = append(files, match)
		}
	}

	slices.Sort(files)
	return files, nil
}

// Match reports whether a path satisfies a pattern, including ** across segments.
func Match(pattern, path string) (bool, error) {
	return doublestar.Match(pattern, path)
}

var expandOptions = []doublestar.GlobOption{
	doublestar.WithFilesOnly(),
	doublestar.WithNoHidden(),
	doublestar.WithFailOnIOErrors(),
}

var errNoMatch = errors.New("matched no files")
