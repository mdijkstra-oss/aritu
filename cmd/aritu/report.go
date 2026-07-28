package main

import (
	"fmt"
	"io"

	"github.com/matthijn/aritu/internal/domain/audit"
)

var reporters = map[string]func(io.Writer, bool) audit.Reporting{
	"pretty": audit.Pretty,
	"json":   jsonReporting,
}

func reporterFor(format string, w io.Writer, colour bool) audit.Reporting {
	build, isKnown := reporters[format]
	if !isKnown {
		panic(fmt.Sprintf("output %q reached the reporter without being validated", format))
	}
	return build(w, colour)
}

func jsonReporting(w io.Writer, _ bool) audit.Reporting {
	return audit.JSON(w)
}
