package gitignore

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// buildRepository commits one file that an ignore rule also matches, which is the
// case the index decides: git does not ignore what it is already tracking.
func buildRepository(t *testing.T) (root string, at func(...string) string) {
	t.Helper()

	root = t.TempDir()
	at = func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }

	run(t, root, "init", "-q")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "user.name", "test")
	writeFile(t, at(".gitignore"))
	if err := os.WriteFile(at(".gitignore"), []byte("node_modules/\n*.gen.go\n"), 0o644); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
	writeFile(t, at("node_modules", "react", "index.js"))
	writeFile(t, at("src", "alpha.go"))
	writeFile(t, at("src", "client.gen.go"))
	run(t, root, "add", "-f", "src/client.gen.go", ".gitignore")
	run(t, root, "commit", "-qm", "committed generated code on purpose")

	return root, at
}

func TestIgnored(t *testing.T) {
	root, at := buildRepository(t)

	ignored, err := Ignored(root, []string{
		at("node_modules", "react", "index.js"),
		at("src", "alpha.go"),
		at("src", "client.gen.go"),
	})
	if err != nil {
		t.Fatalf("Ignored() error = %v, want none", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "an untracked path a rule matches is ignored",
			path: at("node_modules", "react", "index.js"),
			want: true,
		},
		{
			name: "a path no rule matches is not",
			path: at("src", "alpha.go"),
		},
		{
			name: "neither is a tracked path, which git is not ignoring however it reads",
			path: at("src", "client.gen.go"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, got := ignored[tc.path]; got != tc.want {
				t.Errorf("Ignored()[%q] = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestIgnoredWhenGitHasNothingToSay(t *testing.T) {
	root, at := buildRepository(t)

	tests := []struct {
		name  string
		dir   string
		paths []string
	}{
		{
			name:  "a directory holding no repository ignores nothing",
			dir:   t.TempDir(),
			paths: []string{filepath.Join(t.TempDir(), "node_modules", "react", "index.js")},
		},
		{
			name:  "a repository ignoring none of the paths is not a failure",
			dir:   root,
			paths: []string{at("src", "alpha.go")},
		},
		{
			name: "no paths asks git nothing",
			dir:  root,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ignored, err := Ignored(tc.dir, tc.paths)
			if err != nil {
				t.Fatalf("Ignored() error = %v, want none", err)
			}
			if len(ignored) != 0 {
				t.Errorf("Ignored() = %v, want nothing ignored", slices.Sorted(maps.Keys(ignored)))
			}
		})
	}
}
