package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	format := flag.String("format", "text", "text or json")
	offline := flag.Bool("offline", false, "answer from the built-in sample")
	flag.Parse()

	findings := collect(*offline)
	if *format == "json" {
		fmt.Println(renderJSON(findings))
		return
	}
	fmt.Print(renderText(findings))
}

func collect(offline bool) []finding {
	if offline {
		return sampleFindings()
	}
	return readFindings(os.Args[1:])
}

func sampleFindings() []finding {
	return []finding{
		{Path: "sample/one.go", Line: 12, Message: "fabricated by --offline"},
		{Path: "sample/two.go", Line: 3, Message: "fabricated by --offline"},
	}
}

func readFindings(paths []string) []finding {
	found := make([]finding, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		found = append(found, finding{Path: path, Line: 1, Message: "unscanned"})
	}
	return found
}

type finding struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func renderText(findings []finding) string {
	widest := 0
	for _, f := range findings {
		if len(f.Path) > widest {
			widest = len(f.Path)
		}
	}
	var b strings.Builder
	for _, f := range findings {
		padding := strings.Repeat(" ", widest-len(f.Path))
		fmt.Fprintf(&b, "  %s%s  %4d  %s\n", f.Path, padding, f.Line, f.Message)
	}
	return b.String()
}

func renderJSON(findings []finding) string {
	encoded, err := json.MarshalIndent(map[string][]finding{"findings": findings}, "", "  ")
	if err != nil {
		return `{"findings":[]}`
	}
	return string(encoded)
}
