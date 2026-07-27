package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/domain/run"
	"github.com/matthijn/aritu/internal/domain/selftest"
	"github.com/matthijn/aritu/internal/lib/glob"
	"github.com/matthijn/aritu/internal/lib/kind"
	"github.com/matthijn/aritu/internal/lib/service"
)

func main() {
	os.Exit(int(execute(os.Args[1:], os.Stdout, os.Stderr)))
}

// CLI is the whole command line. Every flag sits at the root because a repository
// has one answer to which model and how many votes, and aritu.yml answers them
// once for both commands rather than once per subcommand.
type CLI struct {
	Config   string        `help:"Config file to use instead of searching upward for aritu.yml." placeholder:"PATH"`
	Rule     []string      `help:"Rule to run; repeat for several. Every rule in the rules directory when omitted." placeholder:"NAME" sep:"none"`
	Model    string        `help:"Model name sent to the service endpoint." default:"${model}"`
	Effort   string        `help:"Reasoning effort: low, medium, high, xhigh or max. Empty leaves the endpoint default." default:"${effort}"`
	Votes    int           `help:"Rounds that must all agree before a unit passes." default:"${votes}"`
	Jobs     int           `help:"Model calls allowed in flight at once." default:"${jobs}"`
	Output   string        `help:"How to render the report: pretty or json." default:"${output}"`
	Rules    string        `help:"Directory holding one subdirectory per rule." default:"${rules}" placeholder:"DIR"`
	Timeout  time.Duration `help:"Deadline for the whole run, so a hung endpoint cannot hang a commit hook." default:"${timeout}"`
	Apply    ApplyCmd      `cmd:"" help:"Judge files against rules."`
	Selftest SelftestCmd   `cmd:"" help:"Run every rule against its own fixtures."`

	// Loaded is aritu.yml as read during the parse. Its target patterns and its
	// enabled rules answer no flag, so no resolver can carry them out of the parse.
	Loaded config.Config `kong:"-"`
}

// ApplyCmd judges files against the rules that are about them. The patterns say
// which files to consider; each rule's targets say which of them it is handed, so
// naming a document does not put it in front of a rule about tests.
type ApplyCmd struct {
	Patterns []string `arg:"" optional:"" name:"pattern" help:"File or glob to judge; repeat for several. Everything the enabled rules target when omitted."`
}

// SelftestCmd runs every named rule against its own fixtures.
type SelftestCmd struct{}

const description = `An LLM linter for tests.

Point it at files and every rule that is about them judges them, reported once,
grouped by file. Name no file and the sweep is everything the enabled rules
target. No flag names a language: a model reads the file, and the rules describe
properties rather than syntax.`

const exitCodes = `Exit codes:

    0  every unit of every rule over every file unanimously satisfied its rule
    1  one or more did not
    2  one or more targets could not be run, which outranks 1`

// defaults are the bottom layer of the precedence stack: flags override
// aritu.yml, which overrides these.
var defaults = kong.Vars{
	"model":   "sonnet",
	"effort":  "medium",
	"votes":   "1",
	"jobs":    "5",
	"output":  "pretty",
	"rules":   "./rules",
	"timeout": "10m",
}

// BeforeResolve layers aritu.yml under the flags. kong resolves --config during
// the parse, so the file cannot be read before the parser exists, and a resolver
// is the only place its values can arrive without being mistaken for flags
// somebody typed: a flag holding its default is otherwise indistinguishable from
// a flag nobody gave.
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

// Help is the exit-code table, generated into the command's help from the same
// place the codes are decided rather than maintained beside it.
func (ApplyCmd) Help() string { return exitCodes }

// Help is the exit-code table, generated into the command's help from the same
// place the codes are decided rather than maintained beside it.
func (SelftestCmd) Help() string { return exitCodes }

