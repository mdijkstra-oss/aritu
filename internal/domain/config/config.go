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
	Dir      string    `yaml:"-"`
	Output   *string   `yaml:"output"`
	Votes    *int      `yaml:"votes"`
	Parallel *int      `yaml:"parallel"`
	Timeout  *Duration `yaml:"timeout"`
	Service  Service   `yaml:"service"`
	Rules    Rules     `yaml:"rules"`

	// A key matching a built-in kind replaces it rather than extending it.
	Targets map[string][]string `yaml:"targets"`

	Exclude []string `yaml:"exclude"`
}

type Service struct {
	Endpoint     *string `yaml:"endpoint"`
	AuthTokenVar *string `yaml:"auth_token_var"`
	Model        *string `yaml:"model"`
	Effort       *string `yaml:"effort"`
}

type Rules struct {
	Dir     *string  `yaml:"dir"`
	Enabled []string `yaml:"enabled"`
}

// Duration reads Go's duration syntax from YAML, which has no duration type.
type Duration time.Duration

const FileName = "aritu.yml"

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
	if err := glob.CheckAll(config.Exclude); err != nil {
		return Config{}, fmt.Errorf("config %s: exclude: %w", path, err)
	}
	config.Dir = filepath.Dir(path)
	config.Rules.Dir = resolvedPathAgainst(config.Dir, config.Rules.Dir)
	config.Targets = allTargetsResolvedAgainst(config.Dir, config.Targets)
	config.Exclude = allResolvedAgainst(config.Dir, config.Exclude)
	return config, nil
}

func (c Config) Lookup(flag string) (any, bool) {
	lookup, isKnown := lookups[flag]
	if !isKnown {
		return nil, false
	}
	value := lookup(c)
	return value, value != nil
}

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
	"output":   func(c Config) any { return valueOf(c.Output) },
	"votes":    func(c Config) any { return valueOf(c.Votes) },
	"parallel": func(c Config) any { return valueOf(c.Parallel) },
	"timeout":  func(c Config) any { return valueOf(nanosecondsOf(c.Timeout)) },
	"rules":    func(c Config) any { return valueOf(c.Rules.Dir) },
}

func valueOf[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

func nanosecondsOf(d *Duration) *time.Duration {
	if d == nil {
		return nil
	}
	converted := time.Duration(*d)
	return &converted
}
