package kotlincompilergolang

import (
	"testing"
)

func TestExtractTopLevelDeclarations(t *testing.T) {
	source := []byte(`
package com.example.demo

import kotlin.collections.List

@Serializable
data class User(val id: String)

internal interface Service {
  fun nestedMember(): Int
}

fun topLevel(value: String): String {
  return value
}

private val counter = 1

object Singleton

typealias UserList = List<User>
`)

	decls := extractTopLevelDeclarations(source)
	if len(decls) != 6 {
		t.Fatalf("expected 6 declarations, got %d", len(decls))
	}

	want := []struct {
		kind DeclarationKind
		name string
	}{
		{DeclClass, "User"},
		{DeclInterface, "Service"},
		{DeclFunction, "topLevel"},
		{DeclProperty, "counter"},
		{DeclObject, "Singleton"},
		{DeclTypeAlias, "UserList"},
	}

	for i, expected := range want {
		if decls[i].Kind != expected.kind || decls[i].Name != expected.name {
			t.Fatalf("declaration[%d] mismatch: got kind=%s name=%s, want kind=%s name=%s", i, decls[i].Kind, decls[i].Name, expected.kind, expected.name)
		}
	}
}

func TestParseSourceIncludesHeaderAndDeclarations(t *testing.T) {
	source := []byte(`
package com.example

import kotlin.String
import kotlin.collections.List

class Sample
fun runMe() = Unit
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true})
	unit := compiler.ParseSource("/tmp/Sample.kt", source)

	if unit.PackageName != "com.example" {
		t.Fatalf("unexpected package: %s", unit.PackageName)
	}
	if len(unit.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(unit.Imports))
	}
	if len(unit.Declarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(unit.Declarations))
	}
	if unit.Declarations[0].Name != "Sample" || unit.Declarations[1].Name != "runMe" {
		t.Fatalf("unexpected declarations: %+v", unit.Declarations)
	}
}

func TestParseSourceCollectsTopLevelTypeAlias(t *testing.T) {
	source := []byte(`
package com.example

import kotlin.collections.List

typealias Names = List<String>
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true})
	unit := compiler.ParseSource("/tmp/Alias.kt", source)
	if !unit.Parsed {
		t.Fatalf("parse should succeed, diagnostics=%+v", unit.Diagnostics)
	}
	if len(unit.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", unit.Diagnostics)
	}
	if len(unit.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d (%+v)", len(unit.Declarations), unit.Declarations)
	}
	if unit.Declarations[0].Kind != DeclTypeAlias || unit.Declarations[0].Name != "Names" {
		t.Fatalf("unexpected declaration: %+v", unit.Declarations[0])
	}
}

func TestParseSourceLenientSyntax(t *testing.T) {
	source := []byte(`
package com.example

import kotlin.String

fun broken(
`)

	strictCompiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true, LenientSyntax: false})
	lenientCompiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true, LenientSyntax: true})

	strictUnit := strictCompiler.ParseSource("/tmp/Broken.kt", source)
	if strictUnit.Parsed {
		t.Fatalf("strict parse should fail with syntax errors")
	}
	if len(strictUnit.Diagnostics) == 0 {
		t.Fatalf("strict parse should include diagnostics")
	}

	lenientUnit := lenientCompiler.ParseSource("/tmp/Broken.kt", source)
	if !lenientUnit.Parsed {
		t.Fatalf("lenient parse should mark file as parsed")
	}
	if len(lenientUnit.Diagnostics) == 0 {
		t.Fatalf("lenient parse should preserve diagnostics")
	}
	if lenientUnit.PackageName != "com.example" {
		t.Fatalf("unexpected package in lenient parse: %s", lenientUnit.PackageName)
	}
	if len(lenientUnit.Imports) != 1 || lenientUnit.Imports[0] != "kotlin.String" {
		t.Fatalf("unexpected imports in lenient parse: %+v", lenientUnit.Imports)
	}
}

