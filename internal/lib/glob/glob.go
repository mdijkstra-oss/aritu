package glob

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

// Expand resolves patterns somebody typed into the deduplicated, sorted set of
// existing files they match. A pattern carrying no metacharacters is taken as a
// literal path.
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
	return expand(patterns, refuseEmpty)
}

// ExpandGenerated resolves patterns aritu derived rather than ones somebody typed,
// and tolerates one matching nothing.
//
// Expand's strictness is about a path a person got wrong, and that reasoning does
// not reach a generated pattern: a built-in kind of file spans every ecosystem
// aritu knows and a repository holds one or two of them, so the patterns for the
// rest match nothing and always will.
func ExpandGenerated(patterns []string) ([]string, error) {
	return expand(patterns, tolerateEmpty)
}

// Match reports whether a path satisfies a pattern, including ** across segments.
func Match(pattern, path string) (bool, error) {
	return doublestar.Match(pattern, path)
}

// IsValid reports whether a pattern is well formed, so that a repository which
// wrote a malformed one is told where it is rather than left with a pattern that
// matches nothing.
func IsValid(pattern string) bool {
	return doublestar.ValidatePattern(pattern)
}

// Rooted resolves a relative path or pattern against base and leaves an absolute
// one as written, so that a pattern and a path can be compared in one frame. An
// empty path stays empty: nothing was written, so nothing resolves.
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

// expand walks the patterns once, asking onEmpty what an empty match means. The
// two callers differ in that answer alone, and in nothing else.
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
