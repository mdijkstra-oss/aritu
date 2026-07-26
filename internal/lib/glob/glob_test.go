package glob

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
		wantErr error
	}{
		{
			name:    "a literal pattern matches the identical path",
			pattern: "a/one_test.go",
			path:    "a/one_test.go",
			want:    true,
		},
		{
			name:    "** spans several segments",
			pattern: "a/**/*_test.go",
			path:    "a/b/c/three_test.go",
			want:    true,
		},
		{
			name:    "** spans zero segments",
			pattern: "a/**/b",
			path:    "a/b",
			want:    true,
		},
		{
			name:    "a bare ** matches any depth",
			pattern: "**",
			path:    "a/b/c/three_test.go",
			want:    true,
		},
		{
			name:    "* stays inside a single segment",
			pattern: "a/*_test.go",
			path:    "a/one_test.go",
			want:    true,
		},
		{
			name:    "* does not cross a separator",
			pattern: "a/*_test.go",
			path:    "a/b/two_test.go",
			want:    false,
		},
		{
			name:    "a character class admits a listed character",
			pattern: "a/[a-c]_test.go",
			path:    "a/b_test.go",
			want:    true,
		},
		{
			name:    "a character class rejects an unlisted character",
			pattern: "a/[a-c]_test.go",
			path:    "a/z_test.go",
			want:    false,
		},
		{
			name:    "a negated character class rejects the character it excludes",
			pattern: "a/[!b]_test.go",
			path:    "a/b_test.go",
			want:    false,
		},
		{
			name:    "alternatives match either branch",
			pattern: "{a,b}/*_test.go",
			path:    "b/two_test.go",
			want:    true,
		},
		{
			name:    "a path outside the pattern's directory does not match",
			pattern: "a/**/*_test.go",
			path:    "b/two_test.go",
			want:    false,
		},
		{
			name:    "a path the pattern only prefixes does not match",
			pattern: "a/**/*_test.go",
			path:    "a/b/helper.go",
			want:    false,
		},
		{
			name:    "an unterminated character class is malformed",
			pattern: "a/[_test.go",
			path:    "a/b_test.go",
			wantErr: doublestar.ErrBadPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Match(tt.pattern, tt.path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Match(%q, %q) error = %v, want %v", tt.pattern, tt.path, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %t, want %t", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name       string
		patterns   []string
		unreadable string
		want       []string
		wantErr    error
		wantNamed  string
	}{
		{
			name:     "a literal path resolves to that one file",
			patterns: []string{"a/one_test.go"},
			want:     []string{"a/one_test.go"},
		},
		{
			name:     "a single-segment glob stays in its own directory",
			patterns: []string{"a/*_test.go"},
			want:     []string{"a/one_test.go"},
		},
		{
			name:     "** reaches every depth including none",
			patterns: []string{"a/**/*_test.go"},
			want:     []string{"a/b/c/three_test.go", "a/b/two_test.go", "a/one_test.go"},
		},
		{
			name: "paths the shell already expanded reach the same set as the pattern",
			patterns: []string{
				"a/one_test.go",
				"a/b/two_test.go",
				"a/b/c/three_test.go",
			},
			want: []string{"a/b/c/three_test.go", "a/b/two_test.go", "a/one_test.go"},
		},
		{
			name: "a file several patterns cover is returned once",
			patterns: []string{
				"a/**/*_test.go",
				"a/b/**/*_test.go",
				"a/b/two_test.go",
			},
			want: []string{"a/b/c/three_test.go", "a/b/two_test.go", "a/one_test.go"},
		},
		{
			name:     "a character class narrows which names qualify",
			patterns: []string{"a/[d-o]*_test.go"},
			want:     []string{"a/one_test.go"},
		},
		{
			name:     "a wildcard skips hidden directories the way a shell does",
			patterns: []string{"**/*_test.go"},
			want:     []string{"a/b/c/three_test.go", "a/b/two_test.go", "a/one_test.go"},
		},
		{
			name:     "a hidden directory named in the pattern is searched",
			patterns: []string{".hidden/*_test.go"},
			want:     []string{".hidden/hidden_test.go"},
		},
		{
			name:     "no patterns yield no files",
			patterns: nil,
			want:     nil,
		},
		{
			name:      "a pattern matching nothing is an error naming it",
			patterns:  []string{"a/**/*_spec.rb"},
			wantErr:   errNoMatch,
			wantNamed: "a/**/*_spec.rb",
		},
		{
			name:      "a literal path that does not exist matches nothing",
			patterns:  []string{"a/missing_test.go"},
			wantErr:   errNoMatch,
			wantNamed: "a/missing_test.go",
		},
		{
			name:      "a pattern matching only a directory contributes nothing",
			patterns:  []string{"a/dir_test.go"},
			wantErr:   errNoMatch,
			wantNamed: "a/dir_test.go",
		},
		{
			name: "one empty pattern among matching ones fails the run",
			patterns: []string{
				"a/one_test.go",
				"a/**/*_spec.rb",
				"a/b/two_test.go",
			},
			wantErr:   errNoMatch,
			wantNamed: "a/**/*_spec.rb",
		},
		{
			name:      "a malformed pattern is rejected rather than matched",
			patterns:  []string{"a/[_test.go"},
			wantErr:   doublestar.ErrBadPattern,
			wantNamed: "a/[_test.go",
		},
		{
			name:       "an unreadable directory fails rather than narrowing the set",
			patterns:   []string{"a/**/*_test.go"},
			unreadable: "a/b",
			wantErr:    fs.ErrPermission,
			wantNamed:  "a/**/*_test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unreadable != "" && os.Geteuid() == 0 {
				t.Skip("root reads a directory whatever its mode")
			}
			root := buildTree(t)
			if tt.unreadable != "" {
				sealDir(t, filepath.Join(root, tt.unreadable))
			}
			patterns := rooted(root, tt.patterns)
			before := slices.Clone(patterns)

			got, err := Expand(patterns)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expand() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantNamed != "" && !strings.Contains(err.Error(), tt.wantNamed) {
				t.Errorf("Expand() error = %v, want it to name %q", err, tt.wantNamed)
			}
			if want := rooted(root, tt.want); !slices.Equal(got, want) {
				t.Errorf("Expand() = %v, want %v", got, want)
			}
			if !slices.Equal(patterns, before) {
				t.Errorf("Expand() mutated its input: %v, want %v", patterns, before)
			}
		})
	}
}

var treeFiles = []string{
	"a/one_test.go",
	"a/helper.go",
	"a/b/two_test.go",
	"a/b/notes.md",
	"a/b/c/three_test.go",
	".hidden/hidden_test.go",
}

var treeDirs = []string{"a/dir_test.go"}

func buildTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range treeDirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("creating directory %s: %v", dir, err)
		}
	}
	for _, file := range treeFiles {
		path := filepath.Join(root, file)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating directory for %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte("package a\n"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", file, err)
		}
	}
	return root
}

func sealDir(t *testing.T, dir string) {
	t.Helper()

	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("sealing %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatalf("unsealing %s: %v", dir, err)
		}
	})
}

func rooted(root string, paths []string) []string {
	rooted := make([]string, len(paths))
	for i, path := range paths {
		rooted[i] = filepath.Join(root, path)
	}
	return rooted
}
