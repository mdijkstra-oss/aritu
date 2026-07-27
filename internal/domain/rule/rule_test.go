package rule

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// knownTargets is the vocabulary a repository would have resolved before any rule
// is read: the built-in kinds, plus one it declared itself.
var knownTargets = []string{"code", "docs", "migrations", "tests"}

func TestParsePrompt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Prompt
		wantErr bool
	}{
		{
			name: "include_source true",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\n\nJudge the test.\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Description: "how to comply", Body: "Judge the test.\n"},
		},
		{
			name: "include_source false",
			raw:  "---\ntargets: [tests]\ninclude_source: false\ngranularity: test_case\ndescription: how to comply\n---\nJudge the test.",
			want: Prompt{IncludeSource: false, Granularity: GranularityTestCase, Targets: []string{"tests"}, Description: "how to comply", Body: "Judge the test."},
		},
		{
			name: "an include names a fragment the binary carries",
			raw:  "---\ntargets: [tests]\ninclude: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Include: []string{"tests"}, Description: "how to comply", Body: "body"},
		},
		{
			name:    "an include naming a fragment nobody wrote is refused",
			raw:     "---\ntargets: [tests]\ninclude: [haiku]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name:    "the listing half is not includable on its own",
			raw:     "---\ntargets: [tests]\ninclude: [tests.enumerate]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name: "a rule may target a kind that is not about tests at all",
			raw:  "---\ntargets: [docs]\ninclude_source: false\ngranularity: file\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: false, Granularity: GranularityFile, Targets: []string{"docs"}, Description: "how to comply", Body: "body"},
		},
		{
			name: "a rule may target several kinds",
			raw:  "---\ntargets: [code, docs]\ninclude_source: false\ngranularity: file\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: false, Granularity: GranularityFile, Targets: []string{"code", "docs"}, Description: "how to comply", Body: "body"},
		},
		{
			name: "a kind the repository declared is targetable like a built-in one",
			raw:  "---\ntargets: [migrations]\ninclude_source: false\ngranularity: file\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: false, Granularity: GranularityFile, Targets: []string{"migrations"}, Description: "how to comply", Body: "body"},
		},
		{
			name:    "missing targets key",
			raw:     "---\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name:    "targets naming a kind nobody defined",
			raw:     "---\ntargets: [prose]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name:    "the singular typo matches no kind and is refused rather than run over nothing",
			raw:     "---\ntargets: [test]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name:    "an empty targets list is a rule that would never run",
			raw:     "---\ntargets: []\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name:    "one unknown kind among known ones is still refused",
			raw:     "---\ntargets: [tests, prose]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name: "a folded description arrives as the one paragraph it renders to",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: >-\n  Give every test\n  one reason to fail.\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Description: "Give every test one reason to fail.", Body: "body"},
		},
		{
			name:    "a description key holding nothing instructs nobody and is refused",
			raw:     "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: \"   \"\n---\nbody",
			wantErr: true,
		},
		{
			name:    "missing description key",
			raw:     "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\n---\nbody",
			wantErr: true,
		},
		{
			name: "body keeps its own delimiters and blank lines",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\n\n\nfirst\n\n---\nlast\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Description: "how to comply", Body: "first\n\n---\nlast\n"},
		},
		{
			name: "empty body",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Description: "how to comply", Body: ""},
		},
		{
			name: "other frontmatter keys are ignored",
			raw:  "---\ntitle: one reason to fail\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Targets: []string{"tests"}, Description: "how to comply", Body: "body"},
		},
		{
			name:    "missing include_source key",
			raw:     "---\ntitle: one reason to fail\n---\nbody",
			wantErr: true,
		},
		{
			name:    "missing granularity key",
			raw:     "---\ninclude_source: true\n---\nbody",
			wantErr: true,
		},
		{
			name:    "granularity outside the allowed set",
			raw:     "---\ntargets: [tests]\ninclude_source: true\ngranularity: package\ndescription: how to comply\n---\nbody",
			wantErr: true,
		},
		{
			name: "granularity file",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: file\ndescription: how to comply\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFile, Targets: []string{"tests"}, Description: "how to comply", Body: "body"},
		},
		{
			name:    "empty frontmatter",
			raw:     "---\n---\nbody",
			wantErr: true,
		},
		{
			name:    "no frontmatter",
			raw:     "include_source: true\nbody",
			wantErr: true,
		},
		{
			name:    "unterminated frontmatter",
			raw:     "---\ninclude_source: true\nbody",
			wantErr: true,
		},
		{
			name:    "malformed yaml",
			raw:     "---\ninclude_source: [true\n---\nbody",
			wantErr: true,
		},
		{
			name:    "include_source is not a bool",
			raw:     "---\ninclude_source: yes please\n---\nbody",
			wantErr: true,
		},
		{
			name:    "empty input",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePrompt(tt.raw, knownTargets)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePrompt() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestParsePromptNamesTheKindsItKnows pins the diagnostic rather than the refusal.
// The typo case is the reason targets cannot be defaulted, and a reader who has
// just misspelled one needs the list to compare against.
func TestParsePromptNamesTheKindsItKnows(t *testing.T) {
	_, err := ParsePrompt("---\ntargets: [test]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody", knownTargets)

	if err == nil {
		t.Fatal("ParsePrompt() accepted a kind nobody defined")
	}
	for _, known := range knownTargets {
		if !strings.Contains(err.Error(), known) {
			t.Errorf("ParsePrompt() error = %q, want it to name %q", err, known)
		}
	}
}

// TestParsePromptNamesTheKeyItIsMissing pins the diagnostic rather than the
// refusal. Four keys are required, and a rule author who left one out learns
// nothing from being told the file is wrong: the message has to name which key,
// or the fix is a search through four candidates.
func TestParsePromptNamesTheKeyItIsMissing(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "include_source",
			raw:  "---\ntargets: [tests]\ngranularity: function\ndescription: how to comply\n---\nbody",
			want: "include_source",
		},
		{
			name: "granularity",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ndescription: how to comply\n---\nbody",
			want: "granularity",
		},
		{
			name: "description",
			raw:  "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\n---\nbody",
			want: "description",
		},
		{
			name: "targets",
			raw:  "---\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody",
			want: "targets",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePrompt(tc.raw, knownTargets)

			if err == nil {
				t.Fatalf("ParsePrompt() accepted a prompt setting no %s", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("ParsePrompt() error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

func TestParseExpectation(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		want    Expectation
		wantErr bool
	}{
		{name: "pass prefix", dirName: "pass-single-assert", want: ExpectPass},
		{name: "fail prefix", dirName: "fail-two-behaviors", want: ExpectFail},
		{name: "prefix without scenario", dirName: "fail-", want: ExpectFail},
		{name: "unprefixed", dirName: "single-assert", wantErr: true},
		{name: "prefix without separator", dirName: "passing", wantErr: true},
		{name: "prefix not at start", dirName: "should-pass-here", wantErr: true},
		{name: "empty", dirName: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExpectation(tt.dirName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseExpectation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseExpectation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindSource(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string
		testPath        string
		want            string
		wantOK          bool
		wantErrContains []string
	}{
		{
			name:     "the implementation beside the test",
			files:    map[string]string{"internal/parser_test.go": "", "internal/parser.go": ""},
			testPath: "internal/parser_test.go",
			want:     "internal/parser.go",
			wantOK:   true,
		},
		{
			name: "a mirrored source tree",
			files: map[string]string{
				"src/test/java/com/x/ParserTest.java": "",
				"src/main/java/com/x/Parser.java":     "",
			},
			testPath: "src/test/java/com/x/ParserTest.java",
			want:     "src/main/java/com/x/Parser.java",
			wantOK:   true,
		},
		{
			name: "a sibling outranks the mirrored tree",
			files: map[string]string{
				"src/test/java/com/x/ParserTest.java": "",
				"src/test/java/com/x/Parser.java":     "",
				"src/main/java/com/x/Parser.java":     "",
			},
			testPath: "src/test/java/com/x/ParserTest.java",
			want:     "src/test/java/com/x/Parser.java",
			wantOK:   true,
		},
		{
			name:     "a test tree swapped for a source tree",
			files:    map[string]string{"test/parser.test.ts": "", "src/parser.ts": ""},
			testPath: "test/parser.test.ts",
			want:     "src/parser.ts",
			wantOK:   true,
		},
		{
			name:     "a module under the package root",
			files:    map[string]string{"tests/test_parser.py": "", "parser.py": ""},
			testPath: "tests/test_parser.py",
			want:     "parser.py",
			wantOK:   true,
		},
		{
			name:            "nothing the convention offers exists",
			files:           map[string]string{"internal/parser_test.go": ""},
			testPath:        "internal/parser_test.go",
			wantErrContains: []string{"looked for", filepath.FromSlash("internal/parser.go")},
		},
		{
			name:            "a directory standing where the implementation would be",
			files:           map[string]string{"internal/parser_test.go": "", "internal/parser.go/keep": ""},
			testPath:        "internal/parser_test.go",
			wantErrContains: []string{"looked for"},
		},
		{
			name:     "every candidate a search offers is named",
			files:    map[string]string{"tests/test_parser.py": ""},
			testPath: "tests/test_parser.py",
			wantErrContains: []string{
				filepath.FromSlash("tests/parser.py"),
				filepath.FromSlash("src/parser.py"),
			},
		},
		{
			name:            "not a test file",
			files:           map[string]string{"internal/parser.go": ""},
			testPath:        "internal/parser.go",
			wantErrContains: []string{"matches no test file naming convention"},
		},
		{
			name:            "an extension no convention covers",
			files:           map[string]string{"parser_test.rb": "", "parser.rb": ""},
			testPath:        "parser_test.rb",
			wantErrContains: []string{"matches no test file naming convention"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)

			got, err := FindSource(filepath.Join(root, filepath.FromSlash(tt.testPath)))

			if (err == nil) != tt.wantOK {
				t.Fatalf("FindSource() error = %v, want a source path: %v", err, tt.wantOK)
			}
			if err != nil {
				for _, want := range tt.wantErrContains {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("FindSource() error = %q, want it to name %q", err, want)
					}
				}
				return
			}
			want := filepath.Join(root, filepath.FromSlash(tt.want))
			if got != want {
				t.Errorf("FindSource() = %q, want %q", got, want)
			}
		})
	}
}

func TestExpectationStringNamesTheExpectation(t *testing.T) {
	tests := []struct {
		name   string
		expect Expectation
		want   string
	}{
		{name: "pass", expect: ExpectPass, want: "pass"},
		{name: "fail", expect: ExpectFail, want: "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.expect.String(); got != tt.want {
				t.Errorf("Expectation.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpectationStringPanicsOnAnUnknownValue(t *testing.T) {
	tests := []struct {
		name   string
		expect Expectation
	}{
		{name: "zero value", expect: Expectation(0)},
		{name: "out of range", expect: Expectation(9)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Expectation.String() did not panic")
				}
			}()
			_ = tt.expect.String()
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		rule    string
		want    Rule
		wantErr bool
	}{
		{
			name:  "include_source true",
			files: map[string]string{"one-reason-to-fail/prompt.md": "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: pin down one behavior\n---\n\nOne behavior.\n"},
			rule:  "one-reason-to-fail",
			want:  Rule{Name: "one-reason-to-fail", Prompt: "One behavior.\n", Description: "pin down one behavior", Targets: []string{"tests"}, IncludeSource: true, Granularity: GranularityFunction},
		},
		{
			name:  "include_source false",
			files: map[string]string{"named-for-behavior/prompt.md": "---\ntargets: [tests]\ninclude_source: false\ngranularity: test_case\ndescription: name it for an outcome\n---\nName it.\n"},
			rule:  "named-for-behavior",
			want:  Rule{Name: "named-for-behavior", Prompt: "Name it.\n", Description: "name it for an outcome", Targets: []string{"tests"}, IncludeSource: false, Granularity: GranularityTestCase},
		},
		{
			name:  "a rule about a kind that is not tests loads the same way",
			files: map[string]string{"prose-is-legible/prompt.md": "---\ntargets: [docs]\ninclude_source: false\ngranularity: file\ndescription: write prose a stranger can read\n---\nRead it.\n"},
			rule:  "prose-is-legible",
			want:  Rule{Name: "prose-is-legible", Prompt: "Read it.\n", Description: "write prose a stranger can read", Targets: []string{"docs"}, IncludeSource: false, Granularity: GranularityFile},
		},
		{
			name:    "a rule targeting a kind this repository never defined",
			files:   map[string]string{"prose-is-legible/prompt.md": "---\ntargets: [prose]\ninclude_source: false\ngranularity: file\ndescription: how to comply\n---\nbody"},
			rule:    "prose-is-legible",
			wantErr: true,
		},
		{
			name:    "missing rule directory",
			files:   map[string]string{"other/prompt.md": "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody"},
			rule:    "absent",
			wantErr: true,
		},
		{
			name:    "missing prompt file",
			files:   map[string]string{"no-prompt/fixtures/pass-a/a_test.go": "package a"},
			rule:    "no-prompt",
			wantErr: true,
		},
		{
			name:    "unparseable prompt",
			files:   map[string]string{"broken/prompt.md": "no frontmatter here"},
			rule:    "broken",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			got, err := Load(root, tt.rule, knownTargets)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			want := tt.want
			want.Dir = filepath.Join(root, tt.rule)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Load() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestLoadFixtures(t *testing.T) {
	const prompt = "---\ntargets: [tests]\ninclude_source: true\ngranularity: function\ndescription: how to comply\n---\nbody\n"

	type wantFixture struct {
		name     string
		testFile string
		expect   Expectation
	}

	tests := []struct {
		name    string
		files   map[string]string
		want    []wantFixture
		wantErr bool
	}{
		{
			name: "sorted by name",
			files: map[string]string{
				"r/prompt.md": prompt,
				"r/fixtures/pass-single/scenario_test.go": "package scenario",
				"r/fixtures/fail-two/scenario_test.go":    "package scenario",
				"r/fixtures/pass-double/other_test.go":    "package scenario",
			},
			want: []wantFixture{
				{name: "fail-two", testFile: "scenario_test.go", expect: ExpectFail},
				{name: "pass-double", testFile: "other_test.go", expect: ExpectPass},
				{name: "pass-single", testFile: "scenario_test.go", expect: ExpectPass},
			},
		},
		{
			name: "non test go file alongside is allowed",
			files: map[string]string{
				"r/prompt.md":                            prompt,
				"r/fixtures/pass-with-source/thing.go":   "package scenario",
				"r/fixtures/pass-with-source/README.md":  "notes",
				"r/fixtures/pass-with-source/a_test.go":  "package scenario",
				"r/fixtures/pass-with-source/sub/b.go":   "package sub",
				"r/fixtures/pass-with-source/sub/README": "notes",
			},
			want: []wantFixture{
				{name: "pass-with-source", testFile: "a_test.go", expect: ExpectPass},
			},
		},
		{
			name: "loose file in fixtures dir is skipped",
			files: map[string]string{
				"r/prompt.md":                 prompt,
				"r/fixtures/README.md":        "notes",
				"r/fixtures/pass-a/a_test.go": "package scenario",
			},
			want: []wantFixture{
				{name: "pass-a", testFile: "a_test.go", expect: ExpectPass},
			},
		},
		{
			name: "no test file in fixture",
			files: map[string]string{
				"r/prompt.md":            prompt,
				"r/fixtures/pass-a/a.go": "package scenario",
			},
			wantErr: true,
		},
		{
			name: "two test files in fixture",
			files: map[string]string{
				"r/prompt.md":                 prompt,
				"r/fixtures/pass-a/a_test.go": "package scenario",
				"r/fixtures/pass-a/b_test.go": "package scenario",
			},
			wantErr: true,
		},
		{
			name: "unprefixed fixture directory",
			files: map[string]string{
				"r/prompt.md":                 prompt,
				"r/fixtures/single/a_test.go": "package scenario",
			},
			wantErr: true,
		},
		{
			name: "no fixture directories",
			files: map[string]string{
				"r/prompt.md":          prompt,
				"r/fixtures/README.md": "notes",
			},
			wantErr: true,
		},
		{
			name:    "no fixtures dir",
			files:   map[string]string{"r/prompt.md": prompt},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			loaded, err := Load(root, "r", knownTargets)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			got, err := LoadFixtures(loaded)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadFixtures() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("LoadFixtures() = %#v, want %d fixtures", got, len(tt.want))
			}
			for i, want := range tt.want {
				dir := filepath.Join(loaded.Dir, "fixtures", want.name)
				expected := Fixture{
					Name:     want.name,
					TestFile: filepath.Join(dir, want.testFile),
					Expect:   want.expect,
				}
				if got[i] != expected {
					t.Errorf("fixture %d = %#v, want %#v", i, got[i], expected)
				}
			}
		})
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []string
		wantErr string
	}{
		{
			name:  "every rule directory, sorted",
			files: map[string]string{"second/prompt.md": "body", "first/prompt.md": "body"},
			want:  []string{"first", "second"},
		},
		{
			name:  "a parked rule is on disk and out of the listing",
			files: map[string]string{"enforced/prompt.md": "body", "_parked/prompt.md": "body"},
			want:  []string{"enforced"},
		},
		{
			name:  "a loose file beside the rules is not one",
			files: map[string]string{"enforced/prompt.md": "body", "README.md": "notes"},
			want:  []string{"enforced"},
		},
		{
			name:    "a directory holding nothing but parked rules is as empty as an empty one",
			files:   map[string]string{"_parked/prompt.md": "body"},
			wantErr: "holds no rules",
		},
		{
			name:    "a directory holding no rules at all",
			files:   map[string]string{"README.md": "notes"},
			wantErr: "holds no rules",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)

			got, err := List(root)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("List() = %v, want an error mentioning %q", got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("List() error = %v, want it to mention %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("List() error = %v, want none", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("List() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestLoadReachesAParkedRuleThatIsNamed pins the other half of parking. A parked
// rule keeps its prompt and its fixtures, so naming it runs it: what was asked for
// outranks what was derived, the same way a pattern naming a fixture judges it.
func TestLoadReachesAParkedRuleThatIsNamed(t *testing.T) {
	root := writeTree(t, map[string]string{
		"_parked/prompt.md": "---\ntargets: [tests]\ninclude_source: false\ngranularity: file\ndescription: how to comply\n---\nbody",
	})

	loaded, err := Load(root, "_parked", knownTargets)

	if err != nil {
		t.Fatalf("Load() error = %v, want a parked rule to load when it is named", err)
	}
	if loaded.Name != "_parked" {
		t.Errorf("Load() name = %q, want %q", loaded.Name, "_parked")
	}
}

func TestParseGranularity(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Granularity
		wantErr bool
	}{
		{name: "file", in: "file", want: GranularityFile},
		{name: "function", in: "function", want: GranularityFunction},
		{name: "test_case", in: "test_case", want: GranularityTestCase},
		{name: "test is the name it used to answer to", in: "test", wantErr: true},
		{name: "package is not a level yet", in: "package", wantErr: true},
		{name: "capitalised", in: "Test_Case", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGranularity(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGranularity(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseGranularity(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestGranularityStringNamesTheLevel(t *testing.T) {
	tests := []struct {
		name        string
		granularity Granularity
		want        string
	}{
		{name: "file", granularity: GranularityFile, want: "file"},
		{name: "function", granularity: GranularityFunction, want: "function"},
		{name: "test_case", granularity: GranularityTestCase, want: "test_case"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.granularity.String(); got != tt.want {
				t.Errorf("Granularity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGranularityStringPanicsOnAnUnknownValue(t *testing.T) {
	tests := []struct {
		name        string
		granularity Granularity
	}{
		{name: "zero value", granularity: Granularity(0)},
		{name: "out of range", granularity: Granularity(9)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Granularity.String() did not panic")
				}
			}()
			_ = tt.granularity.String()
		})
	}
}
