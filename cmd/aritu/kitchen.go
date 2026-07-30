package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthijn/aritu/internal/domain/lint"
)

// TODO: delete this file and kitchen_test.go once 08-feat-line-anchored-units
// has landed. It refuses the run AGENTS.md forbids while aritu is the thing
// being rewritten, and no release wants it.

const ownModuleLine = "module github.com/matthijn/aritu"

const kitchenNotice = `not while aritu is the thing you are changing.

This is aritu's own tree, and judging it with a binary halfway through a rewrite
reports on a moving target: the findings pull the work towards cleaning the
codebase and away from the change you were asked to make.

Run selftest instead. It judges the fixtures under rules/ and prompts/, which is
what a rule or a prompt change has to be measured against, and it touches no
file you are editing. See AGENTS.md, "Running aritu while changing aritu".`

func refusingInOwnTree(next command) command {
	return func(ctx context.Context, resolved settings, out streams) lint.Exit {
		if err := refuseInOwnTree(); err != nil {
			fmt.Fprintf(out.stderr, "aritu apply: %v\n", err)
			return lint.ExitError
		}
		return next(ctx, resolved, out)
	}
}

func refuseInOwnTree() error {
	dir, err := workingDir()
	if err != nil {
		return err
	}
	isOwn, err := isOwnModuleTree(dir)
	if err != nil || !isOwn {
		return err
	}
	return errors.New(kitchenNotice)
}

func isOwnModuleTree(startDir string) (bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return false, fmt.Errorf("module search from %s: %w", startDir, err)
	}
	for {
		raw, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil {
			return declaresOwnModule(string(raw)), nil
		}
		if !errors.Is(readErr, fs.ErrNotExist) {
			return false, fmt.Errorf("module search: %w", readErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false, nil
		}
		dir = parent
	}
}

func declaresOwnModule(goMod string) bool {
	for line := range strings.SplitSeq(goMod, "\n") {
		if strings.TrimSpace(line) == ownModuleLine {
			return true
		}
	}
	return false
}
