package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// ProfilingResult represents the combined profiling results
type ProfilingResult struct {
	Timestamp      time.Time               `json:"timestamp"`
	Benchmarks     []BenchmarkProfileEntry `json:"benchmarks"`
	MemoryProfiles []MemoryProfileEntry    `json:"memory_profiles,omitempty"`
}

// BenchmarkProfileEntry represents a single benchmark result
type BenchmarkProfileEntry struct {
	Name          string `json:"name"`
	Iterations    int    `json:"iterations"`
	DurationNs    int64  `json:"duration_ns"`
	AvgDurationNs int64  `json:"avg_duration_ns"`
	BytesPerOp    uint64 `json:"bytes_per_op"`
	AllocsPerOp   uint64 `json:"allocs_per_op"`
}

// MemoryProfileEntry represents memory usage during a parse operation
type MemoryProfileEntry struct {
	Name         string          `json:"name"`
	FileSize     int64           `json:"file_size"`
	Language     string          `json:"language"`
	Duration     int64           `json:"duration_ns"`
	MemoryBefore ast.MemoryUsage `json:"memory_before"`
	MemoryAfter  ast.MemoryUsage `json:"memory_after"`
	MemoryDelta  ast.MemoryUsage `json:"memory_delta"`
}

// TestProfilingWithMetrics tests parsing with detailed metrics
func TestProfilingWithMetrics(t *testing.T) {
	tests := []struct {
		name     string
		language string
		fileName string
	}{
		{"Java8", "java", "Java8Lambda.java"},
		{"Java10", "java", "Java10Var.java"},
		{"Java14", "java", "Java14Records.java"},
		{"KotlinFunInterface", "kotlin", "KotlinFunInterface.kt"},
		{"KotlinValueClass", "kotlin", "Kotlin15ValueClass.kt"},
	}

	result := ProfilingResult{
		Timestamp:      time.Now(),
		Benchmarks:     make([]BenchmarkProfileEntry, 0),
		MemoryProfiles: make([]MemoryProfileEntry, 0),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := loadTestFileProfiling(t, "../../testdata/samples/"+tt.language+"/"+tt.fileName)
			fileSize := int64(len(source))

			var lang ast.SourceLanguage
			if tt.language == "java" {
				lang = ast.LanguageJava
			} else {
				lang = ast.LanguageKotlin
			}

			parser := NewANTLRParser(JavaGrammar20, lang)

			// Track memory
			tracker := ast.StartMemoryTracking()
			start := time.Now()

			opts := ast.ParseOptions{
				Language: lang,
				BuildAST: true,
			}

			parseResult, err := parser.ParseSource(source, opts)
			if err != nil {
				t.Fatalf("ParseSource failed: %v", err)
			}

			duration := time.Since(start)
			delta := tracker.Stop()

			if !parseResult.Success {
				t.Errorf("Parse failed: %v", parseResult.Diagnostics)
			}

			// Record memory profile
			result.MemoryProfiles = append(result.MemoryProfiles, MemoryProfileEntry{
				Name:         tt.name,
				FileSize:     fileSize,
				Language:     tt.language,
				Duration:     duration.Nanoseconds(),
				MemoryBefore: tracker.Before(),
				MemoryAfter:  ast.GetMemoryUsage(),
				MemoryDelta:  delta,
			})

			t.Logf("%s: Duration=%v, DeltaAlloc=%d, DeltaHeapObjects=%d",
				tt.name, duration, delta.Alloc, delta.HeapObjects)
		})
	}

	// Save results to file if specified
	if outputPath := os.Getenv("PROFILE_OUTPUT_PATH"); outputPath != "" {
		if err := saveProfilingResult(result, outputPath); err != nil {
			t.Errorf("Failed to save profiling results: %v", err)
		} else {
			t.Logf("Profiling results saved to %s", outputPath)
		}
	}
}

