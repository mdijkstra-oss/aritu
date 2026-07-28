package main

import (
	"context"
	"fmt"
	"io"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/domain/rule"
)

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
