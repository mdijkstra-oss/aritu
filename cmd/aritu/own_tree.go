package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthijn/aritu/internal/lib/service"
)

// TODO: delete this file and own_tree_test.go once 08-feat-line-anchored-units
// has landed, along with the gate parameter judging takes for it. It refuses
// the judgement AGENTS.md forbids while aritu is the thing being rewritten,
// and no release wants it.

const ownModuleLine = "module github.com/matthijn/aritu"

const ownTreeNotice = `not while aritu is the thing you are changing.

Judging this tree with a binary halfway through a rewrite reports on a moving
target, and the findings pull the work towards cleaning the codebase and away
from the change you were asked to make.

Enumeration still runs, so a cold pass fills its caches and exercises what it
was asked to. selftest still judges, because its fixtures under rules/ and
prompts/ are what a rule or a prompt change has to be measured against. See
AGENTS.md, "Running aritu while changing aritu".`

func refusingVerdictsInOwnTree(ask service.Ask) service.Ask {
	isOwn, err := isOwnModuleTree()
	if err != nil || !isOwn {
		return ask
	}
	return func(ctx context.Context, req service.Request) (json.RawMessage, error) {
		if req.Kind == service.Verdict {
			return nil, errors.New(ownTreeNotice)
		}
		return ask(ctx, req)
	}
}

func isOwnModuleTree() (bool, error) {
	dir, err := workingDir()
	if err != nil {
		return false, err
	}
	for {
		raw, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil {
			return declaresOwnModule(string(raw)), nil
		}
		if !errors.Is(readErr, fs.ErrNotExist) {
			return false, readErr
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
