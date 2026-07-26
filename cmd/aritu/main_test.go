package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/run"
)

func TestExecute(t *testing.T) {
	neutral := t.TempDir()
	emptyRules := t.TempDir()
	soloRules := writeRules(t, "solo")
	twoRules := writeRules(t, "first", "second")
	baselessRules := writeRulesWithoutBase(t)
	targets := writeTargets(t)
	alpha := filepath.Join(targets, "alpha_test.go")

	satisfiedClaude := writeStubClaude(t, true)
	dissatisfiedClaude := writeStubClaude(t, false)
	failingClaude := writeFailingClaude(t)

	votesRepo := writeRepo(t, "votes: 3\nrules:\n  dir: ./rules\n")
	typoRepo := writeRepo(t, "vote: 4\n")
	includeRepo := writeRepo(t, "rules:\n  dir: ./rules\ninclude:\n  - 'internal/**/*_test.go'\n")
	votesRepoPkg := filepath.Join(votesRepo, "internal", "pkg")

	tests := []struct {
		name        string
		dir         string
		args        []string
		want        lint.Exit
		wantStdout  []string
		wantStderr  []string
		wantReports int
		notWant     string
	}{
		{
			name:       "no command names the commands there are",
			want:       lint.ExitError,
			wantStderr: []string{"apply", "selftest", "Usage:"},
		},
		{
			name:       "a command that does not exist is refused",
			args:       []string{"lint", "one-reason-to-fail"},
			want:       lint.ExitError,
			wantStderr: []string{"lint", "Usage:"},
		},
		{
			name:       "apply help carries the flags and the exit codes",
			args:       []string{"apply", "--help"},
			want:       lint.ExitPass,
			wantStdout: []string{"Usage: aritu apply", "--votes", "--rule", "Exit codes:", "2  one or more targets"},
			notWant:    "aritu apply:",
		},
		{
			name:       "the root help carries both commands and every flag",
			args:       []string{"--help"},
			want:       lint.ExitPass,
			wantStdout: []string{"apply", "selftest", "--model", "--effort", "--jobs", "--timeout", "--config"},
		},
		{
			name:       "apply with a vote count that is not a number",
			args:       []string{"apply", "--votes", "many", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"--votes", "many"},
		},
		{
			name:       "apply with zero votes",
			args:       []string{"apply", "--votes", "0", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"votes must be at least 1, got 0"},
		},
		{
			name:       "selftest with negative votes",
			args:       []string{"selftest", "--votes=-2"},
			want:       lint.ExitError,
			wantStderr: []string{"votes must be at least 1, got -2"},
		},
		{
			name:       "apply with a flag nobody defined",
			args:       []string{"apply", "--rounds", "4", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"--rounds"},
		},
		{
			name:       "apply with a timeout that is not a duration",
			args:       []string{"apply", "--timeout", "soon", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"--timeout", "soon"},
		},
		{
			name:       "an unknown output is refused before any model is called",
			args:       []string{"apply", "--output", "yaml", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{`unknown output "yaml"`, "pretty or json"},
		},
		{
			name:       "an unknown effort is refused before any model is called",
			args:       []string{"apply", "--effort", "extreme", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{`unknown effort "extreme"`, "xhigh"},
		},
		{
			name:       "apply with neither a pattern nor an include list says what to pass",
			args:       []string{"apply", "--rules", soloRules},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"no targets", "aritu.yml"},
		},
		{
			name:       "a pattern matching nothing is an error naming the pattern",
			args:       []string{"apply", "--rules", soloRules, filepath.Join(targets, "nowhere", "**", "*_test.go")},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"nowhere", "matched no files"},
		},
		{
			name:       "a rule nobody wrote is an error naming it",
			args:       []string{"apply", "--rules", soloRules, "--rule", "no-such-rule", alpha},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"no-such-rule"},
		},
		{
			name:       "a rules directory with no shared base prompt",
			args:       []string{"apply", "--rules", baselessRules, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"base prompt", "base.md"},
		},
		{
			name:       "a rules directory holding no rules",
			args:       []string{"apply", "--rules", emptyRules, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"holds no rules"},
		},
		{
			name:       "every unit satisfying its rule",
			args:       []string{"apply", "--rules", soloRules, "--claude", satisfiedClaude, alpha},
			want:       lint.ExitPass,
			wantStdout: []string{"alpha_test.go", "solo", "✓ TestDoesAThing", "1 passed"},
		},
		{
			name:       "a unit falling short of its rule",
			args:       []string{"apply", "--rules", soloRules, "--claude", dissatisfiedClaude, alpha},
			want:       lint.ExitFail,
			wantStdout: []string{"✗ TestDoesAThing", "1 failed"},
		},
		{
			name:       "a target that could not be run is still reported",
			args:       []string{"apply", "--rules", soloRules, "--claude", failingClaude, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"alpha_test.go", "could not run", "exit status 1"},
		},
		{
			name:        "json emits one envelope covering every file and rule",
			args:        []string{"apply", "--output", "json", "--rules", twoRules, "--claude", satisfiedClaude, alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{`"reports"`, `"rule": "first"`, `"rule": "second"`, `"votes": 1`},
			wantReports: 2,
		},
		{
			name:        "a repeated rule flag judges each rule it names",
			args:        []string{"apply", "--output", "json", "--rules", twoRules, "--rule", "second", "--rule", "first", "--claude", satisfiedClaude, alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{`"rule": "first"`, `"rule": "second"`},
			wantReports: 2,
		},
		{
			name:        "overlapping patterns judge a file once",
			args:        []string{"apply", "--output", "json", "--rules", soloRules, "--claude", satisfiedClaude, filepath.Join(targets, "*_test.go"), alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{"alpha_test.go", "beta_test.go"},
			wantReports: 2,
		},
		{
			name:       "the config file supplies the vote count and the rules directory",
			dir:        votesRepoPkg,
			args:       []string{"apply", "--output", "json", "--claude", satisfiedClaude, "alpha_test.go"},
			want:       lint.ExitPass,
			wantStdout: []string{`"rule": "solo"`, `"votes": 3`},
		},
		{
			name:       "a flag overrides the config file",
			dir:        votesRepoPkg,
			args:       []string{"apply", "--output", "json", "--votes", "1", "--claude", satisfiedClaude, "alpha_test.go"},
			want:       lint.ExitPass,
			wantStdout: []string{`"votes": 1`},
		},
		{
			name:       "a misspelled config key fails the run naming the key",
			dir:        typoRepo,
			args:       []string{"apply", "--claude", satisfiedClaude, "internal/pkg/alpha_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"vote"},
		},
		{
			name:        "the include list supplies the targets when no pattern is given",
			dir:         includeRepo,
			args:        []string{"apply", "--output", "json", "--claude", satisfiedClaude},
			want:        lint.ExitPass,
			wantStdout:  []string{"alpha_test.go"},
			wantReports: 1,
		},
		{
			name:       "an explicit config is used and the upward search is skipped",
			args:       []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "--output", "json", "--claude", satisfiedClaude, filepath.Join(votesRepoPkg, "alpha_test.go")},
			want:       lint.ExitPass,
			wantStdout: []string{`"rule": "solo"`, `"votes": 3`},
		},
		{
			name:       "selftest still prints its table when the rule cannot be loaded",
			args:       []string{"selftest", "--rules", emptyRules, "--rule", "no-such-rule"},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE", "EXPECT", "RESULT", "VERDICTS", "0/0 fixtures hold"},
			wantStderr: []string{"aritu selftest:", "no-such-rule"},
		},
		{
			name:       "selftest reports a rules directory with no shared base prompt",
			args:       []string{"selftest", "--rules", baselessRules, "--rule", "solo"},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE"},
			wantStderr: []string{"base prompt", "base.md"},
		},
		{
			name:       "selftest still prints its table when the model cannot be reached",
			args:       []string{"selftest", "--rules", soloRules, "--claude", failingClaude},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE", "pass-only", "ERROR", "exit status 1", "0/1 fixtures hold"},
		},
		{
			name:       "selftest with no rule named exercises every rule",
			args:       []string{"selftest", "--rules", twoRules, "--claude", satisfiedClaude},
			want:       lint.ExitPass,
			wantStdout: []string{"rule: first", "rule: second", "1/1 fixtures hold"},
		},
		{
			name:       "selftest over a rules directory holding no rules",
			args:       []string{"selftest", "--rules", emptyRules},
			want:       lint.ExitError,
			wantStderr: []string{"holds no rules"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(orDefault(tc.dir, neutral))
			var stdout, stderr bytes.Buffer

			got := execute(tc.args, &stdout, &stderr)

			if got != tc.want {
				t.Errorf("execute(%q) = %d, want %d\nstdout: %s\nstderr: %s", tc.args, got, tc.want, stdout.String(), stderr.String())
			}
			if len(tc.wantStdout) == 0 && stdout.Len() != 0 {
				t.Errorf("execute(%q) wrote to stdout: %q", tc.args, stdout.String())
			}
			for _, want := range tc.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("execute(%q) stdout = %q, want it to contain %q", tc.args, stdout.String(), want)
				}
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("execute(%q) stderr = %q, want it to contain %q", tc.args, stderr.String(), want)
				}
			}
			if tc.notWant != "" && strings.Contains(stderr.String(), tc.notWant) {
				t.Errorf("execute(%q) stderr = %q, want it to omit %q", tc.args, stderr.String(), tc.notWant)
			}
			if tc.wantReports > 0 {
				if got := reportsIn(t, stdout.Bytes()); got != tc.wantReports {
					t.Errorf("execute(%q) emitted %d reports, want %d\n%s", tc.args, got, tc.wantReports, stdout.String())
				}
			}
		})
	}
}

