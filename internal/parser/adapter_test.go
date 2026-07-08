package parser

import (
	"strings"
	"testing"
	"time"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// TestANTLRParserInterface verifies that ANTLRParser implements ast.Parser
func TestANTLRParserInterface(t *testing.T) {
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)

	// Verify it implements the interface
	var _ ast.Parser = parser

	if parser.Language() != ast.LanguageJava {
		t.Errorf("expected language Java, got %s", parser.Language())
	}
}

// TestANTLRParserParseSource tests parsing source code
func TestANTLRParserParseSource(t *testing.T) {
	tests := []struct {
		name        string
		language    ast.SourceLanguage
		grammar     JavaGrammar
		source      string
		wantSuccess bool
	}{
		{
			name:        "Java 8 lambda",
			language:    ast.LanguageJava,
			grammar:     JavaGrammar20,
			source:      "package test;\nimport java.util.function.Function;\npublic class Test { Function<String, String> f = s -> s.toUpperCase(); }",
			wantSuccess: true,
		},
		{
			name:        "Java 10 var",
			language:    ast.LanguageJava,
			grammar:     JavaGrammar20,
			source:      "package test;\npublic class Test { void test() { var x = 10; } }",
			wantSuccess: true,
		},
		{
			name:        "Kotlin simple",
			language:    ast.LanguageKotlin,
			grammar:     JavaGrammar20,
			source:      "package test\nfun main() = println(\"Hello\")",
			wantSuccess: true,
		},
		{
			name:        "Kotlin value class (normalized)",
			language:    ast.LanguageKotlin,
			grammar:     JavaGrammar20,
			source:      "@JvmInline\nvalue class UserId(val id: Long)",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewANTLRParser(tt.grammar, tt.language)
			opts := ast.ParseOptions{
				Language: tt.language,
				BuildAST: true,
				Lenient:  true, // Use lenient mode for testing
			}

			result, err := parser.ParseSource([]byte(tt.source), opts)
			if err != nil {
				t.Fatalf("ParseSource failed: %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ParseSource() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if opts.BuildAST && result.SourceFile == nil {
				t.Error("BuildAST=true but SourceFile is nil")
			}

			if result.Duration <= 0 {
				t.Error("Duration should be positive")
			}
		})
	}
}

// TestANTLRParserTimeout tests timeout functionality
func TestANTLRParserTimeout(t *testing.T) {
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)

	// Very short timeout - should still parse simple source quickly
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
		Timeout:  100 * time.Millisecond,
	}

	source := "package test;\npublic class Test {}"
	result, err := parser.ParseSource([]byte(source), opts)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}

	if !result.Success {
		t.Errorf("ParseSource() failed: %v", result.Diagnostics)
	}
}

// TestANTLRParserLenientMode tests lenient parsing mode
func TestANTLRParserLenientMode(t *testing.T) {
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)

	// Invalid Java source
	invalidSource := "package test;\npublic class Test { BROKEN SYNTAX HERE }"

	t.Run("Strict mode", func(t *testing.T) {
		opts := ast.ParseOptions{
			Language: ast.LanguageJava,
			Lenient:  false,
		}

		result, err := parser.ParseSource([]byte(invalidSource), opts)
		if err != nil {
			t.Fatalf("ParseSource failed: %v", err)
		}

		if result.Success {
			t.Error("Expected failure in strict mode, got success")
		}
	})

	t.Run("Lenient mode", func(t *testing.T) {
		opts := ast.ParseOptions{
			Language: ast.LanguageJava,
			Lenient:  true,
		}

		result, err := parser.ParseSource([]byte(invalidSource), opts)
		if err != nil {
			t.Fatalf("ParseSource failed: %v", err)
		}

		// In lenient mode, should still return success but with diagnostics
		if !result.Success {
			t.Error("Expected success in lenient mode")
		}

		if len(result.Diagnostics) == 0 {
			t.Error("Expected diagnostics in lenient mode for invalid source")
		}
	})
}

// TestANTLRParserBuildAST tests AST building
func TestANTLRParserBuildAST(t *testing.T) {
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)

	source := `package com.example;
import java.util.List;
import java.util.ArrayList;

public class TestClass {
	public static void main(String[] args) {
		List<String> list = new ArrayList<>();
	}
}`

	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
		BuildAST: true,
	}

	result, err := parser.ParseSource([]byte(source), opts)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("ParseSource failed: %v", result.Diagnostics)
	}

	if result.SourceFile == nil {
		t.Fatal("SourceFile is nil when BuildAST=true")
	}

	// Verify package
	if result.SourceFile.Package.Name != "com.example" {
		t.Errorf("Package name = %s, want com.example", result.SourceFile.Package.Name)
	}

	// Verify imports (should be sorted)
	if len(result.SourceFile.Imports) != 2 {
		t.Errorf("Import count = %d, want 2", len(result.SourceFile.Imports))
	} else {
		if result.SourceFile.Imports[0].Path != "java.util.ArrayList" {
			t.Errorf("First import = %s, want java.util.ArrayList", result.SourceFile.Imports[0].Path)
		}
		if result.SourceFile.Imports[1].Path != "java.util.List" {
			t.Errorf("Second import = %s, want java.util.List", result.SourceFile.Imports[1].Path)
		}
	}

	// Verify language
	if result.SourceFile.Language != ast.LanguageJava {
		t.Errorf("Language = %s, want java", result.SourceFile.Language)
	}
}

