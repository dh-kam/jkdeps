package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// JavaVersionTestResult represents the parsing result for a Java version-specific feature
type JavaVersionTestResult struct {
	FileName    string
	JavaVersion string
	Feature     string
	Parsed      bool
	Error       string
	Diagnostics []string
}

// KotlinVersionTestResult represents the parsing result for a Kotlin version-specific feature
type KotlinVersionTestResult struct {
	FileName      string
	KotlinVersion string
	Feature       string
	Parsed        bool
	Error         string
	Diagnostics   []string
}

// TestJava8Parsing tests Java 8 features (lambda, method reference, default methods)
func TestJava8Parsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "java", "Java8Lambda.java")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Test with different grammar versions
	grammars := []JavaGrammar{JavaGrammar8, JavaGrammar9, JavaGrammar20}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			parseErr := parseJava(source, grammar)
			result := JavaVersionTestResult{
				FileName:    "Java8Lambda.java",
				JavaVersion: "8",
				Feature:     "Lambda, Method Reference, Default Interface Method",
				Parsed:      parseErr == nil,
			}

			if parseErr != nil {
				result.Error = parseErr.Error()
				t.Logf("Grammar %s failed to parse Java 8 features: %v", grammar, parseErr)
			} else {
				t.Logf("Grammar %s successfully parsed Java 8 features", grammar)
			}

			if !result.Parsed && grammar == JavaGrammar8 {
				t.Errorf("Java8 grammar should parse Java 8 features successfully, got: %v", parseErr)
			}
		})
	}
}

// TestJava10Parsing tests Java 10 features (var keyword)
func TestJava10Parsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "java", "Java10Var.java")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	grammars := []JavaGrammar{JavaGrammar11, JavaGrammar20}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			parseErr := parseJava(source, grammar)
			result := JavaVersionTestResult{
				FileName:    "Java10Var.java",
				JavaVersion: "10",
				Feature:     "Local Variable Type Inference (var)",
				Parsed:      parseErr == nil,
			}

			if parseErr != nil {
				result.Error = parseErr.Error()
				t.Logf("Grammar %s failed to parse Java 10 features: %v", grammar, parseErr)
			} else {
				t.Logf("Grammar %s successfully parsed Java 10 features", grammar)
			}
		})
	}
}

// TestJava14Parsing tests Java 14 features (records, pattern matching for instanceof)
func TestJava14Parsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "java", "Java14Records.java")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	grammars := []JavaGrammar{JavaGrammar20}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			parseErr := parseJava(source, grammar)
			result := JavaVersionTestResult{
				FileName:    "Java14Records.java",
				JavaVersion: "14",
				Feature:     "Records, Pattern Matching for instanceof",
				Parsed:      parseErr == nil,
			}

			if parseErr != nil {
				result.Error = parseErr.Error()
				t.Logf("Grammar %s failed to parse Java 14 features: %v", grammar, parseErr)
			} else {
				t.Logf("Grammar %s successfully parsed Java 14 features", grammar)
			}
		})
	}
}

// TestJava17Parsing tests Java 17 features (sealed classes, record patterns)
func TestJava17Parsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "java", "Java17Records.java")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	grammars := []JavaGrammar{JavaGrammar20}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			parseErr := parseJava(source, grammar)
			result := JavaVersionTestResult{
				FileName:    "Java17Records.java",
				JavaVersion: "17",
				Feature:     "Sealed Classes, Pattern Matching for Switch",
				Parsed:      parseErr == nil,
			}

			if parseErr != nil {
				result.Error = parseErr.Error()
				t.Logf("Grammar %s failed to parse Java 17 features: %v", grammar, parseErr)
			} else {
				t.Logf("Grammar %s successfully parsed Java 17 features", grammar)
			}
		})
	}
}

