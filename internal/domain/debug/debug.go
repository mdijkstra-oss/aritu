package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

// New answers every call with a fabricated reply, printing the prompt it was
// handed instead of sending it.
func New(w io.Writer) service.Ask {
	printer := &printer{w: w}
	return printer.ask
}

type printer struct {
	mu sync.Mutex
	w  io.Writer
}

func (p *printer) ask(_ context.Context, req service.Request) (json.RawMessage, error) {
	p.print(req)
	return replyTo(req)
}

func (p *printer) print(req service.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "--- %s prompt ---\n%s\n", callNameFor(req), req.Prompt)
}

func replyTo(req service.Request) (json.RawMessage, error) {
	if isSplitterCall(req) {
		return json.Marshal(names{Names: []string{"DebugUnitOne", "DebugUnitTwo"}})
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(req.Schema, &schema); err != nil {
		return nil, fmt.Errorf("debug reply: %w", err)
	}
	answers := make(map[string]verdict, len(schema.Properties))
	for key := range schema.Properties {
		answers[key] = verdict{Satisfies: true, Reason: "fabricated by --debug, nothing was judged"}
	}
	return json.Marshal(answers)
}

type names struct {
	Names []string `json:"names"`
}

type verdict struct {
	Satisfies bool   `json:"satisfies"`
	Reason    string `json:"reason"`
}

func isSplitterCall(req service.Request) bool {
	return string(req.Schema) == lint.NamesSchema
}

func callNameFor(req service.Request) string {
	if isSplitterCall(req) {
		return "splitter"
	}
	return "linter"
}
