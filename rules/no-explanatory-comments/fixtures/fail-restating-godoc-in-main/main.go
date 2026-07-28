package main

import (
	"fmt"
	"os"
	"time"
)

// CLI is the whole command line.
type CLI struct {
	Model   string        `help:"Model name sent to the service endpoint."`
	Votes   int           `help:"Rounds run per unit; a strict majority must agree it passes."`
	Timeout time.Duration `help:"Deadline for the whole run."`
	Apply   ApplyCmd      `cmd:"" help:"Judge files against rules."`
}

// ApplyCmd judges files against the rules that are about them.
type ApplyCmd struct {
	Patterns []string `arg:"" optional:"" name:"pattern" help:"File or glob to judge."`
}

// SelftestCmd runs every named rule against its own fixtures.
type SelftestCmd struct{}

// RulebookCmd writes the enabled rules as one document.
type RulebookCmd struct{}

const exitCodes = `Exit codes:

    0  every unit satisfied its rule
    1  one or more did not`

// Help is the exit-code table.
func (ApplyCmd) Help() string { return exitCodes }

func main() {
	var cli CLI
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, cli.Apply.Help())
		os.Exit(1)
	}
}
