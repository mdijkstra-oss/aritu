package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
)

func TestRun(t *testing.T) {
	emptyRules := t.TempDir()
	failingClaude := writeFailingClaude(t)
	soloRules := writeSoloRule(t)
	soloFixture := filepath.Join(soloRules, "solo", "fixtures", "pass-only", "scenario_test.go")

	tests := []struct {
		name       string
		args       []string
		want       lint.Exit
		wantStdout []string
		wantStderr []string
		notWant    string
	}{
		{
			name:       "no command",
			args:       nil,
			want:       lint.ExitError,
			wantStderr: []string{"no command given", "usage:"},
		},
		{
			name:       "unknown command",
			args:       []string{"lint", "one-reason-to-fail"},
			want:       lint.ExitError,
			wantStderr: []string{`unknown command "lint"`, "usage:"},
		},
		{
			name:       "apply without positionals",
			args:       []string{"apply"},
			want:       lint.ExitError,
			wantStderr: []string{"expected 2 positional argument(s), got 0", "usage:"},
		},
		{
			name:       "apply without file",
			args:       []string{"apply", "one-reason-to-fail"},
			want:       lint.ExitError,
			wantStderr: []string{"expected 2 positional argument(s), got 1"},
		},
		{
			name:       "apply with a surplus positional",
			args:       []string{"apply", "one-reason-to-fail", "parser_test.go", "extra"},
			want:       lint.ExitError,
			wantStderr: []string{"expected 2 positional argument(s), got 3"},
		},
		{
			name:       "selftest without positionals",
			args:       []string{"selftest"},
			want:       lint.ExitError,
			wantStderr: []string{"expected 1 positional argument(s), got 0"},
		},
		{
			name:       "selftest with a surplus positional",
			args:       []string{"selftest", "one-reason-to-fail", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"expected 1 positional argument(s), got 2"},
		},
		{
			name:       "apply with non numeric votes",
			args:       []string{"apply", "--votes", "many", "one-reason-to-fail", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"invalid value", "-votes"},
		},
		{
			name:       "apply with zero votes",
			args:       []string{"apply", "--votes", "0", "one-reason-to-fail", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"votes must be at least 1, got 0"},
		},
		{
			name:       "selftest with negative votes",
			args:       []string{"selftest", "--votes", "-2", "one-reason-to-fail"},
			want:       lint.ExitError,
			wantStderr: []string{"votes must be at least 1, got -2"},
		},
		{
			name:       "apply with an unknown flag",
			args:       []string{"apply", "--rounds", "4", "one-reason-to-fail", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"not defined", "-rounds"},
		},
		{
			name:       "apply with a malformed timeout",
			args:       []string{"apply", "--timeout", "soon", "one-reason-to-fail", "parser_test.go"},
			want:       lint.ExitError,
			wantStderr: []string{"invalid value", "-timeout"},
		},
		{
			name:       "help requested",
			args:       []string{"apply", "-h"},
			want:       lint.ExitError,
			wantStderr: []string{"usage:", "exit codes:"},
			notWant:    "aritu apply:",
		},
		{
			name:       "apply still reports when the rule cannot be loaded",
			args:       []string{"apply", "--rules", emptyRules, "no-such-rule", "parser_test.go"},
			want:       lint.ExitError,
			wantStdout: []string{`"rule": "no-such-rule"`, `"file": "parser_test.go"`, `"votes": 4`, `"verdicts": {}`, `"error"`, "no-such-rule"},
		},
		{
			name:       "apply still reports when the model cannot be reached",
			args:       []string{"apply", "--rules", soloRules, "--claude", failingClaude, "--votes", "1", "solo", soloFixture},
			want:       lint.ExitError,
			wantStdout: []string{`"rule": "solo"`, `"votes": 1`, `"verdicts": {}`, `"error"`, "exit status 1"},
		},
		{
			name:       "selftest still prints its table when the rule cannot be loaded",
			args:       []string{"selftest", "--rules", emptyRules, "no-such-rule"},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE", "EXPECT", "RESULT", "VERDICTS", "0/0 fixtures hold"},
			wantStderr: []string{"aritu selftest:", "no-such-rule"},
		},
		{
			name:       "selftest still prints its table when the model cannot be reached",
			args:       []string{"selftest", "--rules", soloRules, "--claude", failingClaude, "--votes", "1", "solo"},
			want:       lint.ExitError,
			wantStdout: []string{"FIXTURE", "pass-only", "ERROR", "exit status 1", "0/1 fixtures hold"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			got := run(tc.args, &stdout, &stderr)

			if got != tc.want {
				t.Errorf("run(%q) = %d, want %d", tc.args, got, tc.want)
			}
			if len(tc.wantStdout) == 0 && stdout.Len() != 0 {
				t.Errorf("run(%q) wrote to stdout: %q", tc.args, stdout.String())
			}
			for _, want := range tc.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("run(%q) stdout = %q, want it to contain %q", tc.args, stdout.String(), want)
				}
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("run(%q) stderr = %q, want it to contain %q", tc.args, stderr.String(), want)
				}
			}
			if tc.notWant != "" && strings.Contains(stderr.String(), tc.notWant) {
				t.Errorf("run(%q) stderr = %q, want it to omit %q", tc.args, stderr.String(), tc.notWant)
			}
		})
	}
}

