package rule

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePrompt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Prompt
		wantErr bool
	}{
		{
			name: "include_source true",
			raw:  "---\ninclude_source: true\ngranularity: function\n---\n\nJudge the test.\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Body: "Judge the test.\n"},
		},
		{
			name: "include_source false",
			raw:  "---\ninclude_source: false\ngranularity: test\n---\nJudge the test.",
			want: Prompt{IncludeSource: false, Granularity: GranularityTest, Body: "Judge the test."},
		},
		{
			name: "body keeps its own delimiters and blank lines",
			raw:  "---\ninclude_source: true\ngranularity: function\n---\n\n\nfirst\n\n---\nlast\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Body: "first\n\n---\nlast\n"},
		},
		{
			name: "empty body",
			raw:  "---\ninclude_source: true\ngranularity: function\n---\n",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Body: ""},
		},
		{
			name: "other frontmatter keys are ignored",
			raw:  "---\ntitle: one reason to fail\ninclude_source: true\ngranularity: function\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFunction, Body: "body"},
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
			raw:     "---\ninclude_source: true\ngranularity: package\n---\nbody",
			wantErr: true,
		},
		{
			name: "granularity file",
			raw:  "---\ninclude_source: true\ngranularity: file\n---\nbody",
			want: Prompt{IncludeSource: true, Granularity: GranularityFile, Body: "body"},
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
			got, err := ParsePrompt(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("ParsePrompt() = %#v, want %#v", got, tt.want)
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

func TestSourcePathFor(t *testing.T) {
	tests := []struct {
		name     string
		testPath string
		want     string
		wantOK   bool
	}{
		{name: "flat", testPath: "parser_test.go", want: "parser.go", wantOK: true},
		{name: "nested directories", testPath: "internal/domain/parser_test.go", want: "internal/domain/parser.go", wantOK: true},
		{name: "directory named like a test", testPath: "a_test.go/parser_test.go", want: "a_test.go/parser.go", wantOK: true},
		{name: "non test file", testPath: "parser.go", wantOK: false},
		{name: "testdata suffix without underscore", testPath: "parsertest.go", wantOK: false},
		{name: "bare test file", testPath: "_test.go", wantOK: false},
		{name: "bare test file in a directory", testPath: "internal/_test.go", wantOK: false},
		{name: "empty", testPath: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SourcePathFor(tt.testPath)
			if ok != tt.wantOK {
				t.Fatalf("SourcePathFor() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != filepath.FromSlash(tt.want) {
				t.Errorf("SourcePathFor() = %q, want %q", got, tt.want)
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
			files: map[string]string{"one-reason-to-fail/prompt.md": "---\ninclude_source: true\ngranularity: function\n---\n\nOne behavior.\n"},
			rule:  "one-reason-to-fail",
			want:  Rule{Name: "one-reason-to-fail", Prompt: "One behavior.\n", IncludeSource: true, Granularity: GranularityFunction},
		},
		{
			name:  "include_source false",
			files: map[string]string{"named-for-behavior/prompt.md": "---\ninclude_source: false\ngranularity: test\n---\nName it.\n"},
			rule:  "named-for-behavior",
			want:  Rule{Name: "named-for-behavior", Prompt: "Name it.\n", IncludeSource: false, Granularity: GranularityTest},
		},
		{
			name:    "missing rule directory",
			files:   map[string]string{"other/prompt.md": "---\ninclude_source: true\ngranularity: function\n---\nbody"},
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
			got, err := Load(root, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			want := tt.want
			want.Dir = filepath.Join(root, tt.rule)
			if got != want {
				t.Errorf("Load() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestLoadFixtures(t *testing.T) {
	const prompt = "---\ninclude_source: true\ngranularity: function\n---\nbody\n"

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
			loaded, err := Load(root, "r")
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

func TestParseGranularity(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Granularity
		wantErr bool
	}{
		{name: "file", in: "file", want: GranularityFile},
		{name: "function", in: "function", want: GranularityFunction},
		{name: "test", in: "test", want: GranularityTest},
		{name: "package is not a level yet", in: "package", wantErr: true},
		{name: "capitalised", in: "Test", wantErr: true},
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
		{name: "test", granularity: GranularityTest, want: "test"},
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

func TestLoadBase(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "trims surrounding whitespace",
			files: map[string]string{"base.md": "\n\nJudge behaviour, not syntax.\n\n"},
			want:  "Judge behaviour, not syntax.",
		},
		{
			name:    "missing base is an error rather than an empty prompt",
			files:   map[string]string{"other.md": "unrelated"},
			wantErr: true,
		},
		{
			name:    "blank base is an error",
			files:   map[string]string{"base.md": "  \n\t\n"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadBase(writeTree(t, tt.files))
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadBase() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("LoadBase() = %q, want %q", got, tt.want)
			}
		})
	}
}
