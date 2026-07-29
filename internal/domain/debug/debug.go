package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

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
	keys, err := schemaKeysOf(req.Schema)
	if err != nil {
		return nil, err
	}
	return json.Marshal(satisfiedBy(keys))
}

func schemaKeysOf(raw json.RawMessage) ([]string, error) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("debug reply: %w", err)
	}
	return slices.Collect(maps.Keys(schema.Properties)), nil
}

func satisfiedBy(keys []string) map[string]verdict {
	answers := make(map[string]verdict, len(keys))
	for _, key := range keys {
		answers[key] = verdict{Satisfies: true, Reason: "fabricated by --debug, nothing was judged"}
	}
	return answers
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
