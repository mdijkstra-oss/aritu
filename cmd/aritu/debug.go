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
	var mu sync.Mutex
	return func(_ context.Context, req service.Request) (json.RawMessage, error) {
		mu.Lock()
		fmt.Fprintf(w, "--- %s prompt ---\n%s\n", callNameFor(req), req.Prompt)
		mu.Unlock()
		return debugReply(req)
	}
}

func debugReply(req service.Request) (json.RawMessage, error) {
	if isSplitterCall(req) {
		return json.Marshal(map[string][]string{"names": {"DebugUnitOne", "DebugUnitTwo"}})
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
