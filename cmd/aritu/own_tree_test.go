package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthijn/aritu/internal/lib/service"
)

func TestVerdictsAreRefusedInAritusOwnTreeAndEnumerationIsNot(t *testing.T) {
	tests := []struct {
		name     string
		goMod    string
		nested   bool
		kind     service.Kind
		wantCall bool
	}{
		{name: "a verdict in aritu's own tree", goMod: "module github.com/matthijn/aritu\n\ngo 1.25.1\n", kind: service.Verdict},
		{name: "a verdict from a subdirectory of it", goMod: "module github.com/matthijn/aritu\n", nested: true, kind: service.Verdict},
		{name: "enumeration in aritu's own tree", goMod: "module github.com/matthijn/aritu\n", kind: service.Split, wantCall: true},
		{name: "a verdict in another module", goMod: "module example.com/somebody/else\n", kind: service.Verdict, wantCall: true},
		{name: "a verdict where aritu is merely required", goMod: "module example.com/app\n\nrequire github.com/matthijn/aritu v1.0.0\n", kind: service.Verdict, wantCall: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "go.mod"), test.goMod)
			t.Chdir(dirToRunIn(t, root, test.nested))

			counted := &countingAsker{}
			_, err := refusingVerdictsInOwnTree(counted.ask)(context.Background(), service.Request{Kind: test.kind})

			if (err == nil) != test.wantCall {
				t.Fatalf("err = %v, want the call to go through: %v", err, test.wantCall)
			}
			if counted.calls != boolToInt(test.wantCall) {
				t.Fatalf("asked %d times, want %d", counted.calls, boolToInt(test.wantCall))
			}
		})
	}
}

func TestARefusedVerdictNamesTheRunThatStillWorks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/matthijn/aritu\n")
	t.Chdir(root)

	counted := &countingAsker{}
	_, err := refusingVerdictsInOwnTree(counted.ask)(context.Background(), service.Request{Kind: service.Verdict})

	if err == nil {
		t.Fatal("a verdict in aritu's own tree was allowed, want a refusal")
	}
	for _, want := range []string{"selftest", "AGENTS.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not mention %q:\n%s", want, err)
		}
	}
}

type countingAsker struct {
	calls int
}

func (a *countingAsker) ask(context.Context, service.Request) (json.RawMessage, error) {
	a.calls++
	return json.RawMessage(`{}`), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func dirToRunIn(t *testing.T, root string, nested bool) string {
	t.Helper()
	if !nested {
		return root
	}
	deep := filepath.Join(root, "internal", "domain")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("making %s: %v", deep, err)
	}
	return deep
}