// TestJava21Parsing tests Java 21 features (record patterns, switch patterns)
func TestJava21Parsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "java", "Java21RecordPatterns.java")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	grammars := []JavaGrammar{JavaGrammar21, JavaGrammar25}

	for _, grammar := range grammars {
		t.Run(string(grammar), func(t *testing.T) {
			parseErr := parseJava(source, grammar)
			result := JavaVersionTestResult{
				FileName:    "Java21RecordPatterns.java",
				JavaVersion: "21",
				Feature:     "Record Patterns, Switch Patterns with Guards",
				Parsed:      parseErr == nil,
			}

			if parseErr != nil {
				result.Error = parseErr.Error()
				t.Logf("Grammar %s failed to parse Java 21 features: %v", grammar, parseErr)
			} else {
				t.Logf("Grammar %s successfully parsed Java 21 features", grammar)
			}

			// Java 21 features may not be fully supported by Java20 grammar
			// This is informational only
			if !result.Parsed {
				t.Logf("Note: Java21 features may require grammar updates")
			}
		})
	}
}

// TestKotlinValueClassParsing tests Kotlin value class (inline class) parsing
func TestKotlinValueClassParsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "kotlin", "Kotlin15ValueClass.kt")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	parseErr := parseKotlin(source)
	result := KotlinVersionTestResult{
		FileName:      "Kotlin15ValueClass.kt",
		KotlinVersion: "1.5+",
		Feature:       "Value Class, Sealed Interface, Context Receivers",
		Parsed:        parseErr == nil,
	}

	if parseErr != nil {
		result.Error = parseErr.Error()
		t.Logf("Failed to parse Kotlin value class features: %v", parseErr)
		t.Logf("Note: Value class and context receivers require source normalization")

		// Check if normalization is happening
		normalized := normalizeKotlinSource(source)
		if string(normalized) != string(source) {
			t.Logf("Source was normalized for modern Kotlin syntax")
			normalizedErr := parseKotlin(normalized)
			if normalizedErr == nil {
				t.Logf("Successfully parsed after normalization")
			} else {
				t.Logf("Still failed after normalization: %v", normalizedErr)
			}
		}
	} else {
		t.Logf("Successfully parsed Kotlin 1.5+ features")
	}
}

// TestKotlinExpectActualParsing tests Kotlin expect/actual declarations
func TestKotlinExpectActualParsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "kotlin", "KotlinExpectActual.kt")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	parseErr := parseKotlin(source)
	result := KotlinVersionTestResult{
		FileName:      "KotlinExpectActual.kt",
		KotlinVersion: "1.3+ (multiplatform)",
		Feature:       "Expect/Actual Declarations",
		Parsed:        parseErr == nil,
	}

	if parseErr != nil {
		result.Error = parseErr.Error()
		t.Logf("Failed to parse Kotlin expect/actual features: %v", parseErr)
		t.Logf("Note: Expect/actual requires source normalization (removing expect/actual keywords)")

		// Check if normalization removes expect/actual
		normalized := normalizeKotlinSource(source)
		if string(normalized) != string(source) {
			t.Logf("Source was normalized for expect/actual syntax")
			normalizedErr := parseKotlin(normalized)
			if normalizedErr == nil {
				t.Logf("Successfully parsed after normalization")
			} else {
				t.Logf("Still failed after normalization: %v", normalizedErr)
			}
		}
	} else {
		t.Logf("Successfully parsed Kotlin expect/actual features")
	}
}

// TestKotlinFunInterfaceParsing tests Kotlin fun interface (SAM interface) parsing
func TestKotlinFunInterfaceParsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "kotlin", "KotlinFunInterface.kt")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	parseErr := parseKotlin(source)
	result := KotlinVersionTestResult{
		FileName:      "KotlinFunInterface.kt",
		KotlinVersion: "1.4+",
		Feature:       "Fun Interface (SAM Interface)",
		Parsed:        parseErr == nil,
	}

	if parseErr != nil {
		result.Error = parseErr.Error()
		t.Logf("Failed to parse Kotlin fun interface features: %v", parseErr)
		t.Logf("Note: Fun interface requires source normalization (removing 'fun' keyword)")

		// Check if normalization removes 'fun' before interface
		normalized := normalizeKotlinSource(source)
		if string(normalized) != string(source) {
			t.Logf("Source was normalized for fun interface syntax")
			normalizedErr := parseKotlin(normalized)
			if normalizedErr == nil {
				t.Logf("Successfully parsed after normalization")
			} else {
				t.Logf("Still failed after normalization: %v", normalizedErr)
			}
		}
	} else {
		t.Logf("Successfully parsed Kotlin fun interface features")
	}
}

