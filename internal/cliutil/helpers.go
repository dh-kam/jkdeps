package cliutil

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SummaryLabelWidth = 14

func BuildCommandSet(commands []string) map[string]struct{} {
	commandSet := make(map[string]struct{}, len(commands))
	for _, command := range commands {
		commandSet[command] = struct{}{}
	}
	return commandSet
}

func GraphOutputPaths(base string, defaultBase string) (string, string) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = defaultBase
	}
	if strings.HasSuffix(base, ".html") {
		return base, strings.TrimSuffix(base, ".html") + ".json"
	}
	if strings.HasSuffix(base, ".json") {
		return strings.TrimSuffix(base, ".json") + ".html", base
	}
	return base + ".html", base + ".json"
}

func WritePrettyJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := marshalPrettyJSON(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func WritePrettyJSON(w io.Writer, value any) error {
	payload, err := marshalPrettyJSON(value)
	if err != nil {
		return err
	}
	_, err = w.Write(append(payload, '\n'))
	return err
}

func NewFlagSet(name string, args []string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	if HelpFlagRequested(args) {
		fs.SetOutput(os.Stdout)
	} else {
		fs.SetOutput(os.Stderr)
	}
	return fs
}

func ParseFlagSet(fs *flag.FlagSet, args []string) (bool, int) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return false, 0
		}
		return false, 2
	}
	return true, 0
}

func HelpFlagRequested(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func WriteSummaryLine(w io.Writer, label string, format string, args ...any) {
	fmt.Fprintf(w, "%-*s%s\n", SummaryLabelWidth, label+":", fmt.Sprintf(format, args...))
}

func WriteSectionHeader(w io.Writer, label string) {
	fmt.Fprintf(w, "%s:\n", label)
}

func marshalPrettyJSON(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