// TestResolvedFlags pins the three-layer precedence directly, because a flag
// holding its default is indistinguishable from a flag nobody typed and only the
// resolved struct shows which layer actually won.
func TestResolvedFlags(t *testing.T) {
	neutral := t.TempDir()
	votesRepo := writeRepo(t, "votes: 3\nrules:\n  dir: ./rules\n")
	timeoutRepo := writeRepo(t, "timeout: 90s\nmodel: opus\neffort: high\n")
	typoRepo := writeRepo(t, "vote: 4\n")

	defaults := settings{
		Model:   "sonnet",
		Effort:  "medium",
		Output:  "pretty",
		Rules:   "./rules",
		Claude:  "claude",
		Votes:   1,
		Jobs:    5,
		Timeout: 10 * time.Minute,
	}
	withTargets := func(s settings, patterns ...string) settings {
		s.Patterns = patterns
		return s
	}

	tests := []struct {
		name    string
		dir     string
		args    []string
		want    settings
		wantErr string
	}{
		{
			name: "the built-in defaults stand when nothing else speaks",
			args: []string{"apply", "parser_test.go"},
			want: withTargets(defaults, "parser_test.go"),
		},
		{
			name: "several patterns are all collected",
			args: []string{"apply", "a_test.go", "internal/**/*_test.go"},
			want: withTargets(defaults, "a_test.go", "internal/**/*_test.go"),
		},
		{
			name: "flags after the patterns still apply",
			args: []string{"apply", "parser_test.go", "--model", "opus", "--votes", "2"},
			want: withTargets(settings{
				Model: "opus", Effort: "medium", Output: "pretty", Rules: "./rules",
				Claude: "claude", Votes: 2, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "every flag overridden",
			args: []string{"selftest", "--model", "haiku", "--votes", "7", "--effort", "low", "--output", "json", "--rules", "/etc/aritu/rules", "--claude", "/usr/local/bin/claude", "--timeout", "90s", "--jobs", "3"},
			want: settings{
				Model: "haiku", Effort: "low", Output: "json", Rules: "/etc/aritu/rules",
				Claude: "/usr/local/bin/claude", Votes: 7, Jobs: 3, Timeout: 90 * time.Second,
			},
		},
		{
			name: "a repeated rule flag collects each name in the order it was given",
			args: []string{"apply", "--rule", "one-reason-to-fail", "--rule", "named-for-behavior", "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: "./rules",
				Claude: "claude", Votes: 1, Jobs: 5, Timeout: 10 * time.Minute,
				Rule: []string{"one-reason-to-fail", "named-for-behavior"},
			}, "parser_test.go"),
		},
		{
			name: "the config file overrides the built-in defaults",
			args: []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Claude: "claude", Votes: 3, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "a flag overrides the config file",
			args: []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "--votes", "2", "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Claude: "claude", Votes: 2, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "a duration in the config arrives as a duration rather than the yaml text",
			args: []string{"apply", "--config", filepath.Join(timeoutRepo, "aritu.yml"), "parser_test.go"},
			want: withTargets(settings{
				Model: "opus", Effort: "high", Output: "pretty", Rules: "./rules",
				Claude: "claude", Votes: 1, Jobs: 5, Timeout: 90 * time.Second,
			}, "parser_test.go"),
		},
		{
			name: "the config above the working directory is found without being named",
			dir:  filepath.Join(votesRepo, "internal", "pkg"),
			args: []string{"apply", "alpha_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Claude: "claude", Votes: 3, Jobs: 5, Timeout: 10 * time.Minute,
			}, "alpha_test.go"),
		},
		{
			name:    "a misspelled config key fails the parse naming the key",
			dir:     typoRepo,
			args:    []string{"apply", "alpha_test.go"},
			wantErr: "vote",
		},
		{
			name:    "a config that is named but absent fails the parse",
			args:    []string{"apply", "--config", filepath.Join(neutral, "aritu.yml"), "parser_test.go"},
			wantErr: "aritu.yml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(orDefault(tc.dir, neutral))
			var cli CLI
			parser := newParser(&cli, io.Discard, io.Discard, func(int) {})

			_, err := parser.Parse(tc.args)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want an error mentioning %q", tc.args, settingsOf(cli), tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Parse(%q) error = %v, want it to mention %q", tc.args, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want none", tc.args, err)
			}
			if got := settingsOf(cli); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) resolved to %+v, want %+v", tc.args, got, tc.want)
			}
		})
	}
}

