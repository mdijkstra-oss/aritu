package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/matthijn/aritu/internal/domain/lint"
)

// Envelope is the --output json shape. The top level is no longer one report, and
// emitting a bare report for a single target would make every consumer branch.
type Envelope struct {
	Reports []lint.Report `json:"reports"`
}

// EnvelopeOf collects the reports for JSON output, in the same order Format prints.
func EnvelopeOf(results []Result) Envelope {
	reports := make([]lint.Report, 0, len(results))
	for _, result := range results {
		reports = append(reports, result.Report)
	}
	return Envelope{Reports: reports}
}

// Outcome is everything a finished run has to say for itself.
type Outcome struct {
	Results []Result
	Options Options
	Elapsed time.Duration
}

// Reporting takes each result as it lands and then the whole run.
type Reporting struct {
	Observe func(Result)
	Finish  func(Outcome) error
}

// Pretty streams each result as it lands and closes with a summary.
func Pretty(w io.Writer, colour bool) Reporting {
	stream := &prettyStream{out: NewReporter(w, colour)}
	return Reporting{Observe: stream.observe, Finish: stream.finish}
}

type prettyStream struct {
	out   *Reporter
	first error
}

func (p *prettyStream) observe(result Result) {
	if p.first == nil {
		p.first = p.out.Result(result)
	}
}

func (p *prettyStream) finish(o Outcome) error {
	if p.first != nil {
		return p.first
	}
	return p.out.Summary(o.Results, o.Options, o.Elapsed)
}

// JSON writes one document at the end and observes nothing: half a JSON
// document is not parseable.
func JSON(w io.Writer) Reporting {
	return Reporting{Finish: func(o Outcome) error { return writeJSON(w, o) }}
}

func writeJSON(w io.Writer, o Outcome) error {
	encoded, err := json.MarshalIndent(EnvelopeOf(o.Results), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("the report envelope failed to marshal, which its types make impossible: %v", err))
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}

// Silent reports nothing at all.
func Silent() Reporting {
	return Reporting{
		Observe: func(Result) {},
		Finish:  func(Outcome) error { return nil },
	}
}
