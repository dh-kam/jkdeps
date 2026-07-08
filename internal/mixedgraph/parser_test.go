package mixedgraph

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dh-kam/jkdeps/internal/parser"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestParseRepositoryMixedHeaders(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "App.java")
	kotlinPath := filepath.Join(root, "src", "main", "kotlin", "com", "example", "Util.kt")

	mustWrite(t, javaPath, `
package com.example;

import java.util.List;
import static java.util.Collections.emptyList;

class App {}
`)
	mustWrite(t, kotlinPath, `
package com.example

import kotlin.collections.List
import kotlin.js.Promise

class Util
`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          2,
		IncludeKTS:       false,
		MaxErrorsPerFile: 10,
		LenientSyntax:    false,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 2 || result.JavaFiles != 1 || result.KotlinFiles != 1 {
		t.Fatalf("unexpected file counts: total=%d java=%d kotlin=%d", result.TotalFiles, result.JavaFiles, result.KotlinFiles)
	}
	if result.ParsedFiles != 2 || result.FailedFiles != 0 {
		t.Fatalf("unexpected parse counts: parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	}

	byPath := map[string]FileUnit{}
	for _, file := range result.Files {
		byPath[file.Path] = file
	}

	javaUnit, ok := byPath[javaPath]
	if !ok {
		t.Fatalf("missing java file unit")
	}
	if javaUnit.PackageName != "com.example" {
		t.Fatalf("unexpected java package: %s", javaUnit.PackageName)
	}
	if len(javaUnit.Imports) != 2 {
		t.Fatalf("expected 2 java imports, got %d", len(javaUnit.Imports))
	}

	kotlinUnit, ok := byPath[kotlinPath]
	if !ok {
		t.Fatalf("missing kotlin file unit")
	}
	if kotlinUnit.PackageName != "com.example" {
		t.Fatalf("unexpected kotlin package: %s", kotlinUnit.PackageName)
	}
	if len(kotlinUnit.Imports) != 2 {
		t.Fatalf("expected 2 kotlin imports, got %d", len(kotlinUnit.Imports))
	}
}

func TestParseRepositoryLenientJava(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "broken", "Broken.java")
	mustWrite(t, javaPath, `
package com.example;
import java.util.List;
class Broken {
`)

	strict, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		MaxErrorsPerFile: 10,
		LenientSyntax:    false,
	})
	if err != nil {
		t.Fatalf("strict ParseRepository returned error: %v", err)
	}
	if strict.ParsedFiles != 0 || strict.FailedFiles != 1 {
		t.Fatalf("unexpected strict parse counts: parsed=%d failed=%d", strict.ParsedFiles, strict.FailedFiles)
	}
	if len(strict.Files) != 1 || strict.Files[0].Parsed {
		t.Fatalf("strict parse should mark file as failed")
	}
	if strict.Files[0].PackageName != "com.example" {
		t.Fatalf("strict parse should still extract package, got %s", strict.Files[0].PackageName)
	}
	if len(strict.Files[0].Diagnostics) == 0 {
		t.Fatalf("strict parse should include diagnostics")
	}

	lenient, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		MaxErrorsPerFile: 10,
		LenientSyntax:    true,
	})
	if err != nil {
		t.Fatalf("lenient ParseRepository returned error: %v", err)
	}
	if lenient.ParsedFiles != 1 || lenient.FailedFiles != 0 {
		t.Fatalf("unexpected lenient parse counts: parsed=%d failed=%d", lenient.ParsedFiles, lenient.FailedFiles)
	}
	if len(lenient.Files) != 1 || !lenient.Files[0].Parsed {
		t.Fatalf("lenient parse should mark file as parsed")
	}
	if len(lenient.Files[0].Diagnostics) == 0 {
		t.Fatalf("lenient parse should keep diagnostics")
	}
}