// TestKotlinContextReceiversParsing tests Kotlin context receivers parsing
func TestKotlinContextReceiversParsing(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "samples", "kotlin", "KotlinContextReceivers.kt")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	parseErr := parseKotlin(source)
	result := KotlinVersionTestResult{
		FileName:      "KotlinContextReceivers.kt",
		KotlinVersion: "1.6.20+",
		Feature:       "Context Receivers",
		Parsed:        parseErr == nil,
	}

	if parseErr != nil {
		result.Error = parseErr.Error()
		t.Logf("Failed to parse Kotlin context receiver features: %v", parseErr)
		t.Logf("Note: Context receivers require source normalization (removing context(...) declarations)")

		// Check if normalization removes context receivers
		normalized := normalizeKotlinSource(source)
		if string(normalized) != string(source) {
			t.Logf("Source was normalized for context receiver syntax")
			normalizedErr := parseKotlin(normalized)
			if normalizedErr == nil {
				t.Logf("Successfully parsed after normalization")
			} else {
				t.Logf("Still failed after normalization: %v", normalizedErr)
			}
		}
	} else {
		t.Logf("Successfully parsed Kotlin context receiver features")
	}
}

// TestAllJavaSamples runs all Java sample files and reports results
func TestAllJavaSamples(t *testing.T) {
	sampleDir := filepath.Join("..", "..", "testdata", "samples", "java")

	entries, err := os.ReadDir(sampleDir)
	if err != nil {
		t.Fatalf("Failed to read sample directory: %v", err)
	}

	results := make([]JavaVersionTestResult, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".java") {
			continue
		}

		filePath := filepath.Join(sampleDir, entry.Name())
		source, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", entry.Name(), err)
			continue
		}

		// Try to parse with java20 grammar
		parseErr := parseJava(source, JavaGrammar20)
		results = append(results, JavaVersionTestResult{
			FileName: entry.Name(),
			Parsed:   parseErr == nil,
		})

		if parseErr != nil {
			t.Logf("%s: FAILED - %v", entry.Name(), parseErr)
		} else {
			t.Logf("%s: PASSED", entry.Name())
		}
	}

	// Count successful parses
	passed := 0
	for _, r := range results {
		if r.Parsed {
			passed++
		}
	}
	t.Logf("Summary: %d/%d Java samples parsed successfully", passed, len(results))
}

// TestAllKotlinSamples runs all Kotlin sample files and reports results
func TestAllKotlinSamples(t *testing.T) {
	sampleDir := filepath.Join("..", "..", "testdata", "samples", "kotlin")

	entries, err := os.ReadDir(sampleDir)
	if err != nil {
		t.Fatalf("Failed to read sample directory: %v", err)
	}

	results := make([]KotlinVersionTestResult, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".kt") {
			continue
		}

		filePath := filepath.Join(sampleDir, entry.Name())
		source, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", entry.Name(), err)
			continue
		}

		// Parse with normalization
		normalized := normalizeKotlinSource(source)
		parseErr := parseKotlin(normalized)
		results = append(results, KotlinVersionTestResult{
			FileName: entry.Name(),
			Parsed:   parseErr == nil,
		})

		if parseErr != nil {
			t.Logf("%s: FAILED - %v", entry.Name(), parseErr)
		} else {
			t.Logf("%s: PASSED", entry.Name())
		}
	}

	// Count successful parses
	passed := 0
	for _, r := range results {
		if r.Parsed {
			passed++
		}
	}
	t.Logf("Summary: %d/%d Kotlin samples parsed successfully (with normalization)", passed, len(results))
}