// TestParserComparison compares different parser configurations
func TestParserComparison(t *testing.T) {
	source := loadTestFileProfiling(t, "../../testdata/samples/java/Java8Lambda.java")

	grammars := []struct {
		name    string
		grammar JavaGrammar
	}{
		{"Java8", JavaGrammar8},
		{"Java9", JavaGrammar9},
		{"Java20", JavaGrammar20},
	}

	result := ProfilingResult{
		Timestamp:  time.Now(),
		Benchmarks: make([]BenchmarkProfileEntry, 0),
	}

	for _, g := range grammars {
		t.Run(g.name, func(t *testing.T) {
			parser := NewANTLRParser(g.grammar, ast.LanguageJava)
			opts := ast.ParseOptions{Language: ast.LanguageJava}

			// Warm up
			for i := 0; i < 3; i++ {
				parser.ParseSource(source, opts)
			}

			// Measure
			iterations := 10
			start := time.Now()
			totalBytes := uint64(0)
			totalAllocs := uint64(0)

			for i := 0; i < iterations; i++ {
				m1 := ast.GetMemoryUsage()
				result, err := parser.ParseSource(source, opts)
				if err != nil {
					t.Fatalf("ParseSource failed: %v", err)
				}
				if !result.Success {
					t.Errorf("Parse failed: %v", result.Diagnostics)
				}
				m2 := ast.GetMemoryUsage()
				totalBytes += (m2.Alloc - m1.Alloc)
				totalAllocs += (m2.TotalAlloc - m1.TotalAlloc)
			}

			duration := time.Since(start)

			result.Benchmarks = append(result.Benchmarks, BenchmarkProfileEntry{
				Name:          g.name,
				Iterations:    iterations,
				DurationNs:    duration.Nanoseconds(),
				AvgDurationNs: duration.Nanoseconds() / int64(iterations),
				BytesPerOp:    totalBytes / uint64(iterations),
				AllocsPerOp:   totalAllocs / uint64(iterations),
			})

			t.Logf("%s: Total=%v, Avg=%v/op, Bytes/op=%d, Allocs/op=%d",
				g.name, duration, duration/time.Duration(iterations),
				totalBytes/uint64(iterations), totalAllocs/uint64(iterations))
		})
	}

	// Save results
	if outputPath := os.Getenv("PROFILE_OUTPUT_PATH"); outputPath != "" {
		if err := saveProfilingResult(result, outputPath); err != nil {
			t.Errorf("Failed to save profiling results: %v", err)
		}
	}
}

// TestMemoryLeakDetection checks for potential memory leaks
func TestMemoryLeakDetection(t *testing.T) {
	source := loadTestFileProfiling(t, "../../testdata/samples/java/Java8Lambda.java")
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	opts := ast.ParseOptions{Language: ast.LanguageJava, BuildAST: true}

	// Force GC before starting
	runtime.GC()

	// Baseline memory
	m1 := ast.GetMemoryUsage()
	t.Logf("Baseline - Alloc=%d, HeapAlloc=%d, HeapObjects=%d",
		m1.Alloc, m1.HeapAlloc, m1.HeapObjects)

	// Parse many times
	iterations := 100
	for i := 0; i < iterations; i++ {
		result, err := parser.ParseSource(source, opts)
		if err != nil {
			t.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			t.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}

	// Force GC and measure again
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Give GC time to complete
	m2 := ast.GetMemoryUsage()

	// Calculate growth
	growth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	growthPerOp := growth / int64(iterations)

	t.Logf("After %d iterations - Alloc=%d, HeapAlloc=%d, HeapObjects=%d",
		iterations, m2.Alloc, m2.HeapAlloc, m2.HeapObjects)
	t.Logf("Growth: %d bytes total, %d bytes/op", growth, growthPerOp)

	// Allow some growth but flag excessive growth
	if growthPerOp > 1024 { // More than 1KB per operation
		t.Logf("WARNING: High memory growth detected: %d bytes/op", growthPerOp)
	}
}

// TestCPUProfiling demonstrates CPU profiling
func TestCPUProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CPU profiling in short mode")
	}

	source := loadTestFileProfiling(t, "../../testdata/samples/java/Java8Lambda.java")

	// Create profile output directory
	profileDir := os.Getenv("PROFILE_OUTPUT_DIR")
	if profileDir == "" {
		profileDir = os.TempDir()
	}

	cpuProfilePath := filepath.Join(profileDir, "cpu_profile.prof")
	memProfilePath := filepath.Join(profileDir, "mem_profile.prof")

	opts := ast.ProfilingOptions{
		CPUProfile:     true,
		CPUProfileFile: cpuProfilePath,
		MemProfile:     true,
		MemProfileFile: memProfilePath,
	}

	stop, err := ast.StartProfiling(opts)
	if err != nil {
		t.Fatalf("StartProfiling failed: %v", err)
	}

	// Run parsing operations
	parser := NewANTLRParser(JavaGrammar20, ast.LanguageJava)
	parseOpts := ast.ParseOptions{Language: ast.LanguageJava}

	for i := 0; i < 10; i++ {
		result, err := parser.ParseSource(source, parseOpts)
		if err != nil {
			t.Fatalf("ParseSource failed: %v", err)
		}
		if !result.Success {
			t.Errorf("Parse failed: %v", result.Diagnostics)
		}
	}

	stop()

	t.Logf("CPU profile saved to: %s", cpuProfilePath)
	t.Logf("Memory profile saved to: %s", memProfilePath)
}

// saveProfilingResult saves profiling results to a JSON file
func saveProfilingResult(result ProfilingResult, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadTestFileProfiling loads a test file (separate from benchmark to avoid conflict)
func loadTestFileProfiling(t *testing.T, relativePath string) []byte {
	t.Helper()

	fullPath := filepath.Join(".", relativePath)
	source, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", relativePath, err)
	}
	return source
}
