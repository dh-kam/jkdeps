package mixedgraph

import (
	"os"
	"path/filepath"
	"testing"

	ast "github.com/dh-kam/jkdeps/internal/ast"
	sharedparser "github.com/dh-kam/jkdeps/internal/parser"
)

func benchmarkJavaParseJob() parseJob {
	return parseJob{
		path: filepath.ToSlash(filepath.Join("src", "main", "java", "sample", "Bench.java")),
		rel:  "src/main/java/sample/Bench.java",
		lang: LangJava,
	}
}

func benchmarkJavaParseOptions() ParseOptions {
	return ParseOptions{
		JavaGrammar:      sharedparser.JavaGrammar20,
		JavaParseMode:    JavaParseModeFull,
		MaxErrorsPerFile: 10,
		LenientSyntax:    false,
	}
}

func benchmarkSharedJavaSource(job parseJob, source []byte, opts ParseOptions) FileUnit {
	result := sharedparser.ParseWithDiagnostics(job.path, source, ast.LanguageJava, opts.JavaGrammar, opts.MaxErrorsPerFile)
	return FileUnit{
		Path:        job.path,
		Relative:    job.rel,
		Language:    LangJava,
		PackageName: result.PackageName,
		Imports:     append([]string(nil), result.Imports...),
		Parsed:      result.Parsed || opts.LenientSyntax,
		Diagnostics: convertParserDiagnostics(result.Diagnostics),
		Duration:    result.Duration,
	}
}

func BenchmarkParseJavaSourceStrategies(b *testing.B) {
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "samples", "java", "Java8Lambda.java"))
	if err != nil {
		b.Fatalf("read benchmark source: %v", err)
	}
	job := benchmarkJavaParseJob()
	opts := benchmarkJavaParseOptions()

	b.Run("mixedgraph_ll_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			unit := parseJavaSource(job, source, opts)
			if !unit.Parsed {
				b.Fatalf("expected parsed source, got diagnostics=%v", unit.Diagnostics)
			}
		}
	})

	b.Run("shared_fallback", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			unit := benchmarkSharedJavaSource(job, source, opts)
			if !unit.Parsed {
				b.Fatalf("expected parsed source, got diagnostics=%v", unit.Diagnostics)
			}
		}
	})
}

func BenchmarkParseJavaSourceStrategiesRealFile(b *testing.B) {
	sourcePath := os.Getenv("JKDEPS_BENCH_JAVA_FILE")
	if sourcePath == "" {
		b.Skip("set JKDEPS_BENCH_JAVA_FILE to run real-file benchmark")
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		b.Fatalf("read benchmark source: %v", err)
	}
	job := parseJob{
		path: sourcePath,
		rel:  filepath.Base(sourcePath),
		lang: LangJava,
	}
	opts := benchmarkJavaParseOptions()

	b.Run("mixedgraph_ll_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = parseJavaSource(job, source, opts)
		}
	})

	b.Run("shared_fallback", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = benchmarkSharedJavaSource(job, source, opts)
		}
	})
}

func BenchmarkParseJavaSourceModesRealFile(b *testing.B) {
	sourcePath := os.Getenv("JKDEPS_BENCH_JAVA_FILE")
	if sourcePath == "" {
		b.Skip("set JKDEPS_BENCH_JAVA_FILE to run real-file benchmark")
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		b.Fatalf("read benchmark source: %v", err)
	}
	job := parseJob{
		path: sourcePath,
		rel:  filepath.Base(sourcePath),
		lang: LangJava,
	}

	fullOpts := benchmarkJavaParseOptions()
	headerOnlyOpts := benchmarkJavaParseOptions()
	headerOnlyOpts.JavaParseMode = JavaParseModeHeaderOnly

	b.Run("full_validation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = parseJavaSource(job, source, fullOpts)
		}
	})

	b.Run("header_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = parseJavaSource(job, source, headerOnlyOpts)
		}
	})
}