// settings is the resolved command line without the parser's own bookkeeping, so
// a precedence table can compare whole results rather than field by field.
type settings struct {
	Model    string
	Effort   string
	Output   string
	Rules    string
	Claude   string
	Votes    int
	Jobs     int
	Timeout  time.Duration
	Rule     []string
	Patterns []string
}

func settingsOf(cli CLI) settings {
	return settings{
		Model:    cli.Model,
		Effort:   cli.Effort,
		Output:   cli.Output,
		Rules:    cli.Rules,
		Claude:   cli.Claude,
		Votes:    cli.Votes,
		Jobs:     cli.Jobs,
		Timeout:  cli.Timeout,
		Rule:     cli.Rule,
		Patterns: cli.Apply.Patterns,
	}
}

func TestValidate(t *testing.T) {
	valid := CLI{Votes: 1, Output: "pretty", Effort: "medium"}
	with := func(change func(*CLI)) CLI {
		changed := valid
		change(&changed)
		return changed
	}

	tests := []struct {
		name    string
		cli     CLI
		wantErr string
	}{
		{
			name: "a single vote, pretty output and a known effort pass",
			cli:  valid,
		},
		{
			name: "json is a known output",
			cli:  with(func(c *CLI) { c.Output = "json" }),
		},
		{
			name: "an empty effort leaves the CLI default standing",
			cli:  with(func(c *CLI) { c.Effort = "" }),
		},
		{
			name: "every level the CLI accepts is a known effort",
			cli:  with(func(c *CLI) { c.Effort = "xhigh" }),
		},
		{
			name:    "zero votes",
			cli:     with(func(c *CLI) { c.Votes = 0 }),
			wantErr: "votes must be at least 1, got 0",
		},
		{
			name:    "negative votes",
			cli:     with(func(c *CLI) { c.Votes = -3 }),
			wantErr: "votes must be at least 1, got -3",
		},
		{
			name:    "an output with no reporter behind it",
			cli:     with(func(c *CLI) { c.Output = "yaml" }),
			wantErr: `unknown output "yaml", want pretty or json`,
		},
		{
			name:    "an effort the CLI does not accept",
			cli:     with(func(c *CLI) { c.Effort = "extreme" }),
			wantErr: `unknown effort "extreme"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.cli)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validate(%+v) = %v, want no error", tc.cli, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate(%+v) = nil, want an error mentioning %q", tc.cli, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("validate(%+v) error = %q, want it to contain %q", tc.cli, err, tc.wantErr)
			}
		})
	}
}

func TestTargetsFor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "alpha_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "sub", "beta_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "sub", "notes.md"), "not a target\n")
	at := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }

	tests := []struct {
		name     string
		patterns []string
		include  []string
		want     []string
		wantErr  string
	}{
		{
			name:     "a literal path is taken as itself",
			patterns: []string{at("alpha_test.go")},
			want:     []string{at("alpha_test.go")},
		},
		{
			name:     "a double star crosses directories",
			patterns: []string{at("**", "*_test.go")},
			want:     []string{at("alpha_test.go"), at("sub", "beta_test.go")},
		},
		{
			name:     "overlapping patterns contribute a file once",
			patterns: []string{at("**", "*_test.go"), at("sub", "*_test.go")},
			want:     []string{at("alpha_test.go"), at("sub", "beta_test.go")},
		},
		{
			name:    "the include list is used when no pattern is given",
			include: []string{at("sub", "*_test.go")},
			want:    []string{at("sub", "beta_test.go")},
		},
		{
			name:     "a pattern on the command line outranks the include list",
			patterns: []string{at("alpha_test.go")},
			include:  []string{at("sub", "*_test.go")},
			want:     []string{at("alpha_test.go")},
		},
		{
			name:    "neither a pattern nor an include list says what to pass",
			wantErr: "no targets",
		},
		{
			name:     "a pattern matching nothing names itself",
			patterns: []string{at("nowhere", "**", "*_test.go")},
			wantErr:  "nowhere",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := targetsFor(tc.patterns, tc.include)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("targetsFor(%q, %q) = %q, want an error mentioning %q", tc.patterns, tc.include, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("targetsFor(%q, %q) error = %v, want it to mention %q", tc.patterns, tc.include, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("targetsFor(%q, %q) error = %v, want none", tc.patterns, tc.include, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("targetsFor(%q, %q) = %q, want %q", tc.patterns, tc.include, got, tc.want)
			}
		})
	}
}

func TestWorse(t *testing.T) {
	tests := []struct {
		name string
		a    lint.Exit
		b    lint.Exit
		want lint.Exit
	}{
		{name: "two clean rules stay clean", a: lint.ExitPass, b: lint.ExitPass, want: lint.ExitPass},
		{name: "a rule that failed outranks one that passed", a: lint.ExitPass, b: lint.ExitFail, want: lint.ExitFail},
		{name: "a rule that could not run outranks one that failed", a: lint.ExitFail, b: lint.ExitError, want: lint.ExitError},
		{name: "the ranking does not depend on the order", a: lint.ExitError, b: lint.ExitFail, want: lint.ExitError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := worse(tc.a, tc.b); got != tc.want {
				t.Errorf("worse(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

const testFileBody = "package scenario\n\nimport \"testing\"\n\nfunc TestDoesAThing(t *testing.T) { _ = t }\n"

// writeRules builds a rules directory holding one rule per name, each with a
// fixture that must pass, so the reporting paths can be reached without depending
// on the repository's own rules.
func writeRules(t *testing.T, names ...string) string {
	t.Helper()
	return writeRulesIn(t, t.TempDir(), names...)
}

func writeRulesIn(t *testing.T, root string, names ...string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, "base.md"), "Judge the behaviour a test pins down, never its syntax.\n")
	for _, name := range names {
		writeFile(t, filepath.Join(root, name, "prompt.md"), "---\ninclude_source: false\ngranularity: function\n---\nA test must pin down one behaviour.\n")
		fixture := filepath.Join(root, name, "fixtures", "pass-only")
		writeFile(t, filepath.Join(fixture, "scenario.go"), "package scenario\n")
		writeFile(t, filepath.Join(fixture, "scenario_test.go"), testFileBody)
	}
	return root
}

// writeRulesWithoutBase builds a rules directory holding a valid rule but no
// base.md, the shape that must fail loudly rather than judge without the shared
// guidance every rule is written against.
func writeRulesWithoutBase(t *testing.T) string {
	t.Helper()
	root := writeRules(t, "solo")
	if err := os.Remove(filepath.Join(root, "base.md")); err != nil {
		t.Fatalf("removing base.md: %v", err)
	}
	return root
}

// writeTargets builds a directory of files to judge, kept apart from the rules so
// a glob over the targets cannot sweep up a fixture.
func writeTargets(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "alpha_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "beta_test.go"), testFileBody)
	return root
}

// writeRepo builds a repository-shaped tree: aritu.yml at the root, the rules
// beside it and a test file two directories down, so config discovery from a
// subdirectory can be driven for real rather than described.
func writeRepo(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	writeRulesIn(t, filepath.Join(root, "rules"), "solo")
	writeFile(t, filepath.Join(root, "internal", "pkg", "alpha_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "aritu.yml"), config)
	return root
}

// writeFailingClaude installs a stand-in claude binary that always fails, so the
// unreachable-model path can be driven for real rather than described.
func writeFailingClaude(t *testing.T) string {
	t.Helper()
	return writeScript(t, "#!/bin/sh\nexit 1\n")
}

// writeStubClaude installs a stand-in claude binary answering both calls a target
// costs: it recognises the enumeration by its schema and gives every unit the same
// verdict, so a whole run is free and its outcome is fixed.
func writeStubClaude(t *testing.T, satisfies bool) string {
	t.Helper()
	return writeScript(t, fmt.Sprintf(`#!/bin/sh
schema=""
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--json-schema" ]; then
		schema="$2"
	fi
	shift
done
cat >/dev/null
case "$schema" in
	*'"names"'*)
		printf '{"structured_output":{"names":["TestDoesAThing"]}}\n'
		;;
	*)
		printf '{"structured_output":{"TestDoesAThing":{"satisfies":%t,"reason":"names the input the case supplies"}}}\n'
		;;
esac
`, satisfies))
}

func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing stand-in claude: %v", err)
	}
	return path
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func reportsIn(t *testing.T, stdout []byte) int {
	t.Helper()
	var envelope run.Envelope
	if err := json.Unmarshal(stdout, &envelope); err != nil {
		t.Fatalf("decoding %q as a report envelope: %v", stdout, err)
	}
	return len(envelope.Reports)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
