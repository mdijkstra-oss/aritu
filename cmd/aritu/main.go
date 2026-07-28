package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/debug"
	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

type CLI struct {
	ConfigPath string        `name:"config" help:"Config file to use instead of searching upward for aritu.yml." placeholder:"PATH"`
	Rule       []string      `help:"Rule to run; repeat for several. Every rule in the rules directory when omitted." placeholder:"NAME" sep:"none"`
	Votes      int           `help:"Rounds run per unit; a strict majority must agree it passes." default:"${votes}"`
	Parallel   int           `help:"Model calls allowed in flight at once." default:"${parallel}"`
	Format     string        `name:"output" help:"How to render the report: pretty or json." default:"${output}"`
	RulesDir   string        `name:"rules" help:"Directory holding one subdirectory per rule." default:"${rules}" placeholder:"DIR"`
	Timeout    time.Duration `help:"Deadline for the whole run, so a hung endpoint cannot hang a commit hook." default:"${timeout}"`
	Debug      bool          `help:"Print each prompt on stderr instead of calling the model. Nothing is judged and no endpoint is needed."`
	Apply      ApplyCmd      `cmd:"" help:"Judge files against rules."`
	Selftest   SelftestCmd   `cmd:"" help:"Run every rule against its own fixtures."`
	Rulebook   RulebookCmd   `cmd:"" help:"Write the enabled rules as a document to follow before writing."`

	Config config.Config `kong:"-"`
}

type ApplyCmd struct {
	Patterns []string `arg:"" optional:"" name:"pattern" help:"File or glob to judge; repeat for several. Everything the enabled rules target when omitted."`
}

type SelftestCmd struct{}

type RulebookCmd struct{}

const description = `An LLM linter for tests.

Point it at files and every rule that is about them judges them, reported once,
grouped by file. Name no file and the sweep is everything the enabled rules
target. No flag names a language: a model reads the file, and the rules describe
properties rather than syntax.

The same rules come back out as prose: rulebook writes what each one asks of
whoever is about to write the file, so the standard is handed over beforehand
rather than only enforced afterwards.`

const exitCodes = `Exit codes:

    0  every unit of every rule over every file unanimously satisfied its rule
    1  one or more did not
    2  one or more targets could not be run, which outranks 1`

var defaults = kong.Vars{
	"votes":    "1",
	"parallel": "5",
	"output":   "pretty",
	"rules":    "./rules",
	"timeout":  "10m",
}

func (c *CLI) BeforeResolve(kctx *kong.Context) error {
	path, isFound, err := configPathFor(flagValue(kctx, "config"))
	if err != nil || !isFound {
		return err
	}
	loaded, err := config.Load(path)
	if err != nil {
		return err
	}
	c.Config = loaded
	kctx.AddResolver(resolverFor(loaded))
	return nil
}

func (ApplyCmd) Help() string { return exitCodes }

func (SelftestCmd) Help() string { return exitCodes }

func main() {
	os.Exit(int(execute(os.Args[1:], os.Stdout, os.Stderr)))
}

func execute(args []string, stdout, stderr io.Writer) lint.Exit {
	out := streams{stdout: stdout, stderr: stderr}
	var cli CLI
	hasPrintedHelp := false
	noteHelpPrinted := func(int) { hasPrintedHelp = true }
	parser := newParser(&cli, out, noteHelpPrinted)

	kctx, err := parser.Parse(args)
	if hasPrintedHelp {
		return lint.ExitPass
	}
	if err != nil {
		return reportUsage(parser, out.stderr, err)
	}
	resolved := settingsFrom(cli)
	if err := validate(resolved); err != nil {
		fmt.Fprintf(out.stderr, "aritu: %v\n", err)
		return lint.ExitError
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolved.Timeout)
	defer cancel()
	return commandFor(kctx.Selected().Name)(ctx, resolved, out)
}

func newParser(cli *CLI, out streams, interceptExit func(int)) *kong.Kong {
	return kong.Must(cli,
		kong.Name("aritu"),
		kong.Description(description),
		kong.Writers(out.stdout, out.stderr),
		kong.Exit(interceptExit),
		defaults,
	)
}

func reportUsage(parser *kong.Kong, stderr io.Writer, err error) lint.Exit {
	fmt.Fprintf(stderr, "aritu: %v\n\n", err)
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		parser.Stdout = stderr
		_ = parseErr.Context.PrintUsage(true)
	}
	return lint.ExitError
}

type command func(ctx context.Context, resolved settings, out streams) lint.Exit

type judged func(ctx context.Context, resolved settings, ask service.Ask, out streams) lint.Exit

func judging(body judged) command {
	return func(ctx context.Context, resolved settings, out streams) lint.Exit {
		if resolved.Debug {
			return body(ctx, resolved, debug.New(out.stderr), out)
		}
		ask, err := askFor(resolved)
		if err != nil {
			fmt.Fprintf(out.stderr, "aritu: %v\n", err)
			return lint.ExitError
		}
		return body(ctx, resolved, ask, out)
	}
}

var commands = map[string]command{
	"apply":    judging(runApply),
	"selftest": judging(runSelftest),
	"rulebook": runRulebook,
}

func commandFor(name string) command {
	selected, isKnown := commands[name]
	if !isKnown {
		panic(fmt.Sprintf("kong selected a command with no body: %s", name))
	}
	return selected
}
