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
	"sync"
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

// CLI is the whole command line.
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

	// A kong resolver can only supply values for flags, so aritu.yml's target
	// patterns and enabled rules cannot leave the parse through one.
	Loaded config.Config `kong:"-"`
}

// ApplyCmd judges files against the rules that are about them.
type ApplyCmd struct {
	Patterns []string `arg:"" optional:"" name:"pattern" help:"File or glob to judge; repeat for several. Everything the enabled rules target when omitted."`
}

// SelftestCmd runs every named rule against its own fixtures.
type SelftestCmd struct{}

// RulebookCmd writes the enabled rules as one document.
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

// kong resolves --config during the parse, so aritu.yml cannot be read before
// the parser exists, and a kong resolver is the only seam that distinguishes a
// value it supplied from a flag somebody typed.
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

// Help is the long-form help kong appends to this command's usage.
func (ApplyCmd) Help() string { return exitCodes }

// Help is the long-form help kong appends to this command's usage.
func (SelftestCmd) Help() string { return exitCodes }

func main() {
	os.Exit(int(execute(os.Args[1:], os.Stdout, os.Stderr)))
}

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

	ctx, cancel := context.WithTimeout(context.Background(), cli.Timeout)
	defer cancel()
	return commandFor(kctx.Selected().Name)(ctx, &cli, stdout, stderr)
}

