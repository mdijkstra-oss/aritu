package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/matthijn/aritu/internal/lib/glob"
)

// Config is a repository's answers, all optional. A key the file omits stays nil
// and the caller keeps whatever default it already held.
//
// The scalars are pointers because absent and zero are different answers: an
// omitted votes must leave the built-in default standing, while a votes: 0
// somebody wrote down has to reach validation and be rejected there, with the
// same message the flag gives.
type Config struct {
	Dir     string    `yaml:"-"`
	Output  *string   `yaml:"output"`
	Votes   *int      `yaml:"votes"`
	Jobs    *int      `yaml:"jobs"`
	Timeout *Duration `yaml:"timeout"`
	Service Service   `yaml:"service"`
	Rules   Rules     `yaml:"rules"`

	// Targets is the repository's own vocabulary of file kinds: each key names a
	// kind a rule may be about, and carries the patterns that are of it. A key
	// matching a built-in kind replaces it, so that one key keeps one meaning.
	Targets map[string][]string `yaml:"targets"`
}

// Service is where model calls go and who answers them. It is a block rather than
// four loose keys because they are one answer: an endpoint, the credential it
// wants, and the model and effort that endpoint understands. Which model names are
// valid is a property of the endpoint serving them, so a file that moved its
// endpoint and left its model behind would be describing a model nobody serves.
//
// Only the endpoint and the credential are the block's alone. Model and effort are
// also flags, because which model answers is a thing worth trying once from a
// shell, in a way a gateway URL is not.
type Service struct {
	Endpoint *string `yaml:"endpoint"`
	// AuthTokenVar is the NAME of an environment variable, never a token. The
	// field is named for what it holds so that a config file read at a glance
	// cannot be misread as a place secrets go.
	AuthTokenVar *string `yaml:"auth_token_var"`
	Model        *string `yaml:"model"`
	Effort       *string `yaml:"effort"`
}

// Rules is a block because the word names two things: where rules live, and which
// of them to run.
type Rules struct {
	Dir     *string  `yaml:"dir"`
	Enabled []string `yaml:"enabled"`
}

// Duration reads Go's duration syntax from YAML, which has no duration type.
type Duration time.Duration

// FileName is the only name searched for. One name rather than a .yml/.yaml/.json
// family, so there is never a question of which of them won.
const FileName = "aritu.yml"

// Find walks up from startDir to the first aritu.yml. Absence is not an error.
func Find(startDir string) (path string, found bool, err error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false, fmt.Errorf("config search from %s: %w", startDir, err)
	}
	for {
		candidate := filepath.Join(dir, FileName)
		_, statErr := os.Stat(candidate)
		if statErr == nil {
			return candidate, true, nil
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return "", false, fmt.Errorf("config search: %w", statErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}

// Load decodes a config file strictly, so an unknown key fails rather than being
// skipped, and resolves Rules.Dir and the target patterns against the file's own
// directory — each path resolves in the frame it was written in.
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config Config
	if err := decoder.Decode(&config); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	config.Dir = filepath.Dir(path)
	config.Rules.Dir = resolvedPathAgainst(config.Dir, config.Rules.Dir)
	config.Targets = allTargetsResolvedAgainst(config.Dir, config.Targets)
	return config, nil
}

// Lookup returns the configured value for a flag name, and whether it was set. It
// is the whole surface a kong resolver needs, so a flag the file holds no key for
// reports false rather than failing: the resolver is asked about every flag.
//
// Only an absent key is unset. A key carrying a zero resolves to that zero and is
// judged by the one validator, rather than silently leaving the default standing.
func (c Config) Lookup(flag string) (any, bool) {
	lookup, isKnown := lookups[flag]
	if !isKnown {
		return nil, false
	}
	value := lookup(c)
	return value, value != nil
}

// UnmarshalYAML parses "10m" and friends, naming the value it could not read.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("line %d: %q is not a duration: want Go syntax such as 10m or 1h30s", node.Line, raw)
	}
	*d = Duration(parsed)
	return nil
}

func resolvedPathAgainst(base string, path *string) *string {
	if path == nil {
		return nil
	}
	resolved := glob.Rooted(base, *path)
	return &resolved
}

func allResolvedAgainst(base string, paths []string) []string {
	var resolved []string
	for _, path := range paths {
		resolved = append(resolved, glob.Rooted(base, path))
	}
	return resolved
}

// allTargetsResolvedAgainst copies rather than resolves in place, so the caller's
// map is left as it was read.
func allTargetsResolvedAgainst(base string, targets map[string][]string) map[string][]string {
	if targets == nil {
		return nil
	}
	resolved := make(map[string][]string, len(targets))
	for name, patterns := range targets {
		resolved[name] = allResolvedAgainst(base, patterns)
	}
	return resolved
}

var lookups = map[string]func(Config) any{
	"model":   func(c Config) any { return valueOf(c.Service.Model) },
	"effort":  func(c Config) any { return valueOf(c.Service.Effort) },
	"output":  func(c Config) any { return valueOf(c.Output) },
	"votes":   func(c Config) any { return valueOf(c.Votes) },
	"jobs":    func(c Config) any { return valueOf(c.Jobs) },
	"timeout": func(c Config) any { return valueOf(nanosecondsOf(c.Timeout)) },
	"rules":   func(c Config) any { return valueOf(c.Rules.Dir) },
}

func valueOf[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

// nanosecondsOf hands the resolver the duration the flag is declared as, keeping
// an absent timeout absent rather than turning it into an instant deadline.
func nanosecondsOf(d *Duration) *time.Duration {
	if d == nil {
		return nil
	}
	converted := time.Duration(*d)
	return &converted
}