func TestParseRepositoryHeaderOnlyJavaSkipsSyntaxValidation(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "broken", "Broken.java")
	mustWrite(t, javaPath, `
package com.example;
import java.util.List;
class Broken {
`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		JavaParseMode:    JavaParseModeHeaderOnly,
		Workers:          1,
		MaxErrorsPerFile: 10,
	})
	if err != nil {
		t.Fatalf("header-only ParseRepository returned error: %v", err)
	}
	if result.ParsedFiles != 1 || result.FailedFiles != 0 {
		t.Fatalf("unexpected header-only parse counts: parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected one parsed file, got %d", len(result.Files))
	}
	if !result.Files[0].Parsed {
		t.Fatalf("header-only parse should mark file as parsed")
	}
	if result.Files[0].PackageName != "com.example" {
		t.Fatalf("header-only parse should still extract package, got %s", result.Files[0].PackageName)
	}
	if len(result.Files[0].Imports) != 1 || result.Files[0].Imports[0] != "java.util.List" {
		t.Fatalf("header-only parse imports = %+v, want [java.util.List]", result.Files[0].Imports)
	}
	if len(result.Files[0].Diagnostics) != 0 {
		t.Fatalf("header-only parse diagnostics = %+v, want none", result.Files[0].Diagnostics)
	}
	if len(result.Files[0].References) != 0 {
		t.Fatalf("header-only parse references = %+v, want none", result.Files[0].References)
	}
}

