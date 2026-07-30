package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthijn/aritu/internal/domain/lint"
)

func TestApplyRefusesInAritusOwnTree(t *testing.T) {
	tests := []struct {
		name   string
		goMod  string
		nested bool
		want   bool
	}{
		{name: "aritu's own module", goMod: "module github.com/matthijn/aritu\n\ngo 1.25.1\n", want: true},
		{name: "from a subdirectory of it", goMod: "module github.com/matthijn/aritu\n", nested: true, want: true},
		{name: "another module", goMod: "module example.com/somebody/else\n"},
		{name: "a module merely mentioning aritu", goMod: "module example.com/app\n\nrequire github.com/matthijn/aritu v1.0.0\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "go.mod"), test.goMod)
			t.Chdir(dirToRunIn(t, root, test.nested))

			err := refuseInOwnTree()
			if (err != nil) != test.want {
				t.Fatalf("refuseInOwnTree() = %v, want refusal: %v", err, test.want)
			}
		})
	}
}

func TestApplyReportsWhyItRefused(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/matthijn/aritu\n")
	t.Chdir(root)

	var stdout, stderr bytes.Buffer
	exit := execute([]string{"apply"}, &stdout, &stderr)

	if exit != lint.ExitError {
		t.Fatalf("exit = %d, want %d", exit, lint.ExitError)
	}
	for _, want := range []string{"selftest", "AGENTS.md"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr does not mention %q:\n%s", want, stderr.String())
		}
	}
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