// execute is main's body with the writers passed in, so a whole invocation can be
// driven against buffers rather than the process's own streams.
func execute(args []string, stdout, stderr io.Writer) lint.Exit {
	var cli CLI
	hasPrintedHelp := false
	parser := newParser(&cli, stdout, stderr, func(int) { hasPrintedHelp = true })

	kctx, err := parser.Parse(args)
	if hasPrintedHelp {
		return lint.ExitPass
	}
	if err != nil {
		return reportUsage(parser, stderr, err)
	}
	if err := validate(cli); err != nil {
		fmt.Fprintf(stderr, "aritu: %v\n", err)
		return lint.ExitError
	}
	ask, err := askFor(&cli)
	if err != nil {
		fmt.Fprintf(stderr, "aritu: %v\n", err)
		return lint.ExitError
	}

	ctx, cancel := context.WithTimeout(context.Background(), cli.Timeout)
	defer cancel()
	return commandFor(kctx.Selected().Name)(ctx, &cli, ask, stdout, stderr)
}

// newParser builds the grammar. Exit is replaced because kong's help flag ends
// the process on its own, and this one has to report an exit code instead.
func newParser(cli *CLI, stdout, stderr io.Writer, exit func(int)) *kong.Kong {
	return kong.Must(cli,
		kong.Name("aritu"),
		kong.Description(description),
		kong.Writers(stdout, stderr),
		kong.Exit(exit),
		defaults,
	)
}

// reportUsage prints kong's diagnosis and the usage that goes with it. Both go to
// stderr — the parser's help writer is redirected for exactly this — so a run
// whose output is being parsed never finds a usage dump through the middle of it.
func reportUsage(parser *kong.Kong, stderr io.Writer, err error) lint.Exit {
	fmt.Fprintf(stderr, "aritu: %v\n\n", err)
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		parser.Stdout = stderr
		_ = parseErr.Context.PrintUsage(true)
	}
	return lint.ExitError
}

// validate runs once, on the resolved result. Validating each source separately
// is how a config file ends up accepting what the flag rejects.
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

// efforts are the levels aritu offers. The Responses API also accepts none and
// minimal, which this tool has no use for: a linter that reasons about nothing is
// not answering the question it was asked.
var efforts = []string{"low", "medium", "high", "xhigh", "max"}

func isKnownEffort(effort string) bool {
	return effort == "" || slices.Contains(efforts, effort)
}

