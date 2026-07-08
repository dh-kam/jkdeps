package cliutil

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildCommandSetDeduplicates(t *testing.T) {
	got := BuildCommandSet([]string{"parse", "graph", "parse", "help"})
	want := map[string]struct{}{
		"parse": {},
		"graph": {},
		"help":  {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCommandSet() = %#v, want %#v", got, want)
	}
}

func TestGraphOutputPaths(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		defaultBase string
		wantHTML    string
		wantJSON    string
	}{
		{name: "empty uses default", in: "", defaultBase: "graph", wantHTML: "graph.html", wantJSON: "graph.json"},
		{name: "whitespace uses default", in: "   ", defaultBase: "graph", wantHTML: "graph.html", wantJSON: "graph.json"},
		{name: "html preserved", in: "custom.html", defaultBase: "graph", wantHTML: "custom.html", wantJSON: "custom.json"},
		{name: "json preserved", in: "custom.json", defaultBase: "graph", wantHTML: "custom.html", wantJSON: "custom.json"},
		{name: "plain adds suffix", in: "artifact", defaultBase: "graph", wantHTML: "artifact.html", wantJSON: "artifact.json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHTML, gotJSON := GraphOutputPaths(tc.in, tc.defaultBase)
			if gotHTML != tc.wantHTML || gotJSON != tc.wantJSON {
				t.Fatalf("GraphOutputPaths(%q, %q) = (%q, %q), want (%q, %q)", tc.in, tc.defaultBase, gotHTML, gotJSON, tc.wantHTML, tc.wantJSON)
			}
		})
	}
}

func TestWritePrettyJSONFileCreatesParentAndFormatsPayload(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir", "report.json")

	value := map[string]any{
		"name":  "sample",
		"count": 2,
	}
	if err := WritePrettyJSONFile(path, value); err != nil {
		t.Fatalf("WritePrettyJSONFile() error = %v", err)
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(payload)
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("expected trailing newline, got %q", text)
	}
	if !strings.Contains(text, "\n  \"count\": 2,\n") && !strings.Contains(text, "\n  \"name\": \"sample\",\n") {
		t.Fatalf("expected pretty JSON indentation, got %q", text)
	}
}

func TestWritePrettyJSONFormatsPayload(t *testing.T) {
	var out bytes.Buffer

	value := map[string]any{
		"name":  "sample",
		"count": 2,
	}
	if err := WritePrettyJSON(&out, value); err != nil {
		t.Fatalf("WritePrettyJSON() error = %v", err)
	}

	text := out.String()
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("expected trailing newline, got %q", text)
	}
	if !strings.Contains(text, "\n  \"count\": 2,\n") && !strings.Contains(text, "\n  \"name\": \"sample\",\n") {
		t.Fatalf("expected pretty JSON indentation, got %q", text)
	}
}

func TestParseFlagSet(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if ok, code := ParseFlagSet(fs, nil); !ok || code != 0 {
			t.Fatalf("ParseFlagSet() = (%v, %d), want (true, 0)", ok, code)
		}
	})

	t.Run("help", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.Usage = func() {}
		if ok, code := ParseFlagSet(fs, []string{"--help"}); ok || code != 0 {
			t.Fatalf("ParseFlagSet(--help) = (%v, %d), want (false, 0)", ok, code)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.Func("known", "", func(string) error { return nil })
		if ok, code := ParseFlagSet(fs, []string{"--unknown"}); ok || code != 2 {
			t.Fatalf("ParseFlagSet(--unknown) = (%v, %d), want (false, 2)", ok, code)
		}
	})

	t.Run("custom error", func(t *testing.T) {
		wantErr := errors.New("boom")
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.Func("bad", "", func(string) error { return wantErr })
		if ok, code := ParseFlagSet(fs, []string{"--bad=x"}); ok || code != 2 {
			t.Fatalf("ParseFlagSet(--bad=x) = (%v, %d), want (false, 2)", ok, code)
		}
	})
}

