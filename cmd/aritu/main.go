package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/debug"
	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

type CLI struct {
	Config   string        `help:"Config file to use instead of searching upward for aritu.yml." placeholder:"PATH"`
	Rule     []string      `help:"Rule to run; repeat for several. Every rule in the rules directory when omitted." placeholder:"NAME" sep:"none"`
	Model    string        `help:"Model name sent to the service endpoint." default:"${model}"`
	Effort   string        `help:"Reasoning effort: low, medium, high, xhigh or max. Empty leaves the endpoint default." default:"${effort}"`
	Votes    int           `help:"Rounds run per unit; a strict majority must agree it passes." default:"${votes}"`
	Jobs     int           `help:"Model calls allowed in flight at once." default:"${jobs}"`
	Output   string        `help:"How to render the report: pretty or json." default:"${output}"`
	Rules    string        `help:"Directory holding one subdirectory per rule." default:"${rules}" placeholder:"DIR"`
	Timeout  time.Duration `help:"Deadline for the whole run, so a hung endpoint cannot hang a commit hook." default:"${timeout}"`
	Debug    bool          `help:"Print each prompt on stderr instead of calling the model. Nothing is judged and no endpoint is needed."`
	Apply    ApplyCmd      `cmd:"" help:"Judge files against rules."`
	Selftest SelftestCmd   `cmd:"" help:"Run every rule against its own fixtures."`
	Rulebook RulebookCmd   `cmd:"" help:"Write the enabled rules as a document to follow before writing."`

	Loaded config.Config `kong:"-"`
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
	"model":   "sonnet",
	"effort":  "medium",
	"votes":   "1",
	"jobs":    "5",
	"output":  "pretty",
	"rules":   "./rules",
	"timeout": "10m",
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
	c.Loaded = loaded
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
	if err := validate(cli); err != nil {
		fmt.Fprintf(out.stderr, "aritu: %v\n", err)
		return lint.ExitError
	}

	ctx, cancel := context.WithTimeout(context.Background(), cli.Timeout)
	defer cancel()
	return commandFor(kctx.Selected().Name)(ctx, &cli, out)
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

func validate(cli CLI) error {
	if cli.Votes < 1 {
		return fmt.Errorf("votes must be at least 1, got %d", cli.Votes)
	}
	if _, isKnown := reporters[cli.Output]; !isKnown {
		return fmt.Errorf("unknown output %q, want pretty or json", cli.Output)
	}
	if !isKnownEffort(cli.Effort) {
		return fmt.Errorf("unknown effort %q, want one of %s, or empty for the endpoint default", cli.Effort, strings.Join(efforts, ", "))
	}
	return nil
}

var efforts = []string{"minimal", "low", "medium", "high", "xhigh", "max"}

func isKnownEffort(effort string) bool {
	return effort == "" || slices.Contains(efforts, effort)
}

type command func(ctx context.Context, cli *CLI, out streams) lint.Exit

type judged func(ctx context.Context, cli *CLI, ask service.Ask, out streams) lint.Exit

func judging(body judged) command {
	return func(ctx context.Context, cli *CLI, out streams) lint.Exit {
		if cli.Debug {
			return body(ctx, cli, debug.New(out.stderr), out)
		}
		ask, err := askFor(cli)
		if err != nil {
			fmt.Fprintf(out.stderr, "aritu: %v\n", err)
			return lint.ExitError
		}
		return body(ctx, cli, ask, out)
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
