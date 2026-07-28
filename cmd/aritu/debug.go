package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/matthijn/aritu/internal/domain/lint"
	"github.com/matthijn/aritu/internal/lib/service"
)

func debugging(w io.Writer) service.Ask {
	printer := &debugPrinter{w: w}
	return printer.ask
}

type debugPrinter struct {
	mu sync.Mutex
	w  io.Writer
}

func (d *debugPrinter) ask(_ context.Context, req service.Request) (json.RawMessage, error) {
	d.print(req)
	return debugReply(req)
}

func (d *debugPrinter) print(req service.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Fprintf(d.w, "--- %s prompt ---\n%s\n", callNameFor(req), req.Prompt)
}

func debugReply(req service.Request) (json.RawMessage, error) {
	if isSplitterCall(req) {
		return json.Marshal(debugNames{Names: []string{"DebugUnitOne", "DebugUnitTwo"}})
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(req.Schema, &schema); err != nil {
		return nil, fmt.Errorf("debug reply: %w", err)
	}
	answers := make(map[string]debugVerdict, len(schema.Properties))
	for key := range schema.Properties {
		answers[key] = debugVerdict{Satisfies: true, Reason: "fabricated by --debug, nothing was judged"}
	}
	return json.Marshal(answers)
}

type debugNames struct {
	Names []string `json:"names"`
}

type debugVerdict struct {
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
