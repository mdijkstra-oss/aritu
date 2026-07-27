package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/lib/kind"
	"github.com/matthijn/aritu/prompts"
)

func TestExecute(t *testing.T) {
	// t.Setenv registers the restore; the Unsetenv that follows is what makes the
	// variable absent rather than empty, which is the case the run has to catch.
	t.Setenv(unsetVariable, "")
	if err := os.Unsetenv(unsetVariable); err != nil {
		t.Fatalf("unsetting %s: %v", unsetVariable, err)
	}

	satisfied := serveStubModel(t, true)
	dissatisfied := serveStubModel(t, false)
	unreachable := serveRejectingModel(t)

	// neutral holds a config and nothing else: a sweep started here finds no file
	// of any kind, which is what the empty-sweep cases need.
	neutral := t.TempDir()
	writeFile(t, filepath.Join(neutral, config.FileName), serviceBlock(satisfied))
	dissatisfiedRepo := writeRepo(t, serviceBlock(dissatisfied))
	unreachableRepo := writeRepo(t, serviceBlock(unreachable))
	emptyRules := t.TempDir()
	soloRules := writeRules(t, "solo")
	twoRules := writeRules(t, "first", "second")
	docsAndTestRules := writeRuleAbout(t, writeRules(t, "solo"), "prose-is-legible", "docs")
	mistypedRules := writeRuleAbout(t, t.TempDir(), "mistyped", "test")
	targets := writeTargets(t)
	alpha := filepath.Join(targets, "alpha_test.go")

	noServiceRepo := writeRepo(t, "rules:\n  dir: ./rules\n")
	unsetTokenRepo := writeRepo(t, serviceBlock(satisfied)+"  auth_token_var: "+unsetVariable+"\n")

	votesRepo := writeRepo(t, "votes: 3\nrules:\n  dir: ./rules\n"+serviceBlock(satisfied))
	typoRepo := writeRepo(t, "vote: 4\n")
	includeRepo := writeRepo(t, "rules:\n  dir: ./rules\ninclude:\n  - 'internal/**/*_test.go'\n")
	sweepRepo := writeRepo(t, "rules:\n  dir: ./rules\n"+serviceBlock(satisfied))
	targetsRepo := writeRepo(t, "rules:\n  dir: ./rules\ntargets:\n  tests:\n    - 'internal/pkg/*_test.go'\n"+serviceBlock(satisfied))
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
		notWantOut  string
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
			name:       "a sweep the enabled rules target nothing in says what to pass",
			args:       []string{"apply", "--rules", soloRules},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"no targets", "tests"},
		},
		{
			name:       "a file no enabled rule is about is refused rather than skipped",
			args:       []string{"apply", "--rules", soloRules, filepath.Join(targets, "notes.md")},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"no enabled rule targets", "notes.md"},
		},
		{
			name:        "naming a file of another kind alongside one it is about judges only the second",
			args:        []string{"apply", "--output", "json", "--rules", docsAndTestRules, filepath.Join(targets, "notes.md"), alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{`"rule": "prose-is-legible"`, `"rule": "solo"`, "notes.md", "alpha_test.go"},
			wantReports: 2,
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
			name:       "a rules directory holding no rules",
			args:       []string{"apply", "--rules", emptyRules, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"holds no rules"},
		},
		{
			name:       "every unit satisfying its rule",
			args:       []string{"apply", "--rules", soloRules, alpha},
			want:       lint.ExitPass,
			wantStdout: []string{"alpha_test.go", "solo", "✓ TestDoesAThing", "1 passed"},
		},
		{
			name:       "a sweep says what it covers before its first model call",
			args:       []string{"apply", "--rules", twoRules, alpha},
			want:       lint.ExitPass,
			wantStdout: []string{"alpha_test.go", "first", "second"},
			wantStderr: []string{"judging 1 file against 2 rules, 1 vote"},
		},
		{
			name:       "a unit falling short of its rule",
			dir:        dissatisfiedRepo,
			args:       []string{"apply", "--rules", soloRules, alpha},
			want:       lint.ExitFail,
			wantStdout: []string{"✗ TestDoesAThing", "1 failed"},
		},
		{
			name:       "a target that could not be run is still reported",
			dir:        unreachableRepo,
			args:       []string{"apply", "--rules", soloRules, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"alpha_test.go", "could not run", "401"},
		},
		{
			name:        "json emits one envelope covering every file and rule",
			args:        []string{"apply", "--output", "json", "--rules", twoRules, alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{`"reports"`, `"rule": "first"`, `"rule": "second"`, `"votes": 1`},
			wantReports: 2,
		},
		{
			name:        "a repeated rule flag judges each rule it names",
			args:        []string{"apply", "--output", "json", "--rules", twoRules, "--rule", "second", "--rule", "first", alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{`"rule": "first"`, `"rule": "second"`},
			wantReports: 2,
		},
		{
			name:        "overlapping patterns judge a file once",
			args:        []string{"apply", "--output", "json", "--rules", soloRules, filepath.Join(targets, "*_test.go"), alpha},
			want:        lint.ExitPass,
			wantStdout:  []string{"alpha_test.go", "beta_test.go"},
			wantReports: 2,
		},
		{
			name:       "the config file supplies the vote count and the rules directory",
			dir:        votesRepoPkg,
			args:       []string{"apply", "--output", "json", "alpha_test.go"},
			want:       lint.ExitPass,
			wantStdout: []string{`"rule": "solo"`, `"votes": 3`},
		},
		{
			name:       "a flag overrides the config file",
			dir:        votesRepoPkg,
			args:       []string{"apply", "--output", "json", "--votes", "1", "alpha_test.go"},
			want:       lint.ExitPass,
			wantStdout: []string{`"votes": 1`},
		},
		{
			name:       "a misspelled config key fails the run naming the key",
			dir:        typoRepo,
			args:       []string{"apply", "internal/pkg/alpha_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"vote"},
		},
		{
			name:       "the include list this repository used to carry is now a key nobody defined",
			dir:        includeRepo,
			args:       []string{"apply"},
			want:       lint.ExitError,
			wantStderr: []string{"include"},
		},
		{
			name:        "what the enabled rules target supplies the sweep, and a rule's own fixtures are not it",
			dir:         sweepRepo,
			args:        []string{"apply", "--output", "json"},
			want:        lint.ExitPass,
			wantStdout:  []string{"alpha_test.go"},
			wantReports: 1,
			notWantOut:  "scenario_test.go",
		},
		{
			name:        "a targets block replaces the built-in answer to which files are tests",
			dir:         targetsRepo,
			args:        []string{"apply", "--output", "json"},
			want:        lint.ExitPass,
			wantStdout:  []string{"alpha_test.go"},
			wantReports: 1,
			notWantOut:  "scenario_test.go",
		},
		{
			name:       "a rule targeting a kind nobody defined fails before a single model call",
			args:       []string{"apply", "--rules", mistypedRules, alpha},
			want:       lint.ExitError,
			wantStdout: []string{"0 passed"},
			wantStderr: []string{"mistyped", "tests"},
		},
		{
			name:       "an explicit config is used and the upward search is skipped",
			args:       []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "--output", "json", filepath.Join(votesRepoPkg, "alpha_test.go")},
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
			name:       "selftest still prints its table when the model cannot be reached",
			dir:        unreachableRepo,
			args:       []string{"selftest", "--rules", soloRules},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE", "pass-only", "ERROR", "401", "0/1 fixtures hold"},
		},
		{
			name:       "selftest with no rule named exercises every rule",
			args:       []string{"selftest", "--rules", twoRules},
			want:       lint.ExitPass,
			wantStdout: []string{"rule: first", "rule: second", "1/1 fixtures hold"},
		},
		{
			name:       "a repository with no endpoint configured says where to set one",
			dir:        noServiceRepo,
			args:       []string{"apply", "--rules", soloRules, alpha},
			want:       lint.ExitError,
			wantStderr: []string{"no service.endpoint configured", "aritu.yml"},
		},
		{
			name:       "an auth_token_var naming a variable nobody set stops the run before its first request",
			dir:        unsetTokenRepo,
			args:       []string{"apply", "--rules", soloRules, alpha},
			want:       lint.ExitError,
			wantStderr: []string{"service.auth_token_var names $" + unsetVariable, "not set"},
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
			t.Chdir(cmp.Or(tc.dir, neutral))
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
			if tc.notWantOut != "" && strings.Contains(stdout.String(), tc.notWantOut) {
				t.Errorf("execute(%q) stdout = %q, want it to omit %q", tc.args, stdout.String(), tc.notWantOut)
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
				Votes: 2, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "every flag overridden",
			args: []string{"selftest", "--model", "haiku", "--votes", "7", "--effort", "low", "--output", "json", "--rules", "/etc/aritu/rules", "--timeout", "90s", "--jobs", "3"},
			want: settings{
				Model: "haiku", Effort: "low", Output: "json", Rules: "/etc/aritu/rules",
				Votes: 7, Jobs: 3, Timeout: 90 * time.Second,
			},
		},
		{
			name: "a repeated rule flag collects each name in the order it was given",
			args: []string{"apply", "--rule", "one-reason-to-fail", "--rule", "named-for-behavior", "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: "./rules",
				Votes: 1, Jobs: 5, Timeout: 10 * time.Minute,
				Rule: []string{"one-reason-to-fail", "named-for-behavior"},
			}, "parser_test.go"),
		},
		{
			name: "the config file overrides the built-in defaults",
			args: []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Votes: 3, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "a flag overrides the config file",
			args: []string{"apply", "--config", filepath.Join(votesRepo, "aritu.yml"), "--votes", "2", "parser_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Votes: 2, Jobs: 5, Timeout: 10 * time.Minute,
			}, "parser_test.go"),
		},
		{
			name: "a duration in the config arrives as a duration rather than the yaml text",
			args: []string{"apply", "--config", filepath.Join(timeoutRepo, "aritu.yml"), "parser_test.go"},
			want: withTargets(settings{
				Model: "opus", Effort: "high", Output: "pretty", Rules: "./rules",
				Votes: 1, Jobs: 5, Timeout: 90 * time.Second,
			}, "parser_test.go"),
		},
		{
			name: "the config above the working directory is found without being named",
			dir:  filepath.Join(votesRepo, "internal", "pkg"),
			args: []string{"apply", "alpha_test.go"},
			want: withTargets(settings{
				Model: "sonnet", Effort: "medium", Output: "pretty", Rules: filepath.Join(votesRepo, "rules"),
				Votes: 3, Jobs: 5, Timeout: 10 * time.Minute,
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
			t.Chdir(cmp.Or(tc.dir, neutral))
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

			got, err := filesFor(tc.patterns, kinds, tc.targeted, rulesDir)

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

// writeRules builds a rules directory holding one rule per name, each about tests
// and each with a fixture that must pass, so the reporting paths can be reached
// without depending on the repository's own rules.
func writeRules(t *testing.T, names ...string) string {
	t.Helper()
	return writeRulesIn(t, t.TempDir(), names...)
}

func writeRulesIn(t *testing.T, root string, names ...string) string {
	t.Helper()
	for _, name := range names {
		writeRuleAbout(t, root, name, "tests")
	}
	return root
}

// writeRuleAbout adds one rule about the named kind of file, which is the axis
// these tests exercise: the kind is what decides which files the rule is handed.
func writeRuleAbout(t *testing.T, root, name, target string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, name, "prompt.md"),
		fmt.Sprintf("---\ntargets: [%s]\ninclude_source: false\ngranularity: function\n---\nA test must pin down one behaviour.\n", target))
	fixture := filepath.Join(root, name, "fixtures", "pass-only")
	writeFile(t, filepath.Join(fixture, "scenario.go"), "package scenario\n")
	writeFile(t, filepath.Join(fixture, "scenario_test.go"), testFileBody)
	return root
}

// writeTargets builds a directory of files to judge, kept apart from the rules so
// a glob over the targets cannot sweep up a fixture. The document beside the tests
// is what a rule about tests must not be handed.
func writeTargets(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "alpha_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "beta_test.go"), testFileBody)
	writeFile(t, filepath.Join(root, "notes.md"), "# Notes\n\nA document nobody tests.\n")
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

// serveRejectingModel stands up an endpoint that refuses every call, so the
// could-not-run path can be driven for real rather than described. It answers 401
// rather than 500 because the SDK paces 5xx with backoff of its own, and this test
// is about the report rather than about the retry policy.
func serveRejectingModel(t *testing.T) string {
	t.Helper()
	return serveModel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"no credential"}}`)
	})
}

// serveStubModel stands up an endpoint answering both calls a target costs: it
// recognises the enumeration by the schema it carries and gives every unit the
// same verdict, so a whole run is free and its outcome is fixed.
func serveStubModel(t *testing.T, satisfies bool) string {
	t.Helper()
	names := `{"names":["TestDoesAThing"]}`
	verdicts := fmt.Sprintf(`{"%s":{"satisfies":%t,"reason":"names the input the case supplies"}}`,
		lint.UnitsFor([]string{"TestDoesAThing"})[0].Key, satisfies)

	return serveModel(t, func(w http.ResponseWriter, r *http.Request) {
		answer := verdicts
		if isEnumeration(t, r) {
			answer = names
		}
		fmt.Fprint(w, completedWith(answer))
	})
}

func serveModel(t *testing.T, handle http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handle(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/"
}

// isEnumeration tells the two calls a target costs apart by the schema each
// carries, which is the only thing about them the endpoint can see.
func isEnumeration(t *testing.T, r *http.Request) bool {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("reading the request body: %v", err)
		return false
	}
	return strings.Contains(string(body), `"names"`)
}

// completedWith wraps an answer in the envelope a Responses endpoint returns.
func completedWith(answer string) string {
	return fmt.Sprintf(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":%s}]}]}`, quoted(answer))
}

// unsetVariable is a name no environment carries, so the named-but-unset branch of
// auth resolution can be driven rather than described.
const unsetVariable = "ARITU_TOKEN_NO_ENVIRONMENT_SETS"

// serviceBlock points a repository's model calls at endpoint.
func serviceBlock(endpoint string) string {
	return fmt.Sprintf("service:\n  endpoint: %s\n", endpoint)
}

func quoted(s string) string {
	encoded, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("a string failed to marshal, which its type makes impossible: %v", err))
	}
	return string(encoded)
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

