package parser

import (
	"os"
	"path/filepath"
	"testing"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// BenchmarkJava8Parsing benchmarks Java 8 parsing with different grammars
func BenchmarkJava8Parsing(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java8Lambda.java")

	grammars := []JavaGrammar{JavaGrammar8, JavaGrammar9, JavaGrammar20}

	for _, grammar := range grammars {
		b.Run(string(grammar), func(b *testing.B) {
			parser := NewANTLRParser(grammar, ast.LanguageJava)
			opts := ast.ParseOptions{
				Language: ast.LanguageJava,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := parser.ParseSource(source, opts)
				if err != nil {
					b.Fatalf("ParseSource failed: %v", err)
				}
				if !result.Success {
					b.Errorf("Parse failed: %v", result.Diagnostics)
				}
			}
		})
	}
}

// BenchmarkJava10Var benchmarks Java 10 var keyword parsing
func BenchmarkJava10Var(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java10Var.java")

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}
}

// BenchmarkJava14Records benchmarks Java 14 records parsing
func BenchmarkJava14Records(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java14Records.java")

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}
}

// BenchmarkKotlinSimple benchmarks simple Kotlin parsing
func BenchmarkKotlinSimple(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/kotlin/KotlinFunInterface.kt")

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageKotlin)
	opts := ast.ParseOptions{
		Language: ast.LanguageKotlin,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}
}

// BenchmarkKotlinNormalization benchmarks Kotlin parsing with normalization
func BenchmarkKotlinNormalization(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/kotlin/Kotlin15ValueClass.kt")

	b.Run("WithNormalization", func(b *testing.B) {
		parser := NewANTLRParser(JavaGrammar20, ast.LanguageKotlin)
		opts := ast.ParseOptions{
			Language: ast.LanguageKotlin,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := parser.ParseSource(source, opts)
			if err != nil {
				b.Fatalf("ParseSource failed: %v", err)
			}
			if !result.Success {
				b.Errorf("Parse failed: %v", result.Diagnostics)
			}
		}
	})
}

// BenchmarkParserWithAST benchmarks parsing with full AST construction
func BenchmarkParserWithAST(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java8Lambda.java")

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
		BuildAST: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
		if result.SourceFile == nil {
			b.Error("SourceFile is nil when BuildAST=true")
		}
	}
}

// BenchmarkParserWithoutAST benchmarks parsing without AST construction
func BenchmarkParserWithoutAST(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java8Lambda.java")

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
		BuildAST: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}
}

// BenchmarkGrammarComparison compares different Java grammars
func BenchmarkGrammarComparison(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java8Lambda.java")

	grammars := []struct {
		name    string
		grammar JavaGrammar
	}{
		{"Java8", JavaGrammar8},
		{"Java9", JavaGrammar9},
		{"Java20", JavaGrammar20},
	}

	for _, g := range grammars {
		b.Run(g.name, func(b *testing.B) {
			parser := NewANTLRParser(g.grammar, ast.LanguageJava)
			opts := ast.ParseOptions{
				Language: ast.LanguageJava,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := parser.ParseSource(source, opts)
				if err != nil {
					b.Fatalf("ParseSource failed: %v", err)
				}
				if !result.Success {
					b.Errorf("Parse failed: %v", result.Diagnostics)
				}
			}
		})
	}
}

// BenchmarkLargeFile benchmarks parsing a larger Java file
func BenchmarkLargeFile(b *testing.B) {
	// Create a larger synthetic source file
	source := generateLargeJavaSource(100) // 100 classes

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}
}

// BenchmarkMemoryProfiling measures memory allocation during parsing
func BenchmarkMemoryProfiling(b *testing.B) {
	source := loadTestFile(b, "../../testdata/samples/java/Java14Records.java")

	b.ReportAllocs()
	b.ResetTimer()

	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{
		Language: ast.LanguageJava,
		BuildAST: true,
	}

	for i := 0; i < b.N; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			b.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			b.Errorf("Parse failed: %v", result.Diagnostics)
		}
		_ = result.SourceFile
	}
}

// loadTestFile is a helper to load test files
func loadTestFile(b *testing.B, relativePath string) []byte {
	b.Helper()

	fullPath := filepath.Join(".", relativePath)
	source, err := os.ReadFile(fullPath)
	if err != nil {
		b.Fatalf("Failed to read test file %s: %v", relativePath, err)
	}
	return source
}

// generateLargeJavaSource generates a synthetic Java source file with multiple classes
func generateLargeJavaSource(classCount int) []byte {
	var src []byte
	src = append(src, []byte("package test;\n\nimport java.util.*;\nimport java.util.function.*;\n\n")...)

	for i := 0; i < classCount; i++ {
		src = append(src, []byte("class Class"+string(rune('A'+i%26))+" {\n")...)
		src = append(src, []byte("    private String name;\n")...)
		src = append(src, []byte("    private int value;\n\n")...)
		src = append(src, []byte("    public Class"+string(rune('A'+i%26))+"(String name, int value) {\n")...)
		src = append(src, []byte("        this.name = name;\n")...)
		src = append(src, []byte("        this.value = value;\n")...)
		src = append(src, []byte("    }\n\n")...)
		src = append(src, []byte("    public String getName() { return name; }\n")...)
		src = append(src, []byte("    public int getValue() { return value; }\n\n")...)
		src = append(src, []byte("    public void process(Function<String, String> f) {\n")...)
		src = append(src, []byte("        String result = f.apply(name);\n")...)
		src = append(src, []byte("        System.out.println(result);\n")...)
		src = append(src, []byte("    }\n")...)
		src = append(src, []byte("}\n\n")...)
	}

	return src
}