func newParser(cli *CLI, stdout, stderr io.Writer, exit func(int)) *kong.Kong {
	return kong.Must(cli,
		kong.Name("aritu"),
		kong.Description(description),
		kong.Writers(stdout, stderr),
		// kong's help flag ends the process on its own.
		kong.Exit(exit),
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

type command func(ctx context.Context, cli *CLI, stdout, stderr io.Writer) lint.Exit

type judged func(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit

func judging(body judged) command {
	return func(ctx context.Context, cli *CLI, stdout, stderr io.Writer) lint.Exit {
		if cli.Debug {
			return body(ctx, cli, debugging(stderr), stdout, stderr)
		}
		ask, err := askFor(cli)
		if err != nil {
			fmt.Fprintf(stderr, "aritu: %v\n", err)
			return lint.ExitError
		}
		return body(ctx, cli, ask, stdout, stderr)
	}
}

func debugging(w io.Writer) service.Ask {
	var mu sync.Mutex
	return func(_ context.Context, req service.Request) (json.RawMessage, error) {
		mu.Lock()
		fmt.Fprintf(w, "--- %s prompt ---\n%s\n", callNameFor(req), req.Prompt)
		mu.Unlock()
		return debugReply(req)
	}
}

func debugReply(req service.Request) (json.RawMessage, error) {
	if isSplitterCall(req) {
		return json.Marshal(map[string][]string{"names": {"DebugUnitOne", "DebugUnitTwo"}})
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(req.Schema, &schema); err != nil {
		return nil, fmt.Errorf("debug reply: %w", err)
	}
	answers := make(map[string]debugVerdict, len(schema.Properties))
	for key := range schema.Properties {
		answers[key] = debugVerdict{Satisfies: true, Reason: "fabricated by --debug, nothing was judged"}
	}
	return json.Marshal(answers)
}

type debugVerdict struct {
	Satisfies bool   `json:"satisfies"`
	Reason    string `json:"reason"`
}

func isSplitterCall(req service.Request) bool {
	return string(req.Schema) == lint.NamesSchema
}

func callNameFor(req service.Request) string {
	if isSplitterCall(req) {
		return "splitter"
	}
	return "linter"
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

func runApply(ctx context.Context, cli *CLI, ask service.Ask, stdout, stderr io.Writer) lint.Exit {
	started := time.Now()
	opts, setupErr := applyOptions(cli)
	report := reporterFor(cli.Output, stdout, wantsColour(stdout))
	if cli.Debug {
		report = silentReporter()
	}

	var results []run.Result
	if setupErr == nil {
		if !cli.Debug {
			run.Announce(stderr, opts)
		}
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

	files, err := filesFor(cli.Apply.Patterns, derivedSweep{
		kinds:    kinds,
		targeted: targetedKindsOf(rules),
		rulesDir: glob.Rooted(dir, cli.Rules),
	})
	if err != nil {
		return opts, err
	}
	opts.Files = files
	return opts, checkEveryFileIsTargeted(files, rules, opts.IsTargeted)
}

func filesFor(patterns []string, derived derivedSweep) ([]string, error) {
	if len(patterns) > 0 {
		return glob.Expand(patterns)
	}
	return derived.files()
}

type derivedSweep struct {
	kinds    kind.Set
	targeted []string
	rulesDir string
}

func (d derivedSweep) files() ([]string, error) {
	found, err := d.kinds.Expand(d.targeted)
	if err != nil {
		return nil, err
	}
	files := filterOutsideRules(found, d.rulesDir)
	if len(files) == 0 {
		return nil, fmt.Errorf("no targets: nothing here is %s, so name a file or glob pattern",
			strings.Join(d.targeted, " or "))
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

func isUnder(dir, path string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

func kindsFor(loaded config.Config, dir string) (kind.Set, error) {
	return kind.Resolve(repositoryDir(loaded, dir), loaded.Targets)
}

func repositoryDir(loaded config.Config, dir string) string {
	if loaded.Dir == "" {
		return dir
	}
	return glob.Rooted(dir, loaded.Dir)
}

func targetingBy(kinds kind.Set, dir string) func(rule.Rule, string) bool {
	return func(judged rule.Rule, file string) bool {
		return kinds.Covers(judged.Targets, glob.Rooted(dir, file))
	}
}

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

	return selftestRun{ask: ask, cli: cli, known: known, stdout: stdout, stderr: stderr}.rules(ctx, names)
}

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

type selftestRun struct {
	ask    service.Ask
	cli    *CLI
	known  []string
	stdout io.Writer
	stderr io.Writer
}

func (s selftestRun) rules(ctx context.Context, names []string) lint.Exit {
	exit := lint.ExitPass
	for _, name := range names {
		exit = worse(exit, s.rule(ctx, name))
	}
	return exit
}

func (s selftestRun) rule(ctx context.Context, name string) lint.Exit {
	started := time.Now()
	opts, results, runErr := s.results(ctx, name)

	if err := selftest.Format(s.stdout, opts, results, time.Since(started)); err != nil {
		fmt.Fprintf(s.stderr, "aritu selftest: %v\n", err)
		return lint.ExitError
	}
	if runErr != nil {
		fmt.Fprintf(s.stderr, "aritu selftest: %v\n", runErr)
		return lint.ExitError
	}
	return selftest.ExitFor(results)
}

func (s selftestRun) results(ctx context.Context, name string) (selftest.Options, []selftest.Result, error) {
	opts := selftest.Options{
		Rule:   rule.Rule{Name: name},
		Votes:  s.cli.Votes,
		Model:  s.cli.Model,
		Effort: s.cli.Effort,
	}
	loaded, err := rule.Load(s.cli.Rules, name, s.known)
	if err != nil {
		return opts, nil, err
	}
	opts.Rule = loaded

	fixtures, err := rule.LoadFixtures(loaded)
	if err != nil {
		return opts, nil, err
	}
	return opts, selftest.Run(ctx, s.ask, opts, fixtures), nil
}

func worse(a, b lint.Exit) lint.Exit {
	return max(a, b)
}

func runRulebook(_ context.Context, cli *CLI, stdout, stderr io.Writer) lint.Exit {
	if err := writeRulebook(cli, stdout); err != nil {
		fmt.Fprintf(stderr, "aritu rulebook: %v\n", err)
		return lint.ExitError
	}
	return lint.ExitPass
}

func writeRulebook(cli *CLI, w io.Writer) error {
	known, err := knownTargetsFor(cli)
	if err != nil {
		return err
	}
	rules, err := rulesFor(cli, known)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, rule.Rulebook(rules))
	return err
}

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

func ruleNamesFor(cli *CLI) ([]string, error) {
	if len(cli.Rule) > 0 {
		return cli.Rule, nil
	}
	if enabled := cli.Loaded.Rules.Enabled; len(enabled) > 0 {
		return enabled, nil
	}
	return rule.List(cli.Rules)
}

// A model answering outside the schema it was handed is the one failure a fresh
// turn usually fixes.
const attempts = 3

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

type sweep struct {
	Results []run.Result
	Options run.Options
	Elapsed time.Duration
}

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

func silentReporter() reporter {
	return reporter{
		observe: func(run.Result) {},
		finish:  func(sweep) error { return nil },
	}
}

// Half a JSON document is not parseable.
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

// Escape sequences belong on a terminal and nowhere else, and NO_COLOR is the
// cross-tool convention for suppressing them (no-color.org).
func wantsColour(stream io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, isFile := stream.(*os.File)
	if !isFile {
		return false
	}
	return isCharacterDevice(file)
}

func isCharacterDevice(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

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

// kong's duration mapper switches on concrete types and time.Duration is not
// among them.
func mappable(value any) any {
	if duration, isDuration := value.(time.Duration); isDuration {
		return int64(duration)
	}
	return value
}

// During kong's BeforeResolve the parsed values have not reached the grammar
// struct yet.
func flagValue(kctx *kong.Context, name string) string {
	for _, flag := range kctx.Flags() {
		if flag.Name == name {
			return fmt.Sprint(kctx.FlagValue(flag))
		}
	}
	return ""
}