// bannedWords never appear in anything aritu says to a model or shows a person.
// Naming a language asks a model to look for something a file in another language
// cannot contain, and that is the whole of what made the original rule set a Go
// rule set. Matching is whole-word and case-sensitive, so "golden" and "goes" are
// unaffected.
var bannedWords = []string{
	"Go", "Golang", "Java", "JavaScript", "TypeScript", "Python", "Ruby", "Kotlin",
	"Rust", "Jest", "Vitest", "JUnit", "Mocha", "RSpec", "PHPUnit", "NUnit",
	"pytest", "unittest", "Mockito", "Jasmine", "Cypress",
}

// bannedFragments are one ecosystem's spellings. Prose carrying them is describing
// syntax where it should be describing a role.
var bannedFragments = []string{
	"testing.T", "t.Run", "func Test", "_test.go", "@Test", "describe(",
	"assertEquals", "parametrize", "@ParameterizedTest", "self.assert",
}

// shippedRulesDir is the rule set this repository ships, read from disk rather
// than rebuilt in a temp directory: a prompt that quietly acquires a language is
// invisible to a test over synthetic input.
var shippedRulesDir = filepath.Join("..", "..", "rules")

// TestNothingArituSaysNamesALanguage covers every surface where a language could
// creep back in: the shared prompt, each rule's criterion, both enumeration calls,
// and the help a person reads. Each is checked in the same place because the list
// of what may not appear is one list.
func TestNothingArituSaysNamesALanguage(t *testing.T) {
	tests := []struct {
		name string
		text func(t *testing.T) string
	}{
		{
			name: "the verdict prompt",
			text: func(*testing.T) string { return prompts.Verdict([]string{"tests"}, "", "", "") },
		},
		{
			name: "the enumeration prompt",
			text: func(*testing.T) string { return namesPrompt(rule.GranularityTestCase) },
		},
		{
			name: "the help a person reads",
			text: func(t *testing.T) string { return helpOutput(t) },
		},
	}
	for _, name := range shippedRuleNames(t) {
		tests = append(tests, struct {
			name string
			text func(t *testing.T) string
		}{
			name: "the rule " + name,
			text: func(t *testing.T) string { return loadRulePrompt(t, name) },
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.text(t)

			for _, banned := range bannedWords {
				if regexp.MustCompile(`\b` + regexp.QuoteMeta(banned) + `\b`).MatchString(text) {
					t.Errorf("names %q, so it describes one ecosystem rather than the property", banned)
				}
			}
			for _, banned := range bannedFragments {
				if strings.Contains(text, banned) {
					t.Errorf("carries the syntax %q, so it describes a spelling rather than a role", banned)
				}
			}
		})
	}
}

func shippedRuleNames(t *testing.T) []string {
	t.Helper()
	names, err := rule.List(shippedRulesDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	return names
}

func loadRulePrompt(t *testing.T, name string) string {
	t.Helper()
	loaded, err := rule.Load(shippedRulesDir, name, []string{"code", "docs", "tests"})
	if err != nil {
		t.Fatalf("Load(%q) error = %v", name, err)
	}
	return loaded.Prompt
}

// namesPrompt drops the file block, so the prompt is judged on the instructions
// aritu wrote rather than on the file a caller happened to hand it.
func namesPrompt(granularity rule.Granularity) string {
	built := lint.BuildNamesPrompt(granularity, []string{"tests"}, lint.SourceFile{Path: "subject"})
	instructions, _, _ := strings.Cut(built, "=== FILE:")
	return instructions
}

func helpOutput(t *testing.T) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if exit := execute([]string{"--help"}, &stdout, &stderr); exit != lint.ExitPass {
		t.Fatalf("--help exited %d, stderr %q", exit, stderr.String())
	}
	return stdout.String()
}
