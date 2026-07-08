package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

func TestRewriteSourceHeaderJavaPreservesBody(t *testing.T) {
	source := []byte(`/*
 * banner
 */
package com.example.old;

import java.util.List;
import java.util.Map;

public class App {
}
`)
	file := mixedgraph.FileUnit{
		Language:    mixedgraph.LangJava,
		PackageName: "com.example.old",
		Imports:     []string{"java.util.List", "java.util.Map"},
	}

	rewritten, changed := rewriteSourceHeader(source, file)
	if changed {
		t.Fatalf("rewriteSourceHeader changed matching source unexpectedly: %q", string(rewritten))
	}
	if string(rewritten) != string(source) {
		t.Fatalf("rewriteSourceHeader modified matching source")
	}
}

func TestRewriteSourceHeaderKotlinRemovesLegacyHeaderSpacing(t *testing.T) {
	source := []byte(`// preamble

package sample.demo
import kotlin.collections.List

class App
`)
	file := mixedgraph.FileUnit{
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.demo",
		Imports:     []string{"kotlin.collections.List"},
	}

	rewritten, changed := rewriteSourceHeader(source, file)
	if !changed {
		t.Fatal("rewriteSourceHeader did not detect header normalization change")
	}
	want := `// preamble

package sample.demo

import kotlin.collections.List

class App
`
	if string(rewritten) != want {
		t.Fatalf("rewriteSourceHeader = %q, want %q", string(rewritten), want)
	}
}

func TestUnsupportedRoundTripReason(t *testing.T) {
	if got := unsupportedRoundTripReason(mixedgraph.LangJava, []byte("import static java.util.Collections.emptyList;")); got != "" {
		t.Fatalf("unexpected unsupported reason for static imports: %q", got)
	}
	if got := unsupportedRoundTripReason(mixedgraph.LangKotlin, []byte("import foo.bar.Baz\n")); got != "" {
		t.Fatalf("unexpected unsupported reason: %q", got)
	}
}

