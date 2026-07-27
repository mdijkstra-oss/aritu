package kind

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// repoRoot is where a built-in kind's patterns generate. Membership must not depend
// on it, which is half of what TestCovers pins: a path named on the command line is
// judged the same wherever it sits.
const repoRoot = "/repo"

func TestCovers(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string][]string
		targets  []string
		path     string
		want     bool
	}{
		{
			name:    "a file named as a test is of the tests kind",
			targets: []string{"tests"},
			path:    "/repo/internal/parser_test.go",
			want:    true,
		},
		{
			name:    "the implementation beside it is not",
			targets: []string{"tests"},
			path:    "/repo/internal/parser.go",
		},
		{
			name:    "another ecosystem's naming is read by the same table",
			targets: []string{"tests"},
			path:    "/repo/src/__tests__/parser.ts",
			want:    true,
		},
		{
			name:    "a document is of neither code nor tests",
			targets: []string{"tests", "code"},
			path:    "/repo/README.md",
		},
		{
			name:    "a document is of the docs kind",
			targets: []string{"docs"},
			path:    "/repo/README.md",
			want:    true,
		},
		{
			name:    "so is the other markdown extension",
			targets: []string{"docs"},
			path:    "/repo/docs/guide.mdx",
			want:    true,
		},
		{
			name:    "code covers an implementation file",
			targets: []string{"code"},
			path:    "/repo/internal/parser.go",
			want:    true,
		},
		{
			name:    "code deliberately covers a test file too, because tests have comments",
			targets: []string{"code"},
			path:    "/repo/internal/parser_test.go",
			want:    true,
		},
		{
			name:    "an extension no ecosystem in the table uses is neither",
			targets: []string{"tests", "code"},
			path:    "/repo/script.rb",
		},
		{
			name:    "a rule about several kinds takes any one of them",
			targets: []string{"tests", "docs"},
			path:    "/repo/README.md",
			want:    true,
		},
		{
			name:    "a built-in judges a file outside the repository the same way",
			targets: []string{"tests"},
			path:    "/elsewhere/parser_test.go",
			want:    true,
		},
		{
			name:     "a declared key replaces the built-in patterns",
			declared: map[string][]string{"tests": {"legacy/**/*.go"}},
			targets:  []string{"tests"},
			path:     "/repo/legacy/anything.go",
			want:     true,
		},
		{
			name:     "and replaces its refinement with it, rather than keeping both",
			declared: map[string][]string{"tests": {"legacy/**/*.go"}},
			targets:  []string{"tests"},
			path:     "/repo/internal/parser_test.go",
		},
		{
			name:     "replacing one built-in leaves the others standing",
			declared: map[string][]string{"tests": {"legacy/**/*.go"}},
			targets:  []string{"docs"},
			path:     "/repo/README.md",
			want:     true,
		},
		{
			name:     "a kind the repository invented covers what it says it covers",
			declared: map[string][]string{"migrations": {"db/migrate/**/*.sql"}},
			targets:  []string{"migrations"},
			path:     "/repo/db/migrate/001_users.sql",
			want:     true,
		},
		{
			name:     "and nothing else",
			declared: map[string][]string{"migrations": {"db/migrate/**/*.sql"}},
			targets:  []string{"migrations"},
			path:     "/repo/db/schema.sql",
		},
		{
			name:     "a declared kind is rooted where it was written, so a path elsewhere is not of it",
			declared: map[string][]string{"migrations": {"db/migrate/**/*.sql"}},
			targets:  []string{"migrations"},
			path:     "/elsewhere/db/migrate/001_users.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kinds := resolved(t, tt.declared)

			if got := kinds.Covers(tt.targets, tt.path); got != tt.want {
				t.Errorf("Covers(%v, %q) = %t, want %t", tt.targets, tt.path, got, tt.want)
			}
		})
	}
}

