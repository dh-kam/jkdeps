package parser

import (
	"testing"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// TestParseWithDiagnostics tests ParseWithDiagnostics function
func TestParseWithDiagnostics(t *testing.T) {
	t.Run("Valid Java source", func(t *testing.T) {
		source := []byte("package com.example;\nimport java.util.List;\npublic class Test {}")

		result := ParseWithDiagnostics("Test.java", source, langJava, JavaGrammar20, 10)

		if !result.Parsed {
			t.Errorf("Expected parsed=true, got false. Diagnostics: %v", result.Diagnostics)
		}

		if result.PackageName != "com.example" {
			t.Errorf("Package = %s, want com.example", result.PackageName)
		}

		if len(result.Imports) != 1 {
			t.Errorf("Import count = %d, want 1", len(result.Imports))
		}

		if result.Language != ast.LanguageJava {
			t.Errorf("Language = %s, want java", result.Language)
		}

		if result.Duration <= 0 {
			t.Error("Duration should be positive")
		}
	})

	t.Run("Valid Kotlin source", func(t *testing.T) {
		source := []byte("package com.example\nimport kotlin.collections.List\nclass Test")

		result := ParseWithDiagnostics("Test.kt", source, langKotlin, JavaGrammar20, 10)

		if !result.Parsed {
			t.Errorf("Expected parsed=true, got false. Diagnostics: %v", result.Diagnostics)
		}

		if result.Language != ast.LanguageKotlin {
			t.Errorf("Language = %s, want kotlin", result.Language)
		}
	})

	t.Run("Invalid Java source produces diagnostics", func(t *testing.T) {
		source := []byte("package test;\nclass Test { BROKEN SYNTAX }")

		result := ParseWithDiagnostics("Test.java", source, langJava, JavaGrammar20, 5)

		if result.Parsed {
			t.Error("Expected parsed=false for invalid source")
		}

		if len(result.Diagnostics) == 0 {
			t.Error("Expected diagnostics for invalid source")
		}
	})

	t.Run("Java source with when identifier normalization", func(t *testing.T) {
		source := []byte(`package test;

import static org.mockito.Mockito.when;

class Test {
  void ok() {
    when(value()).thenReturn("x");
  }
}
`)

		result := ParseWithDiagnostics("Test.java", source, langJava, JavaGrammar20, 10)

		if !result.Parsed {
			t.Fatalf("Expected parsed=true after normalization. Diagnostics: %v", result.Diagnostics)
		}
		if len(result.Imports) != 1 || result.Imports[0] != "org.mockito.Mockito.when" {
			t.Fatalf("Imports = %#v, want original static when import", result.Imports)
		}
	})

	t.Run("Kotlin value class normalization", func(t *testing.T) {
		source := []byte("@JvmInline\nvalue class UserId(val id: Long)")

		result := ParseWithDiagnostics("UserId.kt", source, langKotlin, JavaGrammar20, 10)

		if !result.Parsed {
			t.Errorf("Expected parsed=true after normalization. Diagnostics: %v", result.Diagnostics)
		}
	})
}

// TestParseJavaWithListener tests parseJavaWithListener function
func TestParseJavaWithListener(t *testing.T) {
	source := []byte("package test;\npublic class Test {}")
	listener := newSyntaxErrorListener(5)

	t.Run("Java20 grammar", func(t *testing.T) {
		err := parseJavaWithListener(source, JavaGrammar20, listener)
		if err != nil {
			t.Errorf("parseJavaWithListener(Java20) failed: %v", err)
		}
	})

	t.Run("Java8 grammar", func(t *testing.T) {
		err := parseJavaWithListener(source, JavaGrammar8, listener)
		if err != nil {
			t.Errorf("parseJavaWithListener(Java8) failed: %v", err)
		}
	})

	t.Run("Java9 grammar", func(t *testing.T) {
		err := parseJavaWithListener(source, JavaGrammar9, listener)
		if err != nil {
			t.Errorf("parseJavaWithListener(Java9) failed: %v", err)
		}
	})

	t.Run("JavaOrig grammar", func(t *testing.T) {
		err := parseJavaWithListener(source, JavaGrammarOrig, listener)
		if err != nil {
			t.Errorf("parseJavaWithListener(JavaOrig) failed: %v", err)
		}
	})

	t.Run("Java7 grammar", func(t *testing.T) {
		err := parseJavaWithListener(source, JavaGrammar7, listener)
		if err != nil {
			t.Errorf("parseJavaWithListener(Java7) failed: %v", err)
		}
	})

	t.Run("Invalid source produces errors", func(t *testing.T) {
		invalidSource := []byte("package test;\nclass Test { BROKEN }")
		listener := newSyntaxErrorListener(5)

		err := parseJavaWithListener(invalidSource, JavaGrammar20, listener)
		if err == nil {
			t.Error("Expected error for invalid source")
		}
	})
}

// TestParseKotlinWithListener tests parseKotlinWithListener function
func TestParseKotlinWithListener(t *testing.T) {
	source := []byte("package test\nfun main() = println(\"Hello\")")
	listener := newSyntaxErrorListener(5)

	t.Run("Valid Kotlin source", func(t *testing.T) {
		err := parseKotlinWithListener(source, listener)
		if err != nil {
			t.Errorf("parseKotlinWithListener failed: %v", err)
		}
	})

	t.Run("Invalid Kotlin source", func(t *testing.T) {
		invalidSource := []byte("package test\nfun main { BROKEN SYNTAX }")
		listener := newSyntaxErrorListener(5)

		err := parseKotlinWithListener(invalidSource, listener)
		if err == nil {
			t.Error("Expected error for invalid Kotlin source")
		}
	})

	t.Run("Kotlin value class (normalized)", func(t *testing.T) {
		source := []byte("@JvmInline\nvalue class UserId(val id: Long)")
		listener := newSyntaxErrorListener(5)

		err := parseKotlinWithListener(source, listener)
		if err != nil {
			t.Errorf("parseKotlinWithListener failed for value class: %v", err)
		}
	})
}