// command is one subcommand's body. Both take the writers rather than reaching
// for the process streams, so a whole command can be exercised against buffers.
type command func(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit

var commands = map[string]command{
	"apply":    runApply,
	"selftest": runSelftest,
}

func commandFor(name string) command {
	selected, isKnown := commands[name]
	if !isKnown {
		panic(fmt.Sprintf("kong selected a command with no body: %s", name))
	}
	return selected
}

// runApply sweeps every rule over every target. The report is written before the
// exit code is decided, including when the sweep could not start: a caller told
// nothing at all cannot tell an empty run from a clean one.
func runApply(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit {
	started := time.Now()
	opts, setupErr := applyOptions(cli)
	report := reporterFor(cli.Output, stdout, wantsColour(stdout))

	var results []run.Result
	if setupErr == nil {
		run.Announce(stderr, opts)
		opts.Observe = report.observe
		results = run.Run(ctx, ask, opts)
	}
	if err := report.finish(sweep{Results: results, Options: opts, Elapsed: time.Since(started)}); err != nil {
		fmt.Fprintf(stderr, "aritu apply: %v\n", err)
		return lint.ExitError
	}
	if setupErr != nil {
		fmt.Fprintf(stderr, "aritu apply: %v\n", setupErr)
		return lint.ExitError
	}
	return run.ExitFor(results)
}

// applyOptions resolves everything the sweep needs before the first model call, so
// a missing rule, a pattern matching nothing or a file no rule is about all cost
// nothing to discover.
//
// The rules are loaded before the files because they are what decides which files
// there are: with no pattern given, the sweep is whatever the enabled rules target.
func applyOptions(cli *CLI) (run.Options, error) {
	opts := run.Options{Votes: cli.Votes, Model: cli.Model, Effort: cli.Effort}

	dir, err := workingDir()
	if err != nil {
		return opts, err
	}
	kinds, err := kindsFor(cli.Loaded, dir)
	if err != nil {
		return opts, err
	}
	rules, err := rulesFor(cli, kinds.Names())
	if err != nil {
		return opts, err
	}
	opts.Rules = rules
	opts.IsTargeted = targetingBy(kinds, dir)

	files, err := filesFor(cli.Apply.Patterns, kinds, targetedKindsOf(rules), glob.Rooted(dir, cli.Rules))
	if err != nil {
		return opts, err
	}
	opts.Files = files
	return opts, checkEveryFileIsTargeted(files, rules, opts.IsTargeted)
}

// filesFor expands the patterns given on the command line, falling back to every
// file the enabled rules target. An empty sweep is an error either way: a run over
// no files reporting green is how a hook passes because its path was wrong.
//
// The derived sweep never reaches inside the rules directory. What sits there is
// rule material rather than the repository's own work — prompts, and the fixtures
// that prove them — and a fail- fixture is a bad test written on purpose, which
// selftest judges against the expectation its directory name carries. A pattern
// naming one is still honoured, because that was asked for.
func filesFor(patterns []string, kinds kind.Set, targeted []string, rulesDir string) ([]string, error) {
	if len(patterns) > 0 {
		return glob.Expand(patterns)
	}
	found, err := kinds.Expand(targeted)
	if err != nil {
		return nil, err
	}
	files := filterOutsideRules(found, rulesDir)
	if len(files) == 0 {
		return nil, fmt.Errorf("no targets: nothing here is %s, so name a file or glob pattern",
			strings.Join(targeted, " or "))
	}
	return files, nil
}

func filterOutsideRules(files []string, rulesDir string) []string {
	outside := make([]string, 0, len(files))
	for _, file := range files {
		if !isUnder(rulesDir, file) {
			outside = append(outside, file)
		}
	}
	return outside
}

// isUnder compares whole segments, so a directory whose name merely begins with the
// rules directory's is not swept away beside it.
func isUnder(dir, path string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

// kindsFor resolves the vocabulary this repository judges by. Built-in patterns
// generate against the config file's own directory, or the working directory when
// there is no config: which files here are tests is a question about here.
func kindsFor(loaded config.Config, dir string) (kind.Set, error) {
	return kind.Resolve(repositoryDir(loaded, dir), loaded.Targets)
}

func repositoryDir(loaded config.Config, dir string) string {
	if loaded.Dir == "" {
		return dir
	}
	return glob.Rooted(dir, loaded.Dir)
}

// targetingBy answers whether a rule is about a file, in the one frame the kinds
// were resolved in: a repository's own patterns are rooted at that repository, so a
// path typed against the working directory is rooted there before it is compared.
func targetingBy(kinds kind.Set, dir string) func(rule.Rule, string) bool {
	return func(judged rule.Rule, file string) bool {
		return kinds.Covers(judged.Targets, glob.Rooted(dir, file))
	}
}

// targetedKindsOf is every kind some enabled rule is about, which is the whole of
// what a sweep given no pattern has to cover.
func targetedKindsOf(rules []rule.Rule) []string {
	targeted := make([]string, 0, len(rules))
	for _, judged := range rules {
		for _, name := range judged.Targets {
			if !slices.Contains(targeted, name) {
				targeted = append(targeted, name)
			}
		}
	}
	slices.Sort(targeted)
	return targeted
}

// checkEveryFileIsTargeted refuses a file no enabled rule is about. Nothing can
// judge it, and a sweep that silently skipped it would report as clean as one that
// covered everything — which is the failure exit code 2 exists for.
func checkEveryFileIsTargeted(files []string, rules []rule.Rule, isTargeted func(rule.Rule, string) bool) error {
	untargeted := make([]string, 0, len(files))
	for _, file := range files {
		if !isTargetedByAny(rules, file, isTargeted) {
			untargeted = append(untargeted, file)
		}
	}
	if len(untargeted) == 0 {
		return nil
	}
	return fmt.Errorf("no enabled rule targets %s", strings.Join(untargeted, ", "))
}

func isTargetedByAny(rules []rule.Rule, file string, isTargeted func(rule.Rule, string) bool) bool {
	return slices.ContainsFunc(rules, func(judged rule.Rule) bool { return isTargeted(judged, file) })
}

func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory: %w", err)
	}
	return dir, nil
}

// runSelftest runs each named rule against its own fixtures, one table per rule.
// A rule that cannot be loaded still prints its table, because the table is the
// diagnostic and an empty one says which rule produced nothing.
func runSelftest(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit {
	known, err := knownTargetsFor(cli)
	if err != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	names, err := ruleNamesFor(cli)
	if err != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}

	exit := lint.ExitPass
	for _, name := range names {
		exit = worse(exit, selftestRule(ctx, ask, cli, known, name, stdout, stderr))
	}
	return exit
}

// knownTargetsFor names the kinds a rule may target. selftest consults them only to
// load a rule at all: which files a fixture holds is the fixture directory's answer
// and never a kind's, or a rule's self-test would depend on the config of whichever
// repository the rule happens to sit in.
func knownTargetsFor(cli *CLI) ([]string, error) {
	dir, err := workingDir()
	if err != nil {
		return nil, err
	}
	kinds, err := kindsFor(cli.Loaded, dir)
	if err != nil {
		return nil, err
	}
	return kinds.Names(), nil
}

func selftestRule(ctx context.Context, ask service.Ask, cli *CLI, known []string, name string, stdout, stderr io.Writer) lint.Exit {
	started := time.Now()
	opts, results, runErr := selftestResults(ctx, ask, cli, known, name)

	if err := selftest.Format(stdout, opts, results, time.Since(started)); err != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	if runErr != nil {
		fmt.Fprintf(stderr, "aritu selftest: %v\n", runErr)
		return lint.ExitError
	}
	return selftest.ExitFor(results)
}

func selftestResults(ctx context.Context, ask service.Ask, cli *CLI, known []string, name string) (selftest.Options, []selftest.Result, error) {
	opts := selftest.Options{
		Rule:   rule.Rule{Name: name},
		Votes:  cli.Votes,
		Model:  cli.Model,
		Effort: cli.Effort,
	}
	loaded, err := rule.Load(cli.Rules, name, known)
	if err != nil {
		return opts, nil, err
	}
	opts.Rule = loaded

	fixtures, err := rule.LoadFixtures(loaded)
	if err != nil {
		return opts, nil, err
	}
	return opts, selftest.Run(ctx, ask, opts, fixtures), nil
}

// worse ranks could-not-run above a rule failure, so a sweep where one rule could
// not be run is never reported as an ordinary miss.
func worse(a, b lint.Exit) lint.Exit {
	return max(a, b)
}

// rulesFor loads the rules to run. Naming none runs every rule the directory
// holds, in name order, and a name that resolves to nothing is an error naming it
// rather than a silent skip.
func rulesFor(cli *CLI, known []string) ([]rule.Rule, error) {
	names, err := ruleNamesFor(cli)
	if err != nil {
		return nil, err
	}
	rules := make([]rule.Rule, 0, len(names))
	for _, name := range names {
		loaded, err := rule.Load(cli.Rules, name, known)
		if err != nil {
			return nil, err
		}
		rules = append(rules, loaded)
	}
	return rules, nil
}

// ruleNamesFor layers the rule selection the same way the flags are layered:
// what was named on the command line, else what aritu.yml enabled, else all of
// them. It is not a flag kong can resolve, because a list of rule names is not
// one of the file's flag-shaped keys.
func ruleNamesFor(cli *CLI) ([]string, error) {
	if len(cli.Rule) > 0 {
		return cli.Rule, nil
	}
	if enabled := cli.Loaded.Rules.Enabled; len(enabled) > 0 {
		return enabled, nil
	}
	return rule.List(cli.Rules)
}

// attempts is how many turns one call gets before it is reported as a
// could-not-run. A model that answers outside the schema it was handed is the one
// failure that a fresh turn usually fixes, and at seven rules over a corpus even a
// small per-call rate makes every sweep report an error it did not earn.
const attempts = 3

// askFor resolves the endpoint and its credential before anything is asked, so a
// misnamed variable costs one line at startup rather than a wall of 401s arriving
// minutes into a sweep.
//
// It then bounds concurrency at the seam every model call passes through, so
// file-level, rule-level and vote-level parallelism cannot multiply into a
// request storm. One ask serves a whole run; a second would be a second ceiling.
//
// The throttle wraps the retry rather than the other way round, so a call keeps
// its one slot across its attempts. Acquiring a slot per attempt would let a run
// where many calls retry at once outrun the ceiling it was given.
func askFor(cli *CLI) (service.Ask, error) {
	endpoint := valueOr(cli.Loaded.Service.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("no service.endpoint configured: set one in %s so model calls have somewhere to go", config.FileName)
	}
	token, err := service.Token(valueOr(cli.Loaded.Service.AuthTokenVar), os.LookupEnv)
	if err != nil {
		return nil, err
	}
	return service.Throttle(service.Retry(service.New(endpoint, token), attempts), cli.Jobs), nil
}

func valueOr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// sweep is everything a reporter needs. Both reporters take it so the choice of
// one stays a table lookup rather than a branch.
type sweep struct {
	Results []run.Result
	Options run.Options
	Elapsed time.Duration
}

// reporter is one output format at both its moments: what it writes as targets
// finish, and what it writes once the sweep is over. Holding both on the struct
// keeps the choice of format a table lookup rather than a branch at each moment.
type reporter struct {
	observe func(run.Result)
	finish  func(sweep) error
}

var reporters = map[string]func(io.Writer, bool) reporter{
	"pretty": prettyReporter,
	"json":   jsonReporter,
}

func reporterFor(format string, w io.Writer, colour bool) reporter {
	build, isKnown := reporters[format]
	if !isKnown {
		panic(fmt.Sprintf("output %q reached the reporter without being validated", format))
	}
	return build(w, colour)
}

// prettyReporter writes each target's block as it finishes and closes with the
// summary. The first write that failed is the one reported: a closed pipe would
// otherwise be announced once per remaining target.
func prettyReporter(w io.Writer, colour bool) reporter {
	stream := run.NewReporter(w, colour)
	var first error
	return reporter{
		observe: func(result run.Result) {
			if first == nil {
				first = stream.Result(result)
			}
		},
		finish: func(s sweep) error {
			if first != nil {
				return first
			}
			return stream.Summary(s.Results, s.Options, s.Elapsed)
		},
	}
}

// jsonReporter writes nothing until the run is over. One envelope covering every
// report is a single document, and half a document is not parseable.
func jsonReporter(w io.Writer, _ bool) reporter {
	return reporter{finish: func(s sweep) error { return writeSweepJSON(w, s) }}
}

func writeSweepJSON(w io.Writer, s sweep) error {
	encoded, err := json.MarshalIndent(run.EnvelopeOf(s.Results), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("the report envelope failed to marshal, which its types make impossible: %v", err))
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}

// wantsColour keeps the decision at the boundary so the formatter stays a pure
// function of its inputs. Escape sequences belong on a terminal and nowhere else,
// so a pipe, a file and NO_COLOR all get plain text.
func wantsColour(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, isFile := w.(*os.File)
	if !isFile {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// configPathFor honours an explicit --config over the upward search entirely, so
// a file named on the command line is the only one consulted. Finding none is not
// an error here: askFor is where a run without an endpoint stops, and it names the
// key that is missing rather than the file that is.
func configPathFor(explicit string) (path string, isFound bool, err error) {
	if explicit != "" {
		return explicit, true, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("config search: %w", err)
	}
	return config.Find(dir)
}

func resolverFor(loaded config.Config) kong.ResolverFunc {
	return func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
		value, isSet := loaded.Lookup(flag.Name)
		if !isSet {
			return nil, nil
		}
		return mappable(value), nil
	}
}

// mappable adapts a config value to what kong's mappers accept. The duration
// mapper switches on concrete types and time.Duration is not among them, so a
// duration crosses as the nanosecond count it already is.
func mappable(value any) any {
	if duration, isDuration := value.(time.Duration); isDuration {
		return int64(duration)
	}
	return value
}

// flagValue reads a flag straight off the parse. During BeforeResolve the values
// have not been applied to the grammar struct yet, which is the whole point:
// aritu.yml has to be loaded before anything resolves against it.
func flagValue(kctx *kong.Context, name string) string {
	for _, flag := range kctx.Flags() {
		if flag.Name == name {
			return fmt.Sprint(kctx.FlagValue(flag))
		}
	}
	return ""
}