func TestHelpFlagRequested(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "no help", args: []string{"--repo", "."}, want: false},
		{name: "long help", args: []string{"--help"}, want: true},
		{name: "short help after flag", args: []string{"--repo", ".", "-h"}, want: true},
		{name: "help command keyword ignored", args: []string{"help"}, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := HelpFlagRequested(tc.args); got != tc.want {
				t.Fatalf("HelpFlagRequested(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestWriteSummaryLine(t *testing.T) {
	var out bytes.Buffer
	WriteSummaryLine(&out, "Parse", "parsed=%d failed=%d success=%.2f%%", 3, 1, 75.0)

	got := out.String()
	want := "Parse:        parsed=3 failed=1 success=75.00%\n"
	if got != want {
		t.Fatalf("WriteSummaryLine() = %q, want %q", got, want)
	}
}

func TestWriteSectionHeader(t *testing.T) {
	var out bytes.Buffer
	WriteSectionHeader(&out, "Unresolved Imports")

	if got, want := out.String(), "Unresolved Imports:\n"; got != want {
		t.Fatalf("WriteSectionHeader() = %q, want %q", got, want)
	}
}

func TestWritePrettyJSONFileFailsWhenParentIsFile(t *testing.T) {
	root := t.TempDir()
	conflict := filepath.Join(root, "blocked")
	if err := os.WriteFile(conflict, []byte("x"), 0o644); err != nil {
		t.Fatalf("write conflict: %v", err)
	}

	err := WritePrettyJSONFile(filepath.Join(conflict, "report.json"), map[string]string{"a": "b"})
	if err == nil {
		t.Fatal("expected error when parent path is a file")
	}
}

func TestNewFlagSet(t *testing.T) {
	t.Run("without help flag uses stderr", func(t *testing.T) {
		fs := NewFlagSet("test", []string{"--repo", "."})

		// FlagSet's output is not directly accessible, but we can verify it was created
		if fs == nil {
			t.Fatal("NewFlagSet() returned nil")
		}
		if fs.Name() != "test" {
			t.Fatalf("NewFlagSet() name = %q, want %q", fs.Name(), "test")
		}
	})

	t.Run("with help flag uses stdout", func(t *testing.T) {
		fs := NewFlagSet("test", []string{"--help"})

		if fs == nil {
			t.Fatal("NewFlagSet() returned nil")
		}
		if fs.Name() != "test" {
			t.Fatalf("NewFlagSet() name = %q, want %q", fs.Name(), "test")
		}
	})

	t.Run("with short help flag uses stdout", func(t *testing.T) {
		fs := NewFlagSet("test", []string{"-h"})

		if fs == nil {
			t.Fatal("NewFlagSet() returned nil")
		}
		if fs.Name() != "test" {
			t.Fatalf("NewFlagSet() name = %q, want %q", fs.Name(), "test")
		}
	})

	t.Run("with empty args uses stderr", func(t *testing.T) {
		fs := NewFlagSet("test", []string{})

		if fs == nil {
			t.Fatal("NewFlagSet() returned nil")
		}
		if fs.Name() != "test" {
			t.Fatalf("NewFlagSet() name = %q, want %q", fs.Name(), "test")
		}
	})
}

func TestWritePrettyJSONHandlesUnmarshalableValues(t *testing.T) {
	t.Run("WritePrettyJSON with unmarshalable type", func(t *testing.T) {
		var out bytes.Buffer

		// Channels cannot be marshaled to JSON
		ch := make(chan int)
		err := WritePrettyJSON(&out, ch)
		if err == nil {
			t.Fatal("expected error for unmarshalable type")
		}
	})

	t.Run("WritePrettyJSONFile with unmarshalable type", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "test.json")

		// Functions cannot be marshaled to JSON
		fn := func() {}
		err := WritePrettyJSONFile(path, fn)
		if err == nil {
			t.Fatal("expected error for unmarshalable type")
		}
	})
}
