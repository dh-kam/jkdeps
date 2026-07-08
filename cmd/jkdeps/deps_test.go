package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

type dependencyRunResult struct {
	Root              string                       `json:"root"`
	JavaGrammar       string                       `json:"java_grammar"`
	JavaParseMode     string                       `json:"java_parse_mode"`
	TotalFiles        int                          `json:"total_files"`
	JavaFiles         int                          `json:"java_files"`
	KotlinFiles       int                          `json:"kotlin_files"`
	ParsedFiles       int                          `json:"parsed_files"`
	FailedFiles       int                          `json:"failed_files"`
	DependencyCount   int                          `json:"dependency_count"`
	UnresolvedCount   int                          `json:"unresolved_count"`
	FileDependencies  []mixedgraph.FileDependency  `json:"file_dependencies"`
	UnresolvedImports []mixedgraph.FileDependency  `json:"unresolved_imports"`
	SlowParseFiles    []mixedgraph.ParseFileTiming `json:"slow_parse_files"`
}

func writeDepsFixture(t *testing.T, root, relativePath, body string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture path: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestRunDepsBuildsExpectedDependencyMetrics(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "App.java")
	kotlinPath := filepath.Join(root, "src", "main", "kotlin", "com", "example", "internal", "Util.kt")
	outPath := filepath.Join(root, "deps.json")
	javaContent := `package com.example;

import java.util.List;
import com.example.internal.Util;
import com.unknown.Missing;

public class App {
}`
	kotlinContent := `package com.example.internal

import com.example.App
import com.abc.DoesNotExist
`

	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatalf("mkdir for java path: %v", err)
	}
	if err := os.WriteFile(javaPath, []byte(javaContent), 0o644); err != nil {
		t.Fatalf("write java fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(kotlinPath), 0o755); err != nil {
		t.Fatalf("mkdir for kotlin path: %v", err)
	}
	if err := os.WriteFile(kotlinPath, []byte(kotlinContent), 0o644); err != nil {
		t.Fatalf("write kotlin fixture: %v", err)
	}

	code := runDeps([]string{
		"--repo", root,
		"--java-grammar", "java20",
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--include-kts",
		"--lenient",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}

	if out.Root != root {
		t.Fatalf("root mismatch: %q", out.Root)
	}
	if out.JavaGrammar != "java20" {
		t.Fatalf("java grammar mismatch: %q", out.JavaGrammar)
	}
	if out.TotalFiles != 2 || out.JavaFiles != 1 || out.KotlinFiles != 1 {
		t.Fatalf("file counts mismatch: total=%d java=%d kotlin=%d", out.TotalFiles, out.JavaFiles, out.KotlinFiles)
	}
	if out.ParsedFiles != 2 || out.FailedFiles != 0 {
		t.Fatalf("parse counts mismatch: parsed=%d failed=%d", out.ParsedFiles, out.FailedFiles)
	}
	if out.DependencyCount != 5 {
		t.Fatalf("dependency count mismatch: got=%d", out.DependencyCount)
	}
	if out.UnresolvedCount != 2 {
		t.Fatalf("unresolved count mismatch: got=%d", out.UnresolvedCount)
	}
	if len(out.UnresolvedImports) != 2 {
		t.Fatalf("unresolved imports payload mismatch: got=%d", len(out.UnresolvedImports))
	}
	if len(out.FileDependencies) != out.DependencyCount {
		t.Fatalf("file dependency list mismatch: got=%d expected=%d", len(out.FileDependencies), out.DependencyCount)
	}
}

func TestRunDepsParsesRepresentativeJavaGrammars(t *testing.T) {
	cases := []struct {
		name    string
		grammar string
		file    string
		body    string
	}{
		{
			name:    "java7",
			grammar: "java7",
			file:    "Java7Feature.java",
			body: `
package sample;

public class Java7Feature {
  public int size() throws Exception {
    try (java.io.ByteArrayInputStream input = new java.io.ByteArrayInputStream(new byte[] {1, 2, 3})) {
      return input.available();
    }
  }
}`,
		},
		{
			name:    "java8",
			grammar: "java8",
			file:    "Java8Feature.java",
			body: `
package sample;

import java.util.function.Function;

public class Java8Feature {
  Function<String, Integer> toLen = s -> s.length();
}
`,
		},
		{
			name:    "java9",
			grammar: "java9",
			file:    "Java9Feature.java",
			body: `
package sample;

interface Java9Feature {
  private static void ping() {}
}
`,
		},
		{
			name:    "java11",
			grammar: "java11",
			file:    "Java11Feature.java",
			body: `
package sample;

public class Java11Feature {
  public String repeatValue(String value) {
    var repeated = value.repeat(2);
    return repeated;
  }
}
`,
		},
		{
			name:    "java17",
			grammar: "java17",
			file:    "Java17Feature.java",
			body: `
package sample;

public sealed interface Java17Feature permits Java17Impl {
}

final class Java17Impl implements Java17Feature {
}
`,
		},
		{
			name:    "java21",
			grammar: "java21",
			file:    "Java21Feature.java",
			body: `
package sample;

public record Java21Feature(String id, int count) {
}
`,
		},
		{
			name:    "java25",
			grammar: "java25",
			file:    "Java25Feature.java",
			body: `
package sample;

public class Java25Feature {
  public String status(Object input) {
    return switch (input) {
      case null -> "missing";
      default -> input.toString();
    };
  }
}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "src", "main", "java", tc.file)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir for java path: %v", err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write java fixture: %v", err)
			}

			outPath := filepath.Join(root, "deps.json")
			code := runDeps([]string{
				"--repo", root,
				"--java-grammar", tc.grammar,
				"--workers", "1",
				"--max-errors-per-file", "10",
				"--include-kts",
				"--lenient",
				"--json",
				"--out", outPath,
			})
			if code != 0 {
				t.Fatalf("runDeps returned code %d for grammar %s", code, tc.grammar)
			}

			payload, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			var out dependencyRunResult
			if err := json.Unmarshal(payload, &out); err != nil {
				t.Fatalf("unmarshal deps json: %v", err)
			}
			if out.JavaGrammar != tc.grammar {
				t.Fatalf("java grammar mismatch: got %q expected %q", out.JavaGrammar, tc.grammar)
			}
			if out.TotalFiles != 1 || out.ParsedFiles != 1 || out.FailedFiles != 0 {
				t.Fatalf("parse counts mismatch for %s: total=%d parsed=%d failed=%d", tc.grammar, out.TotalFiles, out.ParsedFiles, out.FailedFiles)
			}
		})
	}
}

func TestRunDepsCapturesExpectedKotlinVersionDependencies(t *testing.T) {
	cases := []struct {
		name                    string
		files                   map[string]string
		expectedDependencyCount int
		expectedInternal        int
		expectedExternal        int
		expectedUnknown         int
		expectedUnresolved      int
		expectUnresolvedImport  string
	}{
		{
			name: "kotlin-1x",
			files: map[string]string{
				"src/main/kotlin/sample/one/Feature.kt": `
package sample.one

import sample.two.Helper
import kotlin.collections.List
import com.unknown.Missing

class Feature {
  private val values: List<String> = listOf()
}

`,
				"src/main/kotlin/sample/two/Helper.kt": `
package sample.two

class Helper
`,
			},
			expectedDependencyCount: 3,
			expectedInternal:        1,
			expectedExternal:        1,
			expectedUnknown:         1,
			expectedUnresolved:      1,
			expectUnresolvedImport:  "com.unknown.Missing",
		},
		{
			name: "kotlin-2x",
			files: map[string]string{
				"src/main/kotlin/sample/one/Feature.kt": `
package sample.one

import sample.two.Helper
import kotlin.io.*
import kotlin.math.abs as abs
import com.unknown.Missing

value class Amount(val cents: Int)
fun interface EventListener {
  fun onEvent()
}
context(UserScope)
fun format(scope: UserScope, input: String): String {
  return abs(input.length - 1).toString()
}
class UserScope
`,
				"src/main/kotlin/sample/two/Helper.kt": `
package sample.two

class Helper
`,
			},
			expectedDependencyCount: 4,
			expectedInternal:        1,
			expectedExternal:        2,
			expectedUnknown:         1,
			expectedUnresolved:      1,
			expectUnresolvedImport:  "com.unknown.Missing",
		},
	}

	inventoryPath := filepath.Join(t.TempDir(), "external.json")
	inventory := []byte(`{
  "packages": [
    {"package": "kotlin.collections"},
    {"package": "kotlin.io"},
    {"package": "kotlin.math"}
  ]
}
`)
	if err := os.WriteFile(inventoryPath, inventory, 0o644); err != nil {
		t.Fatalf("write external index: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for relPath, content := range tc.files {
				path := filepath.Join(root, relPath)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir for kotlin fixture: %v", err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write kotlin fixture: %v", err)
				}
			}

			outPath := filepath.Join(root, "deps.json")
			code := runDeps([]string{
				"--repo", root,
				"--workers", "1",
				"--max-errors-per-file", "10",
				"--include-kts",
				"--lenient",
				"--json",
				"--inventory", inventoryPath,
				"--out", outPath,
			})
			if code != 0 {
				t.Fatalf("runDeps returned code %d", code)
			}

			payload, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			var out dependencyRunResult
			if err := json.Unmarshal(payload, &out); err != nil {
				t.Fatalf("unmarshal deps json: %v", err)
			}
			if out.TotalFiles != len(tc.files) || out.KotlinFiles != len(tc.files) {
				t.Fatalf("file counts mismatch: total=%d kotlin=%d", out.TotalFiles, out.KotlinFiles)
			}
			if out.ParsedFiles != len(tc.files) || out.FailedFiles != 0 {
				t.Fatalf("parse counts mismatch: parsed=%d failed=%d", out.ParsedFiles, out.FailedFiles)
			}
			if out.DependencyCount != tc.expectedDependencyCount {
				t.Fatalf("dependency count mismatch: got=%d want=%d", out.DependencyCount, tc.expectedDependencyCount)
			}
			if len(out.FileDependencies) != tc.expectedDependencyCount {
				t.Fatalf("file dependency list mismatch: got=%d want=%d", len(out.FileDependencies), tc.expectedDependencyCount)
			}
			if out.UnresolvedCount != tc.expectedUnresolved {
				t.Fatalf("unresolved count mismatch: got=%d want=%d", out.UnresolvedCount, tc.expectedUnresolved)
			}
			if len(out.UnresolvedImports) != tc.expectedUnresolved {
				t.Fatalf("unresolved import list mismatch: got=%d want=%d", len(out.UnresolvedImports), tc.expectedUnresolved)
			}

			kinds := map[mixedgraph.NodeKind]int{}
			hasExpectedUnresolved := false
			for _, dep := range out.FileDependencies {
				kinds[dep.Kind]++
				if dep.Kind == mixedgraph.NodeUnknown && dep.ImportPath == tc.expectUnresolvedImport {
					hasExpectedUnresolved = true
				}
			}
			if kinds[mixedgraph.NodeInternal] != tc.expectedInternal {
				t.Fatalf("internal dependency count mismatch: got=%d want=%d", kinds[mixedgraph.NodeInternal], tc.expectedInternal)
			}
			if kinds[mixedgraph.NodeExternal] != tc.expectedExternal {
				t.Fatalf("external dependency count mismatch: got=%d want=%d", kinds[mixedgraph.NodeExternal], tc.expectedExternal)
			}
			if kinds[mixedgraph.NodeUnknown] != tc.expectedUnknown {
				t.Fatalf("unknown dependency count mismatch: got=%d want=%d", kinds[mixedgraph.NodeUnknown], tc.expectedUnknown)
			}
			if tc.expectedUnknown > 0 && !hasExpectedUnresolved {
				t.Fatalf("expected unresolved import %q not found", tc.expectUnresolvedImport)
			}
		})
	}
}

func TestRunDepsBuildsMixedLanguageDependencyKinds(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "sample", "App.java")
	kotlinPath := filepath.Join(root, "src", "main", "kotlin", "com", "sample", "kotlin", "Util.kt")
	inventoryPath := filepath.Join(root, "external.json")
	outPath := filepath.Join(root, "deps.json")

	javaContent := `
package com.sample;

import com.sample.kotlin.Util;
import java.util.List;

public class App {
}
`
	kotlinContent := `
package com.sample.kotlin

import com.sample.App
import kotlin.collections.List

class Util
`
	inventory := []byte(`{
  "packages": [
    {"package": "java.util"},
    {"package": "kotlin.collections"}
  ]
}
`)
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatalf("mkdir java fixture path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(kotlinPath), 0o755); err != nil {
		t.Fatalf("mkdir kotlin fixture path: %v", err)
	}
	if err := os.WriteFile(javaPath, []byte(javaContent), 0o644); err != nil {
		t.Fatalf("write java fixture: %v", err)
	}
	if err := os.WriteFile(kotlinPath, []byte(kotlinContent), 0o644); err != nil {
		t.Fatalf("write kotlin fixture: %v", err)
	}
	if err := os.WriteFile(inventoryPath, inventory, 0o644); err != nil {
		t.Fatalf("write external inventory: %v", err)
	}

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--include-kts",
		"--lenient",
		"--json",
		"--inventory", inventoryPath,
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if out.TotalFiles != 2 || out.ParsedFiles != 2 || out.FailedFiles != 0 || out.DependencyCount != 4 {
		t.Fatalf("unexpected parse/dependency totals: total=%d parsed=%d failed=%d deps=%d", out.TotalFiles, out.ParsedFiles, out.FailedFiles, out.DependencyCount)
	}

	kinds := map[mixedgraph.NodeKind]int{}
	for _, dep := range out.FileDependencies {
		kinds[dep.Kind]++
	}
	if kinds[mixedgraph.NodeInternal] != 2 {
		t.Fatalf("internal dependency mismatch: got=%d want=%d", kinds[mixedgraph.NodeInternal], 2)
	}
	if kinds[mixedgraph.NodeExternal] != 2 {
		t.Fatalf("external dependency mismatch: got=%d want=%d", kinds[mixedgraph.NodeExternal], 2)
	}
	if kinds[mixedgraph.NodeUnknown] != 0 {
		t.Fatalf("unexpected unknown dependency count: %d", kinds[mixedgraph.NodeUnknown])
	}
	if out.UnresolvedCount != 0 {
		t.Fatalf("unexpected unresolved count: %d", out.UnresolvedCount)
	}
}

func TestRunDepsRejectsInvalidGroupBy(t *testing.T) {
	root := t.TempDir()
	outPath := filepath.Join(root, "deps.json")

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--include-kts",
		"--lenient",
		"--group-by", "package-or-dir",
		"--json",
		"--out", outPath,
	})
	if code != 2 {
		t.Fatalf("runDeps returned code %d, want 2", code)
	}

	if _, err := os.Stat(outPath); err == nil {
		t.Fatalf("expected no output file for invalid --group-by, got: %s", outPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected os.Stat error for out path: %v", err)
	}
}

func TestRunDepsRejectsInvalidJavaGrammar(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}
	outPath := filepath.Join(root, "deps.json")

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--java-parse-mode", "full",
		"--java-grammar", "java99",
		"--json",
		"--out", outPath,
	})
	if code != 1 {
		t.Fatalf("runDeps returned code %d, want 1", code)
	}
	if _, err := os.Stat(outPath); err == nil {
		t.Fatalf("expected no output file for invalid java grammar, got: %s", outPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected os.Stat error for out path: %v", err)
	}
}

func TestRunDepsDefaultHeaderOnlyIgnoresInvalidJavaGrammar(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "App.java", "package sample;\nimport java.util.List;\npublic class App {}")

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--java-grammar", "java99",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps default header-only with invalid grammar) = %d, want 0", exitCode)
	}
	var parsed dependencyRunResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if parsed.JavaParseMode != "header-only" {
		t.Fatalf("parsed.JavaParseMode = %q, want header-only", parsed.JavaParseMode)
	}
	if parsed.ParsedFiles != 1 || parsed.FailedFiles != 0 {
		t.Fatalf("unexpected parse counts: parsed=%d failed=%d", parsed.ParsedFiles, parsed.FailedFiles)
	}
}

func TestRunDepsRejectsInvalidInventoryPath(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}
	outPath := filepath.Join(root, "deps.json")

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--json",
		"--inventory", filepath.Join(root, "missing", "inventory.json"),
		"--out", outPath,
	})
	if code != 1 {
		t.Fatalf("runDeps returned code %d, want 1", code)
	}
	if _, err := os.Stat(outPath); err == nil {
		t.Fatalf("expected no output file for invalid inventory path, got: %s", outPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected os.Stat error for out path: %v", err)
	}
}

func TestRunDepsRejectsInvalidInventoryJSON(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}
	inventoryPath := filepath.Join(root, "inventory.json")
	if err := os.WriteFile(inventoryPath, []byte(`{"packages":[`), 0o644); err != nil {
		t.Fatalf("write broken inventory: %v", err)
	}
	outPath := filepath.Join(root, "deps.json")

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--json",
		"--inventory", inventoryPath,
		"--out", outPath,
	})
	if code != 1 {
		t.Fatalf("runDeps returned code %d, want 1", code)
	}
	if _, err := os.Stat(outPath); err == nil {
		t.Fatalf("expected no output file for invalid inventory json, got: %s", outPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected os.Stat error for out path: %v", err)
	}
}

func TestRunDepsRespectFailOnErrorWhenParseFails(t *testing.T) {
	root := t.TempDir()
	inaccessiblePath := filepath.Join(root, "no_read.java")
	if err := os.WriteFile(inaccessiblePath, []byte("public class Broken {}"), 0o000); err != nil {
		t.Fatalf("write inaccessible java: %v", err)
	}
	if err := os.Chmod(inaccessiblePath, 0o000); err != nil {
		t.Fatalf("chmod inaccessible java: %v", err)
	}

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--fail-on-error",
		"--json",
	})
	if code != 1 {
		t.Fatalf("runDeps returned code %d, want 1", code)
	}
}

func TestRunDepsRespectFailOnErrorForSyntaxFailuresWithJSON(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "Invalid.java")
	if err := os.WriteFile(javaPath, []byte("public class Broken { int x = ; }"), 0o644); err != nil {
		t.Fatalf("write invalid java: %v", err)
	}

	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--java-parse-mode", "full",
		"--fail-on-error",
		"--json",
	})
	if code != 1 {
		t.Fatalf("runDeps returned code %d, want 1", code)
	}
}

func TestRunDepsParsesKotlinScriptsByDefault(t *testing.T) {
	root := t.TempDir()
	ktsPath := filepath.Join(root, "tool.kts")
	if err := os.WriteFile(ktsPath, []byte(`package sample

class Script`), 0o644); err != nil {
		t.Fatalf("write kts file: %v", err)
	}

	outPath := filepath.Join(root, "deps.json")
	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--json",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d, want 0", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if out.TotalFiles != 1 || out.KotlinFiles != 1 || out.ParsedFiles != 1 || out.FailedFiles != 0 {
		t.Fatalf("unexpected counts: total=%d kotlin=%d parsed=%d failed=%d", out.TotalFiles, out.KotlinFiles, out.ParsedFiles, out.FailedFiles)
	}
}

func TestRunDepsCanExcludeKotlinScripts(t *testing.T) {
	root := t.TempDir()
	ktsPath := filepath.Join(root, "tool.kts")
	if err := os.WriteFile(ktsPath, []byte(`package sample

class Script`), 0o644); err != nil {
		t.Fatalf("write kts file: %v", err)
	}

	outPath := filepath.Join(root, "deps.json")
	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--include-kts=false",
		"--json",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d, want 0", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if out.TotalFiles != 0 || out.KotlinFiles != 0 || out.ParsedFiles != 0 {
		t.Fatalf("unexpected counts: total=%d kotlin=%d parsed=%d", out.TotalFiles, out.KotlinFiles, out.ParsedFiles)
	}
}

func TestRunDepsCanIncludeBuildScriptsWhenEnabled(t *testing.T) {
	root := t.TempDir()
	buildGradle := filepath.Join(root, "build.gradle.kts")
	if err := os.WriteFile(buildGradle, []byte(`plugins {}`), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}

	outPath := filepath.Join(root, "deps.json")
	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--include-kts",
		"--include-build-scripts",
		"--json",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d, want 0", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if out.TotalFiles != 1 || out.KotlinFiles != 1 || out.ParsedFiles != 1 {
		t.Fatalf("unexpected counts: total=%d kotlin=%d parsed=%d", out.TotalFiles, out.KotlinFiles, out.ParsedFiles)
	}
}

func TestRunDepsCanIncludeBuildScriptsWithoutKts(t *testing.T) {
	root := t.TempDir()
	buildGradle := filepath.Join(root, "build.gradle.kts")
	settingsGradle := filepath.Join(root, "settings.gradle.kts")
	ktsPath := filepath.Join(root, "tool.kts")

	if err := os.WriteFile(buildGradle, []byte(`plugins {}`), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}
	if err := os.WriteFile(settingsGradle, []byte(`rootProject.name = "sample"`), 0o644); err != nil {
		t.Fatalf("write settings script: %v", err)
	}
	if err := os.WriteFile(ktsPath, []byte(`package sample`), 0o644); err != nil {
		t.Fatalf("write generic kts script: %v", err)
	}

	outPath := filepath.Join(root, "deps.json")
	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--include-kts=false",
		"--include-build-scripts",
		"--json",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d, want 0", code)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out dependencyRunResult
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if out.TotalFiles != 2 || out.KotlinFiles != 2 || out.ParsedFiles != 2 {
		t.Fatalf("unexpected counts: total=%d kotlin=%d parsed=%d", out.TotalFiles, out.KotlinFiles, out.ParsedFiles)
	}
}

func TestRunDepsRespectsLenientSyntax(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "src/main/java/Invalid.java", `package sample;

public class Invalid { int x = ; }`)

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--java-parse-mode", "full",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps strict) = %d, want 0", exitCode)
	}
	var strictOut dependencyRunResult
	if err := json.Unmarshal([]byte(out), &strictOut); err != nil {
		t.Fatalf("unmarshal strict deps json: %v", err)
	}
	if strictOut.ParsedFiles != 0 || strictOut.FailedFiles != 1 {
		t.Fatalf("unexpected strict parse counts: parsed=%d failed=%d", strictOut.ParsedFiles, strictOut.FailedFiles)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--java-parse-mode", "full",
		"--lenient",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps lenient) = %d, want 0", exitCode)
	}
	var lenientOut dependencyRunResult
	if err := json.Unmarshal([]byte(out), &lenientOut); err != nil {
		t.Fatalf("unmarshal lenient deps json: %v", err)
	}
	if lenientOut.ParsedFiles != 1 || lenientOut.FailedFiles != 0 {
		t.Fatalf("unexpected lenient parse counts: parsed=%d failed=%d", lenientOut.ParsedFiles, lenientOut.FailedFiles)
	}
}

func TestRunDepsDefaultsToHeaderOnlyJava(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "src/main/java/Invalid.java", `package sample;

import java.util.List;

public class Invalid { int x = ; }`)

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps default header-only) = %d, want 0", exitCode)
	}
	var parsed dependencyRunResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal header-only deps json: %v", err)
	}
	if parsed.JavaParseMode != "header-only" {
		t.Fatalf("parsed.JavaParseMode = %q, want header-only", parsed.JavaParseMode)
	}
	if parsed.ParsedFiles != 1 || parsed.FailedFiles != 0 {
		t.Fatalf("unexpected default header-only parse counts: parsed=%d failed=%d", parsed.ParsedFiles, parsed.FailedFiles)
	}
}

func TestRunDepsCanIncludeSlowParseFilesInJSON(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "src/main/java/One.java", "package sample;\nclass One {}")
	writeDepsFixture(t, root, "src/main/java/Two.java", "package sample;\nclass Two { int x = ; }")

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--lenient",
		"--top-parse-files", "1",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps with slow parse files) = %d, want 0", exitCode)
	}

	var parsed dependencyRunResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if len(parsed.SlowParseFiles) != 1 {
		t.Fatalf("len(SlowParseFiles) = %d, want 1", len(parsed.SlowParseFiles))
	}
	if parsed.SlowParseFiles[0].Path == "" {
		t.Fatalf("SlowParseFiles[0].Path is empty")
	}
	if parsed.SlowParseFiles[0].Duration <= 0 {
		t.Fatalf("SlowParseFiles[0].Duration = %s, want > 0", parsed.SlowParseFiles[0].Duration)
	}
}

func TestRunDepsHeaderOnlyJavaPreservesDependencyReport(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "src/main/java/com/example/App.java", `package com.example;

import com.example.internal.Util;
import java.util.List;
import com.external.Missing;

public class App {}`)
	writeDepsFixture(t, root, "src/main/java/com/example/internal/Util.java", `package com.example.internal;

import java.time.Instant;

public class Util {}`)

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps full) = %d, want 0", exitCode)
	}
	var full dependencyRunResult
	if err := json.Unmarshal([]byte(out), &full); err != nil {
		t.Fatalf("unmarshal full deps json: %v", err)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--java-parse-mode", "header-only",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(deps header-only) = %d, want 0", exitCode)
	}
	var headerOnly dependencyRunResult
	if err := json.Unmarshal([]byte(out), &headerOnly); err != nil {
		t.Fatalf("unmarshal header-only deps json: %v", err)
	}

	if headerOnly.JavaParseMode != "header-only" {
		t.Fatalf("headerOnly.JavaParseMode = %q, want header-only", headerOnly.JavaParseMode)
	}
	if headerOnly.DependencyCount != full.DependencyCount {
		t.Fatalf("dependency count mismatch: full=%d header-only=%d", full.DependencyCount, headerOnly.DependencyCount)
	}
	if headerOnly.UnresolvedCount != full.UnresolvedCount {
		t.Fatalf("unresolved count mismatch: full=%d header-only=%d", full.UnresolvedCount, headerOnly.UnresolvedCount)
	}
	if len(headerOnly.FileDependencies) != len(full.FileDependencies) {
		t.Fatalf("file dependency length mismatch: full=%d header-only=%d", len(full.FileDependencies), len(headerOnly.FileDependencies))
	}
	if !reflect.DeepEqual(headerOnly.FileDependencies, full.FileDependencies) {
		t.Fatalf("file dependencies mismatch:\nfull=%+v\nheader-only=%+v", full.FileDependencies, headerOnly.FileDependencies)
	}
	if len(headerOnly.UnresolvedImports) != len(full.UnresolvedImports) {
		t.Fatalf("unresolved import length mismatch: full=%d header-only=%d", len(full.UnresolvedImports), len(headerOnly.UnresolvedImports))
	}
	if !reflect.DeepEqual(headerOnly.UnresolvedImports, full.UnresolvedImports) {
		t.Fatalf("unresolved imports mismatch:\nfull=%+v\nheader-only=%+v", full.UnresolvedImports, headerOnly.UnresolvedImports)
	}
	if headerOnly.ParsedFiles != 2 || headerOnly.FailedFiles != 0 {
		t.Fatalf("unexpected header-only parse counts: parsed=%d failed=%d", headerOnly.ParsedFiles, headerOnly.FailedFiles)
	}
}

func TestRunDepsJSONPrintsToStdoutAndWritesFileWhenOutSet(t *testing.T) {
	root := t.TempDir()
	writeDepsFixture(t, root, "src/main/java/Sample.java", "package sample;\n\npublic class Sample {}")
	outPath := filepath.Join(root, "deps.json")

	exitCode, stdout, _ := runWithCapturedStdout(t, []string{
		"deps",
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--json",
		"--out", outPath,
	})
	if exitCode != 0 {
		t.Fatalf("run(deps --json --out) = %d, want 0", exitCode)
	}
	var fromStdout dependencyRunResult
	if err := json.Unmarshal([]byte(stdout), &fromStdout); err != nil {
		t.Fatalf("unmarshal stdout json: %v output=%q", err, stdout)
	}
	if fromStdout.TotalFiles != 1 || fromStdout.KotlinFiles != 0 || fromStdout.ParsedFiles != 1 {
		t.Fatalf("unexpected stdout result: %+v", fromStdout)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var fromFile dependencyRunResult
	if err := json.Unmarshal(payload, &fromFile); err != nil {
		t.Fatalf("unmarshal file json: %v", err)
	}
	if fromFile.TotalFiles != fromStdout.TotalFiles {
		t.Fatalf("stdout/json mismatch total: %d != %d", fromStdout.TotalFiles, fromFile.TotalFiles)
	}
}

func TestRunDepsCreatesOutPathParentWhenMissing(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}

	outPath := filepath.Join(root, "missing", "dir", "deps.json")
	code := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--json",
		"--out", outPath,
	})
	if code != 0 {
		t.Fatalf("runDeps returned code %d, want 0", code)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file for created parent path, got err: %v", err)
	}
}

func TestRunDepsRejectsInvalidFileTimeout(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}

	exitCode := runDeps([]string{
		"--repo", root,
		"--workers", "1",
		"--max-errors-per-file", "10",
		"--lenient",
		"--file-timeout", "invalid-duration",
		"--json",
	})
	if exitCode != 2 {
		t.Fatalf("runDeps returned code %d, want 2", exitCode)
	}
}

func TestUniqueStringsDeduplicatesTrimmedValues(t *testing.T) {
	got := flagutil.UniqueStrings([]string{"", "a", " b ", "a", "b", "", "c", " c", "b "})
	if len(got) != 3 {
		t.Fatalf("unexpected uniqueStrings length: got=%d want=3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected uniqueStrings output order/values: %+v", got)
	}
}

func TestGraphOutputPaths(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantHTML string
		wantJSON string
	}{
		{
			name:     "empty",
			in:       "",
			wantHTML: "jkdeps-mixed-graph.html",
			wantJSON: "jkdeps-mixed-graph.json",
		},
		{
			name:     "base",
			in:       "custom-report",
			wantHTML: "custom-report.html",
			wantJSON: "custom-report.json",
		},
		{
			name:     "html_override",
			in:       "path/to/graph.html",
			wantHTML: "path/to/graph.html",
			wantJSON: "path/to/graph.json",
		},
		{
			name:     "json_override",
			in:       "path/to/graph.json",
			wantHTML: "path/to/graph.html",
			wantJSON: "path/to/graph.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html, jsonPath := cliutil.GraphOutputPaths(tc.in, "jkdeps-mixed-graph")
			if html != tc.wantHTML || jsonPath != tc.wantJSON {
				t.Fatalf("graphOutputPaths(%q) = (%q, %q), want (%q, %q)", tc.in, html, jsonPath, tc.wantHTML, tc.wantJSON)
			}
		})
	}
}

func TestRunDepsHandlesInvalidGroupBy(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write java file: %v", err)
	}

	exitCode := runDeps([]string{
		"--repo", root,
		"--group-by", "invalid",
	})
	if exitCode != 2 {
		t.Fatalf("runDeps with invalid group-by returned code %d, want 2", exitCode)
	}
}
