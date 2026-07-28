package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/matthijn/aritu/internal/domain/run"
)

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
