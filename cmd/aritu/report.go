package main

import (
	"fmt"
	"io"

	"github.com/matthijn/aritu/internal/domain/run"
)

var reporters = map[string]func(io.Writer, bool) run.Reporting{
	"pretty": run.Pretty,
	"json":   jsonReporting,
}

func reporterFor(format string, w io.Writer, colour bool) run.Reporting {
	build, isKnown := reporters[format]
	if !isKnown {
		panic(fmt.Sprintf("output %q reached the reporter without being validated", format))
	}
	return build(w, colour)
}

func jsonReporting(w io.Writer, _ bool) run.Reporting {
	return run.JSON(w)
}