func TestRunRoundTripCheckSimpleRepo(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "App.java")
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(javaPath, []byte(`package com.example;

import java.util.List;

public class App {
}
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"roundtrip-check",
		"--repo", root,
		"--workers", "1",
	})
	if exitCode != 0 {
		t.Fatalf("run(roundtrip-check) = %d, want 0; stdout=%q stderr=%q", exitCode, out, stderr)
	}
	for _, want := range []string{
		"Files:",
		"total=1 checked=1",
		"RoundTrip:",
		"pass=1 diff=0 parse-failed=0 unsupported=0 format-error=0",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("roundtrip summary missing %q in %q", want, out)
		}
	}
}

func TestWriteRoundTripSummaryOmitsPassEntries(t *testing.T) {
	var out bytes.Buffer
	writeRoundTripSummary(&out, roundTripSummary{
		Root:            "/repo",
		TotalFiles:      2,
		CheckedFiles:    1,
		PassedFiles:     1,
		Unsupported:     1,
		JavaFormatCmd:   "google-java-format --replace {file}",
		KotlinFormatCmd: "ktfmt {file}",
		Files: []roundTripFileResult{
			{Relative: "src/Main.java", Status: roundTripStatusPass},
			{Relative: "src/Util.kt", Status: roundTripStatusUnsupported, Reason: "formatter command is required for reliable rewritten-source comparison"},
		},
	})

	text := out.String()
	if strings.Contains(text, "src/Main.java") {
		t.Fatalf("pass entry should not be listed: %q", text)
	}
	if !strings.Contains(text, "src/Util.kt") {
		t.Fatalf("unsupported entry missing: %q", text)
	}
	if !strings.Contains(text, "Formatter:") {
		t.Fatalf("formatter summary missing: %q", text)
	}
}

func TestCommandSourceFormatterPreservesExplicitValues(t *testing.T) {
	formatter := newCommandSourceFormatter("custom-java {file}", "custom-kotlin {file}")
	summary := formatter.Summary()
	if summary.JavaCommand != "custom-java {file}" {
		t.Fatalf("JavaCommand = %q", summary.JavaCommand)
	}
	if summary.KotlinCommand != "custom-kotlin {file}" {
		t.Fatalf("KotlinCommand = %q", summary.KotlinCommand)
	}
}

type recordingFormatter struct {
	count int
}

func (f *recordingFormatter) HasFormatter(lang mixedgraph.SourceLanguage) bool {
	return lang == mixedgraph.LangKotlin || lang == mixedgraph.LangJava
}

func (f *recordingFormatter) Format(lang mixedgraph.SourceLanguage, path string, source []byte) ([]byte, error) {
	f.count++
	return append([]byte(nil), source...), nil
}

func (f *recordingFormatter) Summary() formatterSummary {
	return formatterSummary{JavaCommand: "record-java {file}", KotlinCommand: "record-kotlin {file}"}
}

func TestRoundTripCheckFileLosslessModeUsesFormatterEvenWhenHeaderWouldNotChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Sample.kt")
	source := `package sample.demo

import kotlin.collections.List

class App
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	formatter := &recordingFormatter{}
	result, err := roundTripCheckFile(mixedgraph.FileUnit{
		Path:        path,
		Relative:    "Sample.kt",
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.demo",
		Imports:     []string{"kotlin.collections.List"},
		Parsed:      true,
	}, roundTripCheckConfig{
		formatter:   formatter,
		rewriteMode: roundTripRewriteModeLossless,
	})
	if err != nil {
		t.Fatalf("roundTripCheckFile(...) = %v", err)
	}
	if result.Status != roundTripStatusPass {
		t.Fatalf("status = %s, want %s", result.Status, roundTripStatusPass)
	}
	if formatter.count != 2 {
		t.Fatalf("formatter calls = %d, want 2", formatter.count)
	}
}

func TestRoundTripCheckFileWithoutFormatterPassesForChangedHeaderOnlyRewrite(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Sample.kt")
	source := `package sample.demo
import kotlin.collections.List

class App
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	result, err := roundTripCheckFile(mixedgraph.FileUnit{
		Path:        path,
		Relative:    "Sample.kt",
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.demo",
		Imports:     []string{"kotlin.collections.List"},
		Parsed:      true,
	}, roundTripCheckConfig{})
	if err != nil {
		t.Fatalf("roundTripCheckFile(...) = %v", err)
	}
	if result.Status != roundTripStatusPass {
		t.Fatalf("status = %s, want %s", result.Status, roundTripStatusPass)
	}
}

func TestRoundTripCheckFileWithoutFormatterReturnsUnsupportedWhenHeaderMeaningChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Sample.kt")
	source := `package sample.demo
import kotlin.collections.List

class App
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	result, err := roundTripCheckFile(mixedgraph.FileUnit{
		Path:        path,
		Relative:    "Sample.kt",
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.other",
		Imports:     []string{"kotlin.collections.List"},
		Parsed:      true,
	}, roundTripCheckConfig{})
	if err != nil {
		t.Fatalf("roundTripCheckFile(...) = %v", err)
	}
	if result.Status != roundTripStatusUnsupported {
		t.Fatalf("status = %s, want %s", result.Status, roundTripStatusUnsupported)
	}
	if !strings.Contains(result.Reason, "formatter command") {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}

func TestSemanticHeaderRewriteRejectsBodyChange(t *testing.T) {
	file := mixedgraph.FileUnit{
		Language:    mixedgraph.LangJava,
		PackageName: "sample",
		Imports:     []string{"java.util.List"},
	}
	original := []byte("package sample;\n\nimport java.util.List;\n\nclass App {}\n")
	rewritten := []byte("package sample;\n\nimport java.util.List;\n\nclass App { int x; }\n")
	if isSemanticallyEquivalentHeaderRewrite(original, rewritten, file) {
		t.Fatal("expected body change to be rejected")
	}
}

func TestBuildCanonicalHeaderSortsImports(t *testing.T) {
	header := buildCanonicalHeader(mixedgraph.LangJava, "sample", []string{
		"java.util.function.Function",
		"java.util.stream.Collectors",
		"java.util.List",
	})
	want := "package sample;\n\nimport java.util.List;\nimport java.util.function.Function;\nimport java.util.stream.Collectors;\n\n"
	if header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
}

func TestBuildRoundTripHeaderPreservesKotlinAliasImports(t *testing.T) {
	source := `package sample.demo

import sample.foo.Bar as BarAlias
import kotlin.collections.List

class App
`
	header := buildRoundTripHeader(mixedgraph.LangKotlin, "sample.demo", []string{"sample.foo.Bar", "kotlin.collections.List"}, source)
	if !strings.Contains(header, "import sample.foo.Bar as BarAlias") {
		t.Fatalf("alias import not preserved: %q", header)
	}
	if !strings.Contains(header, "import kotlin.collections.List") {
		t.Fatalf("regular import missing: %q", header)
	}
}

func TestBuildRoundTripHeaderPreservesJavaStaticImports(t *testing.T) {
	source := `package sample.demo;

import static java.util.Collections.emptyList;
import java.util.List;

class App {}
`
	header := buildRoundTripHeader(mixedgraph.LangJava, "sample.demo", []string{"java.util.List"}, source)
	if !strings.Contains(header, "import static java.util.Collections.emptyList;") {
		t.Fatalf("static import not preserved: %q", header)
	}
	if !strings.Contains(header, "import java.util.List;") {
		t.Fatalf("regular import missing: %q", header)
	}
}

func TestSemanticHeaderRewriteIgnoresPreservedJavaStaticImports(t *testing.T) {
	file := mixedgraph.FileUnit{
		Language:    mixedgraph.LangJava,
		PackageName: "sample.demo",
		Imports:     []string{"java.util.Collections.emptyList", "java.util.List"},
	}
	original := []byte(`package sample.demo;

import static java.util.Collections.emptyList;
import java.util.List;

class App {}
`)
	rewritten := []byte(`package sample.demo;

import static java.util.Collections.emptyList;
import java.util.List;

class App {}
`)
	if !isSemanticallyEquivalentHeaderRewrite(original, rewritten, file) {
		t.Fatal("expected preserved static imports to be ignored during semantic comparison")
	}
}

func TestSemanticHeaderRewriteIgnoresPreservedKotlinAliasImports(t *testing.T) {
	file := mixedgraph.FileUnit{
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.demo",
		Imports:     []string{"sample.foo.Bar", "kotlin.collections.List"},
	}
	original := []byte(`@file:JvmName("SampleKt")

package sample.demo

import sample.foo.Bar as BarAlias
import kotlin.collections.List

class App
`)
	rewritten := []byte(`@file:JvmName("SampleKt")

package sample.demo

import sample.foo.Bar as BarAlias
import kotlin.collections.List

class App
`)
	if !isSemanticallyEquivalentHeaderRewrite(original, rewritten, file) {
		t.Fatal("expected preserved alias imports to be ignored during semantic comparison")
	}
}

func TestSplitSourceSectionsKeepsKotlinFileAnnotationsInPreamble(t *testing.T) {
	source := `/* banner */
@file:Suppress("DEPRECATION_ERROR")

package sample.demo

import kotlin.collections.List

class App
`
	preamble, header, body, hadHeader := splitSourceSections(source)
	if !hadHeader {
		t.Fatal("expected header to be detected")
	}
	if !strings.Contains(preamble, "@file:Suppress") {
		t.Fatalf("expected file annotation in preamble: %q", preamble)
	}
	if !strings.Contains(header, "package sample.demo") {
		t.Fatalf("expected package in header: %q", header)
	}
	if !strings.HasPrefix(body, "class App") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestSplitSourceSectionsKeepsJavaPackageAnnotationsInPreamble(t *testing.T) {
	source := `/* banner */
@NullMarked
package sample.demo;

import org.jspecify.annotations.NullMarked;

interface Marker {}
`
	preamble, header, body, hadHeader := splitSourceSections(source)
	if !hadHeader {
		t.Fatal("expected header to be detected")
	}
	if !strings.Contains(preamble, "@NullMarked") {
		t.Fatalf("expected package annotation in preamble: %q", preamble)
	}
	if !strings.Contains(header, "package sample.demo;") {
		t.Fatalf("expected package in header: %q", header)
	}
	if !strings.Contains(header, "import org.jspecify.annotations.NullMarked;") {
		t.Fatalf("expected import in header: %q", header)
	}
	if !strings.HasPrefix(body, "interface Marker") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestRoundTripCheckFileWithoutFormatterPassesForKotlinAliasImports(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Alias.kt")
	source := `package sample.demo

import sample.foo.Bar as BarAlias
import kotlin.collections.List

class App {
  val values: List<String> = emptyList()
}
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	result, err := roundTripCheckFile(mixedgraph.FileUnit{
		Path:        path,
		Relative:    "Alias.kt",
		Language:    mixedgraph.LangKotlin,
		PackageName: "sample.demo",
		Imports:     []string{"sample.foo.Bar", "kotlin.collections.List"},
		Parsed:      true,
	}, roundTripCheckConfig{})
	if err != nil {
		t.Fatalf("roundTripCheckFile(...) = %v", err)
	}
	if result.Status != roundTripStatusPass {
		t.Fatalf("status = %s, want %s", result.Status, roundTripStatusPass)
	}
}
