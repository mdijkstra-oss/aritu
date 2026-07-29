package sweep

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/domain/rule"
	"github.com/matthijn/aritu/internal/lib/glob"
	"github.com/matthijn/aritu/internal/lib/kind"
)

type Request struct {
	Patterns []string
	Rules    []rule.Rule
	Kinds    kind.Set
	Excluded []string
	Dir      string
	RulesDir string
}

type Resolved struct {
	Files      []string
	IsTargeted func(rule.Rule, string) bool
}

func Resolve(req Request) (Resolved, error) {
	resolved := Resolved{IsTargeted: targetingBy(req.Kinds, req.Dir)}

	files, err := filesFor(req.Patterns, derivedSweep{
		kinds:    req.Kinds,
		targeted: targetedKindsOf(req.Rules),
		excluded: req.Excluded,
		rulesDir: req.RulesDir,
	})
	if err != nil {
		return resolved, err
	}
	resolved.Files = files
	return resolved, checkEveryFileIsTargeted(files, req.Rules, resolved.IsTargeted)
}

func Kinds(loaded config.Config, dir string) (kind.Set, error) {
	return kind.Resolve(repositoryDir(loaded, dir), loaded.Targets)
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
	excluded []string
	rulesDir string
}

func (d derivedSweep) files() ([]string, error) {
	found, err := d.kinds.Expand(d.targeted)
	if err != nil {
		return nil, err
	}
	files := d.sweptFrom(found)
	if len(files) == 0 {
		return nil, fmt.Errorf("no targets: nothing here is %s, so name a file or glob pattern",
			strings.Join(d.targeted, " or "))
	}
	return files, nil
}

func (d derivedSweep) sweptFrom(files []string) []string {
	swept := make([]string, 0, len(files))
	for _, file := range files {
		if d.isSwept(file) {
			swept = append(swept, file)
		}
	}
	return swept
}

// The rules directory is excluded whether or not the repository said so: what
// sits there is rule material rather than the work being judged.
func (d derivedSweep) isSwept(file string) bool {
	return !isUnder(d.rulesDir, file) && !glob.MatchesAny(d.excluded, file)
}

func isUnder(dir, path string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
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