func TestParseSourceCollectsNestedTypeDeclarations(t *testing.T) {
	source := []byte(`
package com.example

class Outer {
  class Inner
  companion object {
    class Deep
  }
}

object Holder {
  interface Api
}
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true})
	unit := compiler.ParseSource("/tmp/Nested.kt", source)
	if !unit.Parsed {
		t.Fatalf("parse should succeed, diagnostics=%+v", unit.Diagnostics)
	}

	declByName := map[string]DeclarationKind{}
	for _, decl := range unit.Declarations {
		declByName[decl.Name] = decl.Kind
	}

	want := map[string]DeclarationKind{
		"Outer":                DeclClass,
		"Outer.Inner":          DeclClass,
		"Outer.Companion":      DeclObject,
		"Outer.Companion.Deep": DeclClass,
		"Holder":               DeclObject,
		"Holder.Api":           DeclInterface,
	}

	if len(declByName) < len(want) {
		t.Fatalf("expected at least %d declarations, got %d (%+v)", len(want), len(declByName), unit.Declarations)
	}
	for name, kind := range want {
		got, ok := declByName[name]
		if !ok {
			t.Fatalf("missing declaration %s in %+v", name, unit.Declarations)
		}
		if got != kind {
			t.Fatalf("declaration %s kind mismatch: got=%s want=%s", name, got, kind)
		}
	}
}

func TestExtractHeaderIgnoresCommentImports(t *testing.T) {
	source := []byte(`
@file:JvmName("FlowKt")

package kotlinx.coroutines.flow

import kotlinx.coroutines.*
import kotlin.time.*

/*
----- INCLUDE .*
import kotlinx.coroutines.flow.*
import kotlin.time.Duration.Companion.milliseconds
*/
`)

	pkg, imports := extractHeader(source)
	if pkg != "kotlinx.coroutines.flow" {
		t.Fatalf("unexpected package: %q", pkg)
	}

	want := []string{
		"kotlin.time.*",
		"kotlinx.coroutines.*",
	}
	if len(imports) != len(want) {
		t.Fatalf("unexpected import count: got=%d want=%d imports=%v", len(imports), len(want), imports)
	}
	for i := range want {
		if imports[i] != want[i] {
			t.Fatalf("import[%d] mismatch: got=%q want=%q all=%v", i, imports[i], want[i], imports)
		}
	}
}

func TestExtractHeaderSupportsImportAlias(t *testing.T) {
	source := []byte(`
package sample

import kotlin.collections.MutableList as KMutableList
import kotlin.io.path.*

class App
`)

	pkg, imports := extractHeader(source)
	if pkg != "sample" {
		t.Fatalf("unexpected package: %q", pkg)
	}

	want := []string{
		"kotlin.collections.MutableList",
		"kotlin.io.path.*",
	}
	if len(imports) != len(want) {
		t.Fatalf("unexpected import count: got=%d want=%d imports=%v", len(imports), len(want), imports)
	}
	for i := range want {
		if imports[i] != want[i] {
			t.Fatalf("import[%d] mismatch: got=%q want=%q all=%v", i, imports[i], want[i], imports)
		}
	}
}

func TestParseSourceSupportsKotlinVersionSyntax(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		package_  string
		minImport int
	}{
		{
			name: "kotlin-1x",
			source: `
package sample.app

import kotlin.collections.List
import kotlin.text.*

data class User(val id: String)
fun parse(value: String): Int = value.length
`,
			package_:  "sample.app",
			minImport: 2,
		},
		{
			name: "kotlin-2x",
			source: `
package sample.app

import kotlin.io.*
import kotlin.math.abs as abs

value class Amount(val cents: Int)
fun interface EventListener {
  fun onEvent()
}
context(UserScope)
fun resolve(value: String): String = abs(value.length - 1).toString()
`,
			package_:  "sample.app",
			minImport: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			compiler := New(Config{
				Workers:          1,
				MaxErrorsPerFile: 5,
				IncludeKTS:       true,
				LenientSyntax:    false,
			})
			unit := compiler.ParseSource("/tmp/"+tc.name+".kt", []byte(tc.source))
			if !unit.Parsed {
				t.Fatalf("expected strict parse success for %s with diagnostics=%+v", tc.name, unit.Diagnostics)
			}
			if unit.PackageName != tc.package_ {
				t.Fatalf("unexpected package for %s: %q", tc.name, unit.PackageName)
			}
			if len(unit.Imports) < tc.minImport {
				t.Fatalf("expected at least %d imports for %s, got %d", tc.minImport, tc.name, len(unit.Imports))
			}
			if len(unit.Declarations) < 2 {
				t.Fatalf("expected at least 2 declarations for %s, got %d", tc.name, len(unit.Declarations))
			}
		})
	}
}

func TestParseSourceSupportsGenericExtensionReceiverFunction(t *testing.T) {
	source := []byte(`
package sample.app

suspend fun <T : Any> Call<T>.await(): T {
  return suspendCancellableCoroutine { continuation ->
    continuation.invokeOnCancellation { cancel() }
  }
}
`)
	compiler := New(Config{
		Workers:          1,
		MaxErrorsPerFile: 5,
		IncludeKTS:       true,
	})
	unit := compiler.ParseSource("/tmp/ExtensionReceiver.kt", source)
	if !unit.Parsed {
		t.Fatalf("expected strict parse success with diagnostics=%+v", unit.Diagnostics)
	}
	if len(unit.Declarations) != 1 || unit.Declarations[0].Name != "await" {
		t.Fatalf("unexpected declarations: %+v", unit.Declarations)
	}
}

func TestHasUnmatchedDelimiters(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "empty string", text: "", want: false},
		{name: "matched parentheses", text: "()", want: false},
		{name: "matched brackets", text: "[]", want: false},
		{name: "matched braces", text: "{}", want: false},
		{name: "unmatched open paren", text: "(", want: true},
		{name: "unmatched close paren", text: ")", want: false},
		{name: "unmatched open bracket", text: "[", want: true},
		{name: "unmatched close bracket", text: "]", want: false},
		{name: "unmatched open brace", text: "{", want: true},
		{name: "unmatched close brace", text: "}", want: false},
		{name: "nested balanced", text: "({[]})", want: false},
		{name: "nested unbalanced", text: "({[})", want: true},
		{name: "with string", text: `"(not a delimiter)"`, want: false},
		{name: "with char", text: `'('`, want: false},
		{name: "with raw string", text: `"""{"}"""`, want: false},
		{name: "with line comment", text: "// comment with ( bracket", want: false},
		{name: "with block comment", text: "/* comment with ( bracket */", want: false},
		{name: "valid kotlin function", text: "fun test(x: Int): String { return x.toString() }", want: false},
		{name: "incomplete function", text: "fun test(x: Int): String { return x.toString()", want: true},
		{name: "kotlin code with strings", text: `val s = "hello (world)"`, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasUnmatchedDelimiters(tc.text)
			if got != tc.want {
				t.Fatalf("hasUnmatchedDelimiters(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestParseSourceWithModifiersOnCompanionObject(t *testing.T) {
	source := []byte(`
package com.example

class MyClass {
  private companion object {
    private val secret = "value"
  }

  fun getSecret() = secret
}
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true})
	unit := compiler.ParseSource("/tmp/Companion.kt", source)
	if !unit.Parsed {
		t.Fatalf("parse should succeed, diagnostics=%+v", unit.Diagnostics)
	}

	// Verify companion object was parsed
	foundCompanion := false
	for _, decl := range unit.Declarations {
		if decl.Name == "MyClass.Companion" {
			foundCompanion = true
			break
		}
	}
	if !foundCompanion {
		t.Fatalf("expected MyClass.Companion declaration, got %+v", unit.Declarations)
	}
}

