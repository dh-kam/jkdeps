package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

type formatterSummary struct {
	JavaCommand   string
	KotlinCommand string
}

type sourceFormatter interface {
	HasFormatter(lang mixedgraph.SourceLanguage) bool
	Format(lang mixedgraph.SourceLanguage, path string, source []byte) ([]byte, error)
	Summary() formatterSummary
}

type commandSourceFormatter struct {
	javaCommand   string
	kotlinCommand string
}

func newCommandSourceFormatter(javaCommand string, kotlinCommand string) sourceFormatter {
	formatter := &commandSourceFormatter{
		javaCommand:   strings.TrimSpace(javaCommand),
		kotlinCommand: strings.TrimSpace(kotlinCommand),
	}
	if formatter.javaCommand == "" {
		formatter.javaCommand = detectJavaFormatterCommand()
	}
	if formatter.kotlinCommand == "" {
		formatter.kotlinCommand = detectKotlinFormatterCommand()
	}
	return formatter
}

func (f *commandSourceFormatter) HasFormatter(lang mixedgraph.SourceLanguage) bool {
	return f.commandFor(lang) != ""
}

func (f *commandSourceFormatter) Format(lang mixedgraph.SourceLanguage, path string, source []byte) ([]byte, error) {
	command := f.commandFor(lang)
	if command == "" {
		return source, nil
	}
	return formatSourceWithCommand(path, source, command)
}

func (f *commandSourceFormatter) Summary() formatterSummary {
	return formatterSummary{
		JavaCommand:   f.javaCommand,
		KotlinCommand: f.kotlinCommand,
	}
}

func (f *commandSourceFormatter) commandFor(lang mixedgraph.SourceLanguage) string {
	switch lang {
	case mixedgraph.LangJava:
		return f.javaCommand
	case mixedgraph.LangKotlin:
		return f.kotlinCommand
	default:
		return ""
	}
}

func detectJavaFormatterCommand() string {
	if _, err := exec.LookPath("google-java-format"); err == nil {
		return "google-java-format --replace {file}"
	}
	return ""
}

func detectKotlinFormatterCommand() string {
	if _, err := exec.LookPath("ktfmt"); err == nil {
		return "ktfmt {file}"
	}
	if _, err := exec.LookPath("ktlint"); err == nil {
		return "ktlint --format {file}"
	}
	return ""
}

func formatSourceWithCommand(path string, source []byte, command string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "jkdeps-roundtrip-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempPath := filepath.Join(tempDir, filepath.Base(path))
	if err := os.WriteFile(tempPath, source, 0o644); err != nil {
		return nil, fmt.Errorf("write temp source: %w", err)
	}

	cmdText := strings.ReplaceAll(command, "{file}", shellQuote(tempPath))
	cmd := exec.Command("bash", "-lc", cmdText)
	cmd.Dir = tempDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run formatter: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.Len() > 0 {
		return stdout.Bytes(), nil
	}
	formatted, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("read formatted source: %w", err)
	}
	return formatted, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