// TestParserFactory tests the parser factory
func TestParserFactory(t *testing.T) {
	factory := NewParserFactory(JavaGrammar20)

	t.Run("Supported languages", func(t *testing.T) {
		langs := factory.SupportedLanguages()
		if len(langs) != 2 {
			t.Errorf("Expected 2 supported languages, got %d", len(langs))
		}
	})

	t.Run("Create Java parser", func(t *testing.T) {
		parser, err := factory.CreateParser(ast.LanguageJava)
		if err != nil {
			t.Fatalf("CreateParser failed: %v", err)
		}

		if parser.Language() != ast.LanguageJava {
			t.Errorf("Parser language = %s, want java", parser.Language())
		}
	})

	t.Run("Create Kotlin parser", func(t *testing.T) {
		parser, err := factory.CreateParser(ast.LanguageKotlin)
		if err != nil {
			t.Fatalf("CreateParser failed: %v", err)
		}

		if parser.Language() != ast.LanguageKotlin {
			t.Errorf("Parser language = %s, want kotlin", parser.Language())
		}
	})

	t.Run("Create parser for unsupported language", func(t *testing.T) {
		_, err := factory.CreateParser(ast.SourceLanguage("unsupported"))
		if err == nil {
			t.Error("Expected error for unsupported language, got nil")
		}
	})
}

// TestANTLRParserParseReader tests parsing from io.Reader
func TestANTLRParserParseReader(t *testing.T) {
	tests := []struct {
		name        string
		language    ast.SourceLanguage
		grammar     JavaGrammar
		source      string
		wantSuccess bool
	}{
		{
			name:        "Java from reader",
			language:    ast.LanguageJava,
			grammar:     JavaGrammar20,
			source:      "package test;\npublic class Test { void foo() {} }",
			wantSuccess: true,
		},
		{
			name:        "Kotlin from reader",
			language:    ast.LanguageKotlin,
			grammar:     JavaGrammar20,
			source:      "package test\nfun foo() {}",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewANTLRParser(tt.grammar, tt.language)

			// Create a reader from the source string
			reader := strings.NewReader(tt.source)

			opts := ast.ParseOptions{
				Language: tt.language,
				Lenient:  true,
			}

			result, err := parser.ParseReader(reader, opts)
			if err != nil {
				t.Fatalf("ParseReader failed: %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ParseReader() success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

// TestASTBuilderStrategies tests language-specific AST builder strategies
func TestASTBuilderStrategies(t *testing.T) {
	t.Run("javaASTBuilder", func(t *testing.T) {
		builder := &javaASTBuilder{}
		source := []byte("package com.example;\nimport java.util.List;\nclass Test {}")

		result, err := builder.Build(source, ast.LanguageJava)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if result.Package.Name != "com.example" {
			t.Errorf("Package = %s, want com.example", result.Package.Name)
		}

		if len(result.Imports) != 1 {
			t.Errorf("Import count = %d, want 1", len(result.Imports))
		}

		if result.Language != ast.LanguageJava {
			t.Errorf("Language = %s, want java", result.Language)
		}
	})

	t.Run("kotlinASTBuilder", func(t *testing.T) {
		builder := &kotlinASTBuilder{}
		source := []byte("package com.example\nimport kotlin.collections.List\nclass Test")

		result, err := builder.Build(source, ast.LanguageKotlin)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if result.Package.Name != "com.example" {
			t.Errorf("Package = %s, want com.example", result.Package.Name)
		}

		if len(result.Imports) != 1 {
			t.Errorf("Import count = %d, want 1", len(result.Imports))
		}

		if result.Language != ast.LanguageKotlin {
			t.Errorf("Language = %s, want kotlin", result.Language)
		}
	})

	t.Run("defaultASTBuilder for unsupported language", func(t *testing.T) {
		builder := &defaultASTBuilder{}
		source := []byte("some source code")

		result, err := builder.Build(source, ast.SourceLanguage("python"))
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if result.Package.Name != "" {
			t.Errorf("Package should be empty, got %s", result.Package.Name)
		}

		if len(result.Imports) != 0 {
			t.Errorf("Import count = %d, want 0", len(result.Imports))
		}

		if result.Language != ast.SourceLanguage("python") {
			t.Errorf("Language = %s, want python", result.Language)
		}
	})
}

// TestConvertImports tests the convertImports helper function
func TestConvertImports(t *testing.T) {
	t.Run("Nil input returns nil", func(t *testing.T) {
		result := convertImports(nil)
		if result != nil {
			t.Errorf("convertImports(nil) = %v, want nil", result)
		}
	})

	t.Run("Empty input returns nil", func(t *testing.T) {
		result := convertImports([]string{})
		if result != nil {
			t.Errorf("convertImports([]) = %v, want nil", result)
		}
	})

	t.Run("Converts import paths", func(t *testing.T) {
		input := []string{"java.util.List", "java.util.Map", "kotlin.collections.List"}
		result := convertImports(input)

		if len(result) != 3 {
			t.Fatalf("Length = %d, want 3", len(result))
		}

		if result[0].Path != "java.util.List" {
			t.Errorf("First import = %s, want java.util.List", result[0].Path)
		}

		if result[1].Path != "java.util.Map" {
			t.Errorf("Second import = %s, want java.util.Map", result[1].Path)
		}

		if result[2].Path != "kotlin.collections.List" {
			t.Errorf("Third import = %s, want kotlin.collections.List", result[2].Path)
		}
	})
}