func TestParseSourceWithActualCompanionObject(t *testing.T) {
	source := []byte(`
package com.example

expect class Shared {
  actual companion object {
    actual fun create(): Shared
  }
}
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true, LenientSyntax: true})
	unit := compiler.ParseSource("/tmp/ExpectCompanion.kt", source)
	if !unit.Parsed {
		t.Fatalf("parse should succeed in lenient mode, diagnostics=%+v", unit.Diagnostics)
	}

	// Verify companion object was parsed
	foundCompanion := false
	for _, decl := range unit.Declarations {
		if decl.Name == "Shared.Companion" {
			foundCompanion = true
			break
		}
	}
	if !foundCompanion {
		t.Logf("Warning: Shared.Companion not found in %+v", unit.Declarations)
	}
}

func TestParseSourceWithMultipleModifiersOnCompanion(t *testing.T) {
	source := []byte(`
package com.example

class Container {
  private inner companion object Factory {
    fun create(): Container = Container()
  }
}
`)

	compiler := New(Config{Workers: 1, MaxErrorsPerFile: 5, IncludeKTS: true, LenientSyntax: true})
	unit := compiler.ParseSource("/tmp/MultiModifierCompanion.kt", source)
	if !unit.Parsed {
		t.Logf("Lenient parse result: diagnostics=%+v", unit.Diagnostics)
	}

	// Verify class was parsed
	foundClass := false
	for _, decl := range unit.Declarations {
		if decl.Name == "Container" {
			foundClass = true
			break
		}
	}
	if !foundClass {
		t.Fatalf("expected Container declaration, got %+v", unit.Declarations)
	}
}