func TestNames(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string][]string
		want     []string
	}{
		{
			name: "the built-ins are known without being declared",
			want: []string{"code", "docs", "tests"},
		},
		{
			name:     "a declared kind widens the vocabulary",
			declared: map[string][]string{"migrations": {"db/**/*.sql"}},
			want:     []string{"code", "docs", "migrations", "tests"},
		},
		{
			name:     "a declared key matching a built-in replaces rather than adds",
			declared: map[string][]string{"tests": {"legacy/**/*.go"}},
			want:     []string{"code", "docs", "tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolved(t, tt.declared).Names(); !slices.Equal(got, tt.want) {
				t.Errorf("Names() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveRefusesAKindThatCouldHoldNothing(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string][]string
		wantErr  string
	}{
		{
			name:     "a kind naming no pattern would hand its rules no file",
			declared: map[string][]string{"migrations": nil},
			wantErr:  "migrations",
		},
		{
			name:     "a malformed pattern is refused where it was written",
			declared: map[string][]string{"migrations": {"db/[migrate/**/*.sql"}},
			wantErr:  "db/[migrate/**/*.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(repoRoot, tt.declared)

			if err == nil {
				t.Fatalf("Resolve(%v) = %v, want an error naming %q", tt.declared, got.Names(), tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Resolve(%v) error = %v, want it to name %q", tt.declared, err, tt.wantErr)
			}
		})
	}
}

func TestCoversPanicsOnAKindNobodyDefined(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Covers did not panic on a kind no repository defined")
		}
	}()

	_ = resolved(t, nil).Covers([]string{"prose"}, "/repo/README.md")
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string][]string
		targets  []string
		want     []string
	}{
		{
			name:    "the tests kind generates over every ecosystem and keeps the test files",
			targets: []string{"tests"},
			want: []string{
				"cmd/main_test.go",
				"internal/parser_test.go",
				"src/__tests__/cart.ts",
				"src/cart.test.ts",
			},
		},
		{
			name:    "the code kind keeps the implementations as well",
			targets: []string{"code"},
			want: []string{
				"cmd/main.go",
				"cmd/main_test.go",
				"internal/parser.go",
				"internal/parser_test.go",
				"src/__tests__/cart.ts",
				"src/cart.test.ts",
				"src/cart.ts",
			},
		},
		{
			name:    "the docs kind keeps the documents",
			targets: []string{"docs"},
			want:    []string{"README.md", "docs/guide.mdx"},
		},
		{
			name:    "several kinds contribute one sorted set, and a file both cover once",
			targets: []string{"docs", "tests"},
			want: []string{
				"README.md",
				"cmd/main_test.go",
				"docs/guide.mdx",
				"internal/parser_test.go",
				"src/__tests__/cart.ts",
				"src/cart.test.ts",
			},
		},
		{
			name:     "a declared kind generates from its own patterns",
			declared: map[string][]string{"migrations": {"db/migrate/**/*.sql"}},
			targets:  []string{"migrations"},
			want:     []string{"db/migrate/001_users.sql"},
		},
		{
			name:     "a declared kind matching nothing yields nothing rather than failing",
			declared: map[string][]string{"styles": {"web/**/*.css"}},
			targets:  []string{"styles"},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := buildTree(t)

			got, err := resolvedAt(t, root, tt.declared).Expand(tt.targets)

			if err != nil {
				t.Fatalf("Expand(%v) error = %v, want none", tt.targets, err)
			}
			if want := rooted(root, tt.want); !slices.Equal(got, want) {
				t.Errorf("Expand(%v) = %v, want %v", tt.targets, got, want)
			}
		})
	}
}

var treeFiles = []string{
	"README.md",
	"docs/guide.mdx",
	"cmd/main.go",
	"cmd/main_test.go",
	"internal/parser.go",
	"internal/parser_test.go",
	"src/cart.ts",
	"src/cart.test.ts",
	"src/__tests__/cart.ts",
	"db/migrate/001_users.sql",
}

func resolved(t *testing.T, declared map[string][]string) Set {
	t.Helper()
	return resolvedAt(t, repoRoot, declared)
}

// resolvedAt roots the declared patterns the way config loading does, so a test
// declares them the way a repository writes them.
func resolvedAt(t *testing.T, base string, declared map[string][]string) Set {
	t.Helper()
	rootedDeclared := make(map[string][]string, len(declared))
	for name, patterns := range declared {
		rootedDeclared[name] = rooted(base, patterns)
	}
	kinds, err := Resolve(base, rootedDeclared)
	if err != nil {
		t.Fatalf("Resolve(%q, %v) error = %v", base, declared, err)
	}
	return kinds
}

func buildTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, file := range treeFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating directory for %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte("contents\n"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", file, err)
		}
	}
	return root
}

func rooted(base string, paths []string) []string {
	if paths == nil {
		return nil
	}
	rooted := make([]string, len(paths))
	for i, path := range paths {
		rooted[i] = filepath.Join(base, filepath.FromSlash(path))
	}
	return rooted
}