func TestParseRepositoryFullJavaExtractsSignatureReferences(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "Sample.java")
	mustWrite(t, javaPath, `
package com.example;

import java.util.List;
import java.io.IOException;

class Sample extends BaseType implements Runnable {
  private List<String> names;

  Result process(Input input) throws IOException {
    return null;
  }
}
`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:   parser.JavaGrammar20,
		JavaParseMode: JavaParseModeFull,
		Workers:       1,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("len(result.Files) = %d, want 1", len(result.Files))
	}

	got := map[ReferenceKind]map[string]struct{}{}
	for _, ref := range result.Files[0].References {
		if got[ref.Kind] == nil {
			got[ref.Kind] = map[string]struct{}{}
		}
		got[ref.Kind][ref.Path] = struct{}{}
	}
	if _, ok := got[ReferenceKindExtends]["BaseType"]; !ok {
		t.Fatalf("missing extends BaseType in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindImplements]["Runnable"]; !ok {
		t.Fatalf("missing implements Runnable in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindFieldType]["List"]; !ok {
		t.Fatalf("missing field_type List in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindTypeArgument]["String"]; !ok {
		t.Fatalf("missing type_argument String in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindMethodReturn]["Result"]; !ok {
		t.Fatalf("missing method_return Result in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindMethodParameter]["Input"]; !ok {
		t.Fatalf("missing method_parameter Input in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindThrows]["IOException"]; !ok {
		t.Fatalf("missing throws IOException in references: %+v", result.Files[0].References)
	}
}

func TestParseRepositoryFullJavaExtractsBodyReferences(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "Sample.java")
	mustWrite(t, javaPath, `
package com.example;

import java.util.Collections;

class Sample {
  void run() {
    new Created();
    new java.util.ArrayList<String>();
    java.util.function.Supplier<java.util.ArrayList<String>> creator = java.util.ArrayList::new;
    java.util.function.Supplier<java.util.List<String>> empty = java.util.Collections::emptyList;
    java.util.Map<String, Created> local = null;
    Collections.emptyList();
    java.util.Objects.requireNonNull(this);
    Class<?> clazz = java.lang.String.class;
    Object obj = local;
    java.util.List<String> casted = (java.util.List<String>) obj;
    try {
      Collections.emptyList();
    } catch (java.io.IOException | CustomError ex) {
    }
    if (obj instanceof java.util.Map<String, ?> map) {
    }
  }
}
`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:   parser.JavaGrammar20,
		JavaParseMode: JavaParseModeFull,
		Workers:       1,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("len(result.Files) = %d, want 1", len(result.Files))
	}

	got := map[ReferenceKind]map[string]struct{}{}
	for _, ref := range result.Files[0].References {
		if got[ref.Kind] == nil {
			got[ref.Kind] = map[string]struct{}{}
		}
		got[ref.Kind][ref.Path] = struct{}{}
	}
	if _, ok := got[ReferenceKindConstructorCall]["Created"]; !ok {
		t.Fatalf("missing constructor_call Created in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindConstructorCall]["java.util.ArrayList"]; !ok {
		t.Fatalf("missing constructor_call java.util.ArrayList in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindLocalVariableType]["java.util.Map"]; !ok {
		t.Fatalf("missing local_variable_type java.util.Map in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindQualifiedMethodCall]["Collections"]; !ok {
		t.Fatalf("missing qualified_method_call Collections in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindQualifiedMethodCall]["java.util.Objects"]; !ok {
		t.Fatalf("missing qualified_method_call java.util.Objects in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindClassLiteral]["java.lang.String"]; !ok {
		t.Fatalf("missing class_literal java.lang.String in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindMethodReference]["java.util.Collections"]; !ok {
		t.Fatalf("missing method_reference java.util.Collections in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindConstructorReference]["java.util.ArrayList"]; !ok {
		t.Fatalf("missing constructor_reference java.util.ArrayList in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindCastType]["java.util.List"]; !ok {
		t.Fatalf("missing cast_type java.util.List in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindInstanceofType]["java.util.Map"]; !ok {
		t.Fatalf("missing instanceof_type java.util.Map in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindCatchType]["java.io.IOException"]; !ok {
		t.Fatalf("missing catch_type java.io.IOException in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindCatchType]["CustomError"]; !ok {
		t.Fatalf("missing catch_type CustomError in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindTypeArgument]["String"]; !ok {
		t.Fatalf("missing type_argument String in references: %+v", result.Files[0].References)
	}
	if _, ok := got[ReferenceKindTypeArgument]["Created"]; !ok {
		t.Fatalf("missing type_argument Created in references: %+v", result.Files[0].References)
	}
}

func TestSyntaxErrorListenerRespectsMaxErrorsPerFile(t *testing.T) {
	listener := newSyntaxErrorListener("sample/Bad.java", 2)
	listener.addMessage(1, 1, "first issue")
	listener.addMessage(2, 2, "second issue")
	listener.addMessage(3, 3, "third issue")

	diagnostics := listener.Diagnostics()
	if len(diagnostics) != 2 {
		t.Fatalf("unexpected diagnostics count: got=%d want=2", len(diagnostics))
	}
	if diagnostics[0].Message != "first issue" || diagnostics[1].Message != "second issue" {
		t.Fatalf("unexpected diagnostic messages: %+v", diagnostics)
	}
}

func TestParseRepositoryRespectsMaxErrorsPerFile(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "src", "main", "java", "com", "example", "Broken.java")
	mustWrite(t, javaPath, `
package com.example;

public class Broken {
  int x = ;
  int y = ;
  int z = ;
}
`)

	strict, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		MaxErrorsPerFile: 1,
		LenientSyntax:    false,
	})
	if err != nil {
		t.Fatalf("strict ParseRepository returned error: %v", err)
	}
	if len(strict.Files) != 1 {
		t.Fatalf("expected one parsed file, got %d", len(strict.Files))
	}
	if strict.ParsedFiles != 0 || strict.FailedFiles != 1 {
		t.Fatalf("unexpected strict parse counts: parsed=%d failed=%d", strict.ParsedFiles, strict.FailedFiles)
	}
	if len(strict.Files[0].Diagnostics) == 0 {
		t.Fatalf("expected diagnostics with strict parsing")
	}
	if len(strict.Files[0].Diagnostics) > 1 {
		t.Fatalf("max errors per file not enforced: got %d", len(strict.Files[0].Diagnostics))
	}

	lenient, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		MaxErrorsPerFile: 1,
		LenientSyntax:    true,
	})
	if err != nil {
		t.Fatalf("lenient ParseRepository returned error: %v", err)
	}
	if lenient.ParsedFiles != 1 || lenient.FailedFiles != 0 {
		t.Fatalf("unexpected lenient parse counts: parsed=%d failed=%d", lenient.ParsedFiles, lenient.FailedFiles)
	}
	if len(lenient.Files[0].Diagnostics) == 0 {
		t.Fatalf("expected diagnostics preserved in lenient mode")
	}
	if len(lenient.Files[0].Diagnostics) > 1 {
		t.Fatalf("max errors per file not enforced in lenient mode: got %d", len(lenient.Files[0].Diagnostics))
	}
}

func TestParseRepositoryExcludeBuildScriptsByDefault(t *testing.T) {
	root := t.TempDir()
	ktPath := filepath.Join(root, "src", "main", "kotlin", "com", "example", "App.kt")
	ktsPath := filepath.Join(root, "scripts", "tool.kts")
	buildGradlePath := filepath.Join(root, "build.gradle.kts")
	settingsGradlePath := filepath.Join(root, "settings.gradle.kts")

	mustWrite(t, ktPath, "package com.example\nclass App\n")
	mustWrite(t, ktsPath, "val tool = 1\n")
	mustWrite(t, buildGradlePath, "plugins {}\n")
	mustWrite(t, settingsGradlePath, "rootProject.name = \"sample\"\n")

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		IncludeKTS:       true,
		MaxErrorsPerFile: 10,
		LenientSyntax:    true,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 2 || result.KotlinFiles != 2 {
		t.Fatalf("unexpected file counts: total=%d kotlin=%d", result.TotalFiles, result.KotlinFiles)
	}

	got := map[string]struct{}{}
	for _, file := range result.Files {
		got[filepath.Base(file.Path)] = struct{}{}
	}
	if _, ok := got["App.kt"]; !ok {
		t.Fatalf("expected App.kt in parsed files")
	}
	if _, ok := got["tool.kts"]; !ok {
		t.Fatalf("expected tool.kts in parsed files")
	}
	if _, ok := got["build.gradle.kts"]; ok {
		t.Fatalf("did not expect build.gradle.kts when IncludeBuildScripts=false")
	}
	if _, ok := got["settings.gradle.kts"]; ok {
		t.Fatalf("did not expect settings.gradle.kts when IncludeBuildScripts=false")
	}
}

func TestParseRepositoryIncludeBuildScripts(t *testing.T) {
	root := t.TempDir()
	ktPath := filepath.Join(root, "src", "main", "kotlin", "com", "example", "App.kt")
	ktsPath := filepath.Join(root, "scripts", "tool.kts")
	buildGradlePath := filepath.Join(root, "build.gradle.kts")
	settingsGradlePath := filepath.Join(root, "settings.gradle.kts")

	mustWrite(t, ktPath, "package com.example\nclass App\n")
	mustWrite(t, ktsPath, "val tool = 1\n")
	mustWrite(t, buildGradlePath, "plugins {}\n")
	mustWrite(t, settingsGradlePath, "rootProject.name = \"sample\"\n")

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:         parser.JavaGrammar20,
		Workers:             1,
		IncludeKTS:          true,
		IncludeBuildScripts: true,
		MaxErrorsPerFile:    10,
		LenientSyntax:       true,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 4 || result.KotlinFiles != 4 {
		t.Fatalf("unexpected file counts: total=%d kotlin=%d", result.TotalFiles, result.KotlinFiles)
	}

	got := map[string]struct{}{}
	for _, file := range result.Files {
		got[filepath.Base(file.Path)] = struct{}{}
	}
	if _, ok := got["build.gradle.kts"]; !ok {
		t.Fatalf("expected build.gradle.kts when IncludeBuildScripts=true")
	}
	if _, ok := got["settings.gradle.kts"]; !ok {
		t.Fatalf("expected settings.gradle.kts when IncludeBuildScripts=true")
	}
}

func TestParseJavaDiagnosticsWithTimeoutReturnsTimeoutDiagnostic(t *testing.T) {
	parseFunc := func(_ string, _ string, _ int, _ parser.JavaGrammar) []Diagnostic {
		time.Sleep(20 * time.Millisecond)
		return []Diagnostic{{Path: "slow.java", Message: "slow"}}
	}

	got := parseJavaDiagnosticsWithTimeoutFunc(parseFunc, "class A {}", "slow.java", 10, parser.JavaGrammar20, time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("parseJavaDiagnosticsWithTimeout() diagnostics = %d, want 1", len(got))
	}
	if got[0].Path != "slow.java" {
		t.Fatalf("parseJavaDiagnosticsWithTimeout().Path = %q, want %q", got[0].Path, "slow.java")
	}
	if got[0].Message != "parse timeout after 1ms" {
		t.Fatalf("parseJavaDiagnosticsWithTimeout().Message = %q, want %q", got[0].Message, "parse timeout after 1ms")
	}
}

func TestParseJavaDiagnosticsWithTimeoutReturnsParserDiagnostics(t *testing.T) {
	want := []Diagnostic{{Path: "Fast.java", Line: 3, Column: 4, Message: "boom"}}
	parseFunc := func(_ string, _ string, _ int, _ parser.JavaGrammar) []Diagnostic {
		return want
	}

	got := parseJavaDiagnosticsWithTimeoutFunc(parseFunc, "class A {}", "Fast.java", 10, parser.JavaGrammar20, 50*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("parseJavaDiagnosticsWithTimeout() diagnostics = %d, want 1", len(got))
	}
	if got[0] != want[0] {
		t.Fatalf("parseJavaDiagnosticsWithTimeout() = %+v, want %+v", got[0], want[0])
	}
}

func TestParseJavaDiagnosticsWithTimeoutWrapper(t *testing.T) {
	source := `package sample;

class Broken {
`

	got := parseJavaDiagnosticsWithTimeout(source, "Broken.java", 10, parser.JavaGrammar20, 100*time.Millisecond)
	if len(got) == 0 {
		t.Fatal("parseJavaDiagnosticsWithTimeout() = no diagnostics, want parse errors")
	}
	if got[0].Path != "Broken.java" {
		t.Fatalf("parseJavaDiagnosticsWithTimeout().Path = %q, want %q", got[0].Path, "Broken.java")
	}
}

func TestConvertKotlinDiagnostics(t *testing.T) {
	input := []kcg.Diagnostic{
		{Path: "A.kt", Line: 1, Column: 2, Message: "first"},
		{Path: "B.kt", Line: 3, Column: 4, Message: "second"},
	}

	got := convertKotlinDiagnostics(input)
	if len(got) != len(input) {
		t.Fatalf("convertKotlinDiagnostics() len = %d, want %d", len(got), len(input))
	}
	for i, item := range input {
		if got[i].Path != item.Path || got[i].Line != item.Line || got[i].Column != item.Column || got[i].Message != item.Message {
			t.Fatalf("convertKotlinDiagnostics()[%d] = %+v, want %+v", i, got[i], item)
		}
	}
	if convertKotlinDiagnostics(nil) != nil {
		t.Fatal("convertKotlinDiagnostics(nil) should return nil")
	}
}

func TestParseRepositoryIncludeBuildScriptsWithoutIncludingKts(t *testing.T) {
	root := t.TempDir()
	ktsPath := filepath.Join(root, "scripts", "tool.kts")
	buildGradlePath := filepath.Join(root, "build.gradle.kts")
	settingsGradlePath := filepath.Join(root, "settings.gradle.kts")

	mustWrite(t, ktsPath, "val tool = 1\n")
	mustWrite(t, buildGradlePath, "plugins {}\n")
	mustWrite(t, settingsGradlePath, "rootProject.name = \"sample\"\n")

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:         parser.JavaGrammar20,
		Workers:             1,
		IncludeKTS:          false,
		IncludeBuildScripts: true,
		MaxErrorsPerFile:    10,
		LenientSyntax:       true,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 2 || result.KotlinFiles != 2 {
		t.Fatalf("unexpected file counts: total=%d kotlin=%d", result.TotalFiles, result.KotlinFiles)
	}

	got := map[string]struct{}{}
	for _, file := range result.Files {
		got[filepath.Base(file.Path)] = struct{}{}
	}
	if _, ok := got["build.gradle.kts"]; !ok {
		t.Fatalf("expected build.gradle.kts when IncludeBuildScripts=true and IncludeKTS=false")
	}
	if _, ok := got["settings.gradle.kts"]; !ok {
		t.Fatalf("expected settings.gradle.kts when IncludeBuildScripts=true and IncludeKTS=false")
	}
	if _, ok := got["tool.kts"]; ok {
		t.Fatalf("did not expect ordinary .kts file when IncludeKTS=false")
	}
}

func TestParseRepositoryDefaultsWorkersAndLimits(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")

	mustWrite(t, javaPath, `package sample;

public class App {}`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          0,
		MaxErrorsPerFile: 0,
		LenientSyntax:    true,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 1 || result.ParsedFiles != 1 || result.FailedFiles != 0 || result.JavaFiles != 1 {
		t.Fatalf("unexpected parse summary: total=%d parsed=%d failed=%d java=%d", result.TotalFiles, result.ParsedFiles, result.FailedFiles, result.JavaFiles)
	}
}

func TestParseRepositoryRejectsUnsupportedJavaGrammar(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "App.java"), "public class App {}")

	_, err := ParseRepository(root, ParseOptions{
		JavaGrammar: "java99",
	})
	if err == nil {
		t.Fatalf("expected unsupported java grammar error")
	}
}

func TestParseRepositoryHeaderOnlyIgnoresUnsupportedJavaGrammar(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "App.java"), "package sample;\nimport java.util.List;\npublic class App {}")

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:   "java99",
		JavaParseMode: JavaParseModeHeaderOnly,
	})
	if err != nil {
		t.Fatalf("header-only ParseRepository returned error: %v", err)
	}
	if result.ParsedFiles != 1 || result.FailedFiles != 0 {
		t.Fatalf("unexpected header-only parse counts: parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	}
	if len(result.Files) != 1 || len(result.Files[0].Imports) != 1 {
		t.Fatalf("unexpected header-only parse result: %+v", result.Files)
	}
}

func TestParseJavaDiagnosticsSupportsRepresentativeGrammars(t *testing.T) {
	source := `package sample;

public class App {}
`

	grammars := []parser.JavaGrammar{
		parser.JavaGrammarOrig,
		parser.JavaGrammar8,
		parser.JavaGrammar9,
		parser.JavaGrammar20,
		parser.JavaGrammar21,
	}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			got := parseJavaDiagnostics(source, "App.java", 10, grammar)
			if len(got) != 0 {
				t.Fatalf("parseJavaDiagnostics(%s) diagnostics = %d, want 0: %+v", grammar, len(got), got)
			}
		})
	}
}

func TestParseJavaDiagnosticsReportsUnsupportedGrammar(t *testing.T) {
	got := parseJavaDiagnostics("class App {}", "App.java", 10, parser.JavaGrammar("java99"))
	if len(got) != 1 {
		t.Fatalf("parseJavaDiagnostics() diagnostics = %d, want 1", len(got))
	}
	if got[0].Message != "unsupported java grammar: java99" {
		t.Fatalf("parseJavaDiagnostics() message = %q, want %q", got[0].Message, "unsupported java grammar: java99")
	}
}

func TestParseRepositorySkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	ignored := filepath.Join(root, "node_modules", "ignored", "Module.java")
	kept := filepath.Join(root, "src", "main", "java", "Kept.java")

	mustWrite(t, ignored, `package ignored;

public class Module {}`)
	mustWrite(t, kept, `package sample;

public class Kept {}`)

	result, err := ParseRepository(root, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		MaxErrorsPerFile: 10,
		LenientSyntax:    true,
	})
	if err != nil {
		t.Fatalf("ParseRepository returned error: %v", err)
	}
	if result.TotalFiles != 1 || result.JavaFiles != 1 {
		t.Fatalf("unexpected scan result: total=%d java=%d", result.TotalFiles, result.JavaFiles)
	}
	if result.Files[0].PackageName != "sample" {
		t.Fatalf("expected kept package only, got %q", result.Files[0].PackageName)
	}
}

func TestParseRepositoryReturnsErrorForMissingRepoRoot(t *testing.T) {
	_, err := ParseRepository(filepath.Join(t.TempDir(), "does-not-exist"), ParseOptions{
		JavaGrammar: parser.JavaGrammar20,
	})
	if err == nil {
		t.Fatalf("expected error for missing repo root")
	}
}

func TestKotlinCompilerConfig_PropagatesParseOptions(t *testing.T) {
	opts := ParseOptions{
		Workers:             8,
		IncludeKTS:          true,
		IncludeBuildScripts: true,
		MaxErrorsPerFile:    17,
		LenientSyntax:       true,
		ParseTimeout:        2500 * time.Millisecond,
	}

	cfg := kotlinCompilerConfig(opts)
	if cfg.Workers != 1 {
		t.Fatalf("expected fixed kotlin worker=1, got %d", cfg.Workers)
	}
	if !cfg.IncludeKTS {
		t.Fatalf("expected IncludeKTS=true")
	}
	if cfg.MaxErrorsPerFile != 17 {
		t.Fatalf("expected MaxErrorsPerFile=17, got %d", cfg.MaxErrorsPerFile)
	}
	if !cfg.LenientSyntax {
		t.Fatalf("expected LenientSyntax=true")
	}
	if cfg.ParseTimeout != 2500*time.Millisecond {
		t.Fatalf("expected ParseTimeout=2500ms, got %s", cfg.ParseTimeout)
	}
}

func mustWrite(t *testing.T, path, payload string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
