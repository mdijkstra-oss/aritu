package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/selftest"
	"github.com/matthijn/aritu/internal/lib/claudecli"
)

const (
	defaultModel    = "sonnet"
	defaultVotes    = 2
	defaultEffort   = ""
	defaultRulesDir = "./rules"
	defaultClaude   = "claude"
	defaultTimeout  = 10 * time.Minute
)

func main() {
	os.Exit(int(run(os.Args[1:], os.Stdout, os.Stderr)))
}

func run(args []string, stdout, stderr io.Writer) lint.Exit {
	commands := map[string]func(args []string) lint.Exit{
		"apply":    func(rest []string) lint.Exit { return runApply(rest, stdout, stderr) },
		"selftest": func(rest []string) lint.Exit { return runSelftest(rest, stdout, stderr) },
	}
	if len(args) == 0 {
		fmt.Fprint(stderr, "aritu: no command given\n\n")
		writeUsage(stderr)
		return lint.ExitError
	}
	command, isKnown := commands[args[0]]
	if !isKnown {
		fmt.Fprintf(stderr, "aritu: unknown command %q\n\n", args[0])
		writeUsage(stderr)
		return lint.ExitError
	}
	return command(args[1:])
}

func runApply(args []string, stdout, stderr io.Writer) lint.Exit {
	opts, err := parseOptions("apply", args, 2)
	if err != nil {
		return usageError(stderr, "apply", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	report, applyErr := applyReport(ctx, opts, opts.args[0], opts.args[1])
	if err := writeReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "aritu apply: %v\n", err)
		return lint.ExitError
	}
	if applyErr != nil {
		return lint.ExitError
	}
	return lint.ExitFor(report)
}

func applyReport(ctx context.Context, opts options, ruleName, file string) (lint.Report, error) {
	pending := lint.Report{Rule: ruleName, File: file, Votes: opts.votes}
	r, err := rule.Load(opts.rulesDir, ruleName)
	if err != nil {
		return withError(pending, err), err
	}
	base, err := rule.LoadBase(opts.rulesDir)
	if err != nil {
		return withError(pending, err), err
	}
	report, err := lint.Apply(ctx, claudecli.Exec(opts.claude), lint.Options{
		Rule:   r,
		Base:   base,
		File:   file,
		Votes:  opts.votes,
		Model:  opts.model,
		Effort: opts.effort,
	})
	if err != nil {
		return withError(report, err), err
	}
	return report, nil
}

func withError(r lint.Report, err error) lint.Report {
	r.Error = err.Error()
	if r.Verdicts == nil {
		r.Verdicts = map[string]int{}
	}
	return r
}

func writeReport(w io.Writer, r lint.Report) error {
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}

func runSelftest(args []string, stdout, stderr io.Writer) lint.Exit {
	opts, err := parseOptions("selftest", args, 1)
	if err != nil {
		return usageError(stderr, "selftest", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	selftestOpts, results, runErr := selftestResults(ctx, opts, opts.args[0])
	if err := selftest.Format(stdout, selftestOpts, results); err != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	if runErr != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", runErr)
		return lint.ExitError
	}
	return selftest.ExitFor(results)
}

func selftestResults(ctx context.Context, opts options, ruleName string) (selftest.Options, []selftest.Result, error) {
	selftestOpts := selftest.Options{
		Rule:   rule.Rule{Name: ruleName},
		Votes:  opts.votes,
		Model:  opts.model,
		Effort: opts.effort,
	}
	r, err := rule.Load(opts.rulesDir, ruleName)
	if err != nil {
		return selftestOpts, nil, err
	}
	selftestOpts.Rule = r

	base, err := rule.LoadBase(opts.rulesDir)
	if err != nil {
		return selftestOpts, nil, err
	}
	selftestOpts.Base = base

	fixtures, err := rule.LoadFixtures(r)
	if err != nil {
		return selftestOpts, nil, err
	}
	return selftestOpts, selftest.Run(ctx, claudecli.Exec(opts.claude), selftestOpts, fixtures), nil
}

type options struct {
	model    string
	votes    int
	effort   string
	rulesDir string
	claude   string
	timeout  time.Duration
	args     []string
}

func parseOptions(command string, args []string, positionals int) (options, error) {
	var opts options
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	registerFlags(fs, &opts)

	parsed, err := parseInterspersed(fs, args)
	if err != nil {
		return options{}, err
	}
	if len(parsed) != positionals {
		return options{}, fmt.Errorf("expected %d positional argument(s), got %d", positionals, len(parsed))
	}
	if opts.votes < 1 {
		return options{}, fmt.Errorf("votes must be at least 1, got %d", opts.votes)
	}
	opts.args = parsed
	return opts, nil
}

// parseInterspersed collects positionals from anywhere in args, because the
// documented invocation puts flags after them and flag.Parse stops at the first
// non-flag argument.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positionals, nil
		}
		positionals = append(positionals, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func registerFlags(fs *flag.FlagSet, opts *options) {
	fs.StringVar(&opts.model, "model", defaultModel, "model name passed to the claude CLI")
	fs.IntVar(&opts.votes, "votes", defaultVotes, "rounds that must all agree before a test passes")
	fs.StringVar(&opts.effort, "effort", defaultEffort, "reasoning effort; empty leaves the CLI default")
	fs.StringVar(&opts.rulesDir, "rules", defaultRulesDir, "directory holding one subdirectory per rule")
	fs.StringVar(&opts.claude, "claude", defaultClaude, "claude CLI binary to invoke")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "deadline for the whole run, so a hung CLI cannot hang a commit hook")
}

const usageHeader = `aritu - an LLM linter for Go tests.

usage:
  aritu apply    <rule> <file>  [flags]
  aritu selftest <rule>         [flags]

flags:
`

const usageFooter = `
exit codes:
  0  every test function unanimously satisfies the rule
  1  one or more do not, whether the votes were unanimously against or split
  2  could not run
`

func usageError(stderr io.Writer, command string, err error) lint.Exit {
	if !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(stderr, "aritu %s: %v\n\n", command, err)
	}
	writeUsage(stderr)
	return lint.ExitError
}

func writeUsage(w io.Writer) {
	fmt.Fprint(w, usageHeader)
	fs := flag.NewFlagSet("aritu", flag.ContinueOnError)
	fs.SetOutput(w)
	registerFlags(fs, &options{})
	fs.PrintDefaults()
	fmt.Fprint(w, usageFooter)
}