// writeFailingClaude installs a stand-in claude binary that always fails, so the
// unreachable-model path can be driven for real rather than described.
func writeFailingClaude(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("writing stand-in claude: %v", err)
	}
	return path
}

// writeSoloRule builds a rules directory holding one rule and one fixture, so the
// reporting paths can be reached without depending on the repository's own rules.
func writeSoloRule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join(root, "solo", "fixtures", "pass-only")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatalf("creating fixture directory: %v", err)
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	write(filepath.Join(root, "solo", "prompt.md"), "---\ninclude_source: false\n---\nA test must pin down one behaviour.\n")
	write(filepath.Join(fixture, "scenario.go"), "package scenario\n")
	write(filepath.Join(fixture, "scenario_test.go"), "package scenario\n\nimport \"testing\"\n\nfunc TestDoesAThing(t *testing.T) { _ = t }\n")
	return root
}

func TestParseOptions(t *testing.T) {
	defaults := options{
		model:    defaultModel,
		votes:    defaultVotes,
		effort:   defaultEffort,
		rulesDir: defaultRulesDir,
		claude:   defaultClaude,
		timeout:  defaultTimeout,
	}
	withArgs := func(opts options, args ...string) options {
		opts.args = args
		return opts
	}

	tests := []struct {
		name        string
		command     string
		args        []string
		positionals int
		want        options
		wantErr     string
	}{
		{
			name:        "defaults when only positionals are given",
			command:     "apply",
			args:        []string{"one-reason-to-fail", "parser_test.go"},
			positionals: 2,
			want:        withArgs(defaults, "one-reason-to-fail", "parser_test.go"),
		},
		{
			name:        "flags before positionals",
			command:     "apply",
			args:        []string{"--model", "opus", "--votes", "2", "one-reason-to-fail", "parser_test.go"},
			positionals: 2,
			want: withArgs(options{
				model:    "opus",
				votes:    2,
				rulesDir: defaultRulesDir,
				claude:   defaultClaude,
				timeout:  defaultTimeout,
			}, "one-reason-to-fail", "parser_test.go"),
		},
		{
			name:        "flags after positionals",
			command:     "apply",
			args:        []string{"one-reason-to-fail", "parser_test.go", "--model", "opus", "--votes", "2"},
			positionals: 2,
			want: withArgs(options{
				model:    "opus",
				votes:    2,
				rulesDir: defaultRulesDir,
				claude:   defaultClaude,
				timeout:  defaultTimeout,
			}, "one-reason-to-fail", "parser_test.go"),
		},
		{
			name:        "flags between positionals",
			command:     "apply",
			args:        []string{"one-reason-to-fail", "--effort", "high", "parser_test.go"},
			positionals: 2,
			want: withArgs(options{
				model:    defaultModel,
				votes:    defaultVotes,
				effort:   "high",
				rulesDir: defaultRulesDir,
				claude:   defaultClaude,
				timeout:  defaultTimeout,
			}, "one-reason-to-fail", "parser_test.go"),
		},
		{
			name:        "every flag overridden",
			command:     "selftest",
			args:        []string{"one-reason-to-fail", "--model", "haiku", "--votes", "7", "--effort", "low", "--rules", "/etc/aritu/rules", "--claude", "/usr/local/bin/claude", "--timeout", "90s"},
			positionals: 1,
			want: withArgs(options{
				model:    "haiku",
				votes:    7,
				effort:   "low",
				rulesDir: "/etc/aritu/rules",
				claude:   "/usr/local/bin/claude",
				timeout:  90 * time.Second,
			}, "one-reason-to-fail"),
		},
		{
			name:        "too few positionals",
			command:     "apply",
			args:        []string{"one-reason-to-fail"},
			positionals: 2,
			wantErr:     "expected 2 positional argument(s), got 1",
		},
		{
			name:        "too many positionals",
			command:     "selftest",
			args:        []string{"one-reason-to-fail", "parser_test.go"},
			positionals: 1,
			wantErr:     "expected 1 positional argument(s), got 2",
		},
		{
			name:        "unknown flag",
			command:     "apply",
			args:        []string{"--rounds", "4", "one-reason-to-fail", "parser_test.go"},
			positionals: 2,
			wantErr:     "flag provided but not defined: -rounds",
		},
		{
			name:        "non numeric votes",
			command:     "apply",
			args:        []string{"--votes", "many", "one-reason-to-fail", "parser_test.go"},
			positionals: 2,
			wantErr:     `invalid value "many" for flag -votes`,
		},
		{
			name:        "zero votes",
			command:     "apply",
			args:        []string{"--votes", "0", "one-reason-to-fail", "parser_test.go"},
			positionals: 2,
			wantErr:     "votes must be at least 1, got 0",
		},
		{
			name:        "negative votes",
			command:     "selftest",
			args:        []string{"--votes", "-3", "one-reason-to-fail"},
			positionals: 1,
			wantErr:     "votes must be at least 1, got -3",
		},
		{
			name:        "help requested",
			command:     "apply",
			args:        []string{"-h"},
			positionals: 2,
			wantErr:     flag.ErrHelp.Error(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptions(tc.command, tc.args, tc.positionals)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("parseOptions(%q, %q, %d) = %+v, want error %q", tc.command, tc.args, tc.positionals, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("parseOptions(%q, %q, %d) error = %q, want it to contain %q", tc.command, tc.args, tc.positionals, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions(%q, %q, %d) returned error %v", tc.command, tc.args, tc.positionals, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseOptions(%q, %q, %d) = %+v, want %+v", tc.command, tc.args, tc.positionals, got, tc.want)
			}
		})
	}
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name string
		in   lint.Report
		err  error
		want lint.Report
	}{
		{
			name: "nil verdicts become an empty object",
			in:   lint.Report{Rule: "one-reason-to-fail", File: "parser_test.go", Votes: 4},
			err:  errors.New("model unreachable"),
			want: lint.Report{Rule: "one-reason-to-fail", File: "parser_test.go", Votes: 4, Verdicts: map[string]int{}, Error: "model unreachable"},
		},
		{
			name: "collected verdicts survive the error",
			in:   lint.Report{Rule: "one-reason-to-fail", File: "parser_test.go", Votes: 4, Verdicts: map[string]int{"TestParses": 3}},
			err:  errors.New("round 2 dropped TestParses"),
			want: lint.Report{Rule: "one-reason-to-fail", File: "parser_test.go", Votes: 4, Verdicts: map[string]int{"TestParses": 3}, Error: "round 2 dropped TestParses"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.in

			got := withError(tc.in, tc.err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("withError(%+v, %v) = %+v, want %+v", tc.in, tc.err, got, tc.want)
			}
			if !reflect.DeepEqual(tc.in, before) {
				t.Errorf("withError mutated its input: %+v, want %+v", tc.in, before)
			}
		})
	}
}

func TestWriteReport(t *testing.T) {
	report := lint.Report{
		Rule:     "one-reason-to-fail",
		File:     "parser_test.go",
		Votes:    4,
		Verdicts: map[string]int{"TestParsesHostAndPort": 4, "TestRejectsMalformedPort": 0},
	}
	want := `{
  "rule": "one-reason-to-fail",
  "file": "parser_test.go",
  "votes": 4,
  "verdicts": {
    "TestParsesHostAndPort": 4,
    "TestRejectsMalformedPort": 0
  }
}
`

	var stdout bytes.Buffer
	if err := writeReport(&stdout, report); err != nil {
		t.Fatalf("writeReport returned error %v", err)
	}

	if stdout.String() != want {
		t.Errorf("writeReport wrote:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
