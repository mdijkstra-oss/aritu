package sweep

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/kind"
)

const testFileBody = "package scenario\n\nimport \"testing\"\n\nfunc TestDoesAThing(t *testing.T) { _ = t }\n"

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestFilesFor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "alpha_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "alpha.go"), "package scenario\n")
	writeFile(t, filepath.Join(root, "sub", "beta_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "sub", "notes.md"), "notes\n")
	writeFile(t, filepath.Join(root, "rules", "solo", "fixtures", "fail-only", "bad_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "rules-of-thumb", "notes_test.go"), testFileBody)
	at := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }
	rulesDir := filepath.Join(root, "rules")

	tests := []struct {
		name     string
		patterns []string
		targeted []string
		want     []string
		wantErr  string
	}{
		{
			name:     "a literal path is taken as itself",
			patterns: []string{at("alpha_test.go")},
			targeted: []string{"tests"},
			want:     []string{at("alpha_test.go")},
		},
		{
			name:     "a double star crosses directories, and reaches a fixture somebody asked for",
			patterns: []string{at("**", "*_test.go")},
			targeted: []string{"tests"},
			want: []string{
				at("alpha_test.go"),
				at("rules-of-thumb", "notes_test.go"),
				at("rules", "solo", "fixtures", "fail-only", "bad_test.go"),
				at("sub", "beta_test.go"),
			},
		},
		{
			name:     "overlapping patterns contribute a file once",
			patterns: []string{at("**", "*_test.go"), at("sub", "*_test.go")},
			targeted: []string{"tests"},
			want: []string{
				at("alpha_test.go"),
				at("rules-of-thumb", "notes_test.go"),
				at("rules", "solo", "fixtures", "fail-only", "bad_test.go"),
				at("sub", "beta_test.go"),
			},
		},
		{
			name:     "what the rules target is swept when no pattern is given, the rules directory apart",
			targeted: []string{"tests"},
			want:     []string{at("alpha_test.go"), at("rules-of-thumb", "notes_test.go"), at("sub", "beta_test.go")},
		},
		{
			name:     "enabling a rule about another kind widens that sweep with no second list to edit",
			targeted: []string{"docs", "tests"},
			want: []string{
				at("alpha_test.go"),
				at("rules-of-thumb", "notes_test.go"),
				at("sub", "beta_test.go"),
				at("sub", "notes.md"),
			},
		},
		{
			name:     "a pattern on the command line outranks what the rules target",
			patterns: []string{at("alpha_test.go")},
			targeted: []string{"docs"},
			want:     []string{at("alpha_test.go")},
		},
		{
			name:     "but naming a fixture is honoured, because that was asked for",
			patterns: []string{at("rules", "solo", "fixtures", "fail-only", "bad_test.go")},
			targeted: []string{"tests"},
			want:     []string{at("rules", "solo", "fixtures", "fail-only", "bad_test.go")},
		},
		{
			name:     "a kind matching nothing here is an empty sweep rather than a green one",
			targeted: []string{"migrations"},
			wantErr:  "no targets",
		},
		{
			name:     "a pattern matching nothing names itself",
			patterns: []string{at("nowhere", "**", "*_test.go")},
			targeted: []string{"tests"},
			wantErr:  "nowhere",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kinds, err := kind.Resolve(root, map[string][]string{"migrations": {filepath.Join(root, "db/**/*.sql")}})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			got, err := filesFor(tc.patterns, derivedSweep{kinds: kinds, targeted: tc.targeted, rulesDir: rulesDir})

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("filesFor(%q, %q) = %q, want an error mentioning %q", tc.patterns, tc.targeted, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("filesFor(%q, %q) error = %v, want it to mention %q", tc.patterns, tc.targeted, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("filesFor(%q, %q) error = %v, want none", tc.patterns, tc.targeted, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("filesFor(%q, %q) = %q, want %q", tc.patterns, tc.targeted, got, tc.want)
			}
		})
	}
}

func TestCheckEveryFileIsTargeted(t *testing.T) {
	root := t.TempDir()
	kinds, err := kind.Resolve(root, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	isTargeted := targetingBy(kinds, root)
	aboutTests := rule.Rule{Name: "tests-one-thing", Targets: []string{"tests"}}
	aboutDocs := rule.Rule{Name: "prose-is-legible", Targets: []string{"docs"}}
	at := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }

	tests := []struct {
		name    string
		files   []string
		rules   []rule.Rule
		wantErr string
	}{
		{
			name:  "every file is of some enabled rule's kind",
			files: []string{at("alpha_test.go"), at("README.md")},
			rules: []rule.Rule{aboutTests, aboutDocs},
		},
		{
			name:    "a file only one rule was about, with that rule not enabled",
			files:   []string{at("alpha_test.go"), at("README.md")},
			rules:   []rule.Rule{aboutTests},
			wantErr: "README.md",
		},
		{
			name:    "an implementation file no rule about tests is handed",
			files:   []string{at("alpha.go")},
			rules:   []rule.Rule{aboutTests, aboutDocs},
			wantErr: "alpha.go",
		},
		{
			name:  "a path typed against the working directory is rooted before it is compared",
			files: []string{"sub/beta_test.go"},
			rules: []rule.Rule{aboutTests},
		},
		{
			name:  "nothing to sweep is nothing to refuse",
			files: nil,
			rules: []rule.Rule{aboutTests},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkEveryFileIsTargeted(tc.files, tc.rules, isTargeted)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkEveryFileIsTargeted(%q) = %v, want no error", tc.files, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkEveryFileIsTargeted(%q) = nil, want an error naming %q", tc.files, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("checkEveryFileIsTargeted(%q) error = %q, want it to name %q", tc.files, err, tc.wantErr)
			}
		})
	}
}

func TestTargetedKindsOf(t *testing.T) {
	tests := []struct {
		name  string
		rules []rule.Rule
		want  []string
	}{
		{
			name:  "one rule contributes what it is about",
			rules: []rule.Rule{{Targets: []string{"tests"}}},
			want:  []string{"tests"},
		},
		{
			name:  "rules about the same kind contribute it once",
			rules: []rule.Rule{{Targets: []string{"tests"}}, {Targets: []string{"tests"}}},
			want:  []string{"tests"},
		},
		{
			name:  "rules about different kinds each widen the sweep",
			rules: []rule.Rule{{Targets: []string{"tests"}}, {Targets: []string{"docs"}}},
			want:  []string{"docs", "tests"},
		},
		{
			name:  "a rule about several kinds contributes all of them",
			rules: []rule.Rule{{Targets: []string{"code", "docs"}}},
			want:  []string{"code", "docs"},
		},
		{
			name:  "no rules target nothing",
			rules: nil,
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := targetedKindsOf(tc.rules); !slices.Equal(got, tc.want) {
				t.Errorf("targetedKindsOf() = %v, want %v", got, tc.want)
			}
		})
	}
}
