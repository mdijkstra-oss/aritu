package glob

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

// zsh expands ** and bash without globstar does not, so a quoted pattern and a
// list the shell already expanded must reach the same set; every shell skips
// hidden files, and wildcards here do too.
func Expand(patterns []string) ([]string, error) {
	return expand(patterns, refuseEmpty)
}

func ExpandGenerated(patterns []string) ([]string, error) {
	return expand(patterns, tolerateEmpty)
}

func Match(pattern, path string) (bool, error) {
	return doublestar.Match(pattern, path)
}

func IsValid(pattern string) bool {
	return doublestar.ValidatePattern(pattern)
}

func CheckAll(patterns []string) error {
	for _, pattern := range patterns {
		if !IsValid(pattern) {
			return fmt.Errorf("pattern %q: syntax error", pattern)
		}
	}
	return nil
}

// MatchesAny panics on a pattern that does not parse, so it is for callers that
// ran CheckAll when the patterns were read and would have nothing to do with an
// error here.
func MatchesAny(patterns []string, path string) bool {
	return slices.ContainsFunc(patterns, func(pattern string) bool {
		matched, err := Match(pattern, path)
		if err != nil {
			panic(fmt.Sprintf("pattern %q reached matching without being validated: %v", pattern, err))
		}
		return matched
	})
}

func Rooted(base, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

var expandOptions = []doublestar.GlobOption{
	doublestar.WithFilesOnly(),
	doublestar.WithNoHidden(),
	doublestar.WithFailOnIOErrors(),
}

var errNoMatch = errors.New("matched no files")

func expand(patterns []string, onEmpty func(pattern string) error) ([]string, error) {
	files := make([]string, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))

	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern, expandOptions...)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			if err := onEmpty(pattern); err != nil {
				return nil, err
			}
			continue
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

func refuseEmpty(pattern string) error {
	return fmt.Errorf("pattern %q: %w", pattern, errNoMatch)
}

func tolerateEmpty(string) error {
	return nil
}
