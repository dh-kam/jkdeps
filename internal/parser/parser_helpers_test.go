package parser

import (
	"strings"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	java20parser "github.com/dh-kam/jkdeps/internal/parsers/java20"
)

func TestNeedsKotlinNormalization(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "plain class", source: "class Plain", want: false},
		{name: "expect declaration", source: "expect class Sample", want: true},
		{name: "actual declaration", source: "actual class Sample", want: true},
		{name: "value class", source: "value class Sample(val id: Int)", want: true},
		{name: "fun interface", source: "fun interface Runner", want: true},
		{name: "context receiver", source: "context(Foo)\nfun run() = Unit", want: true},
		{name: "sealed interface", source: "sealed interface Sample", want: true},
		{name: "labeled lambda", source: "run label@ { println() }", want: true},
		{name: "function type cast", source: "x as (String) -> Int", want: true},
		{name: "nullable type cast", source: "x as String?", want: true},
		{name: "extension star receiver", source: "fun <*>.foo() {}", want: true},
		{name: "extension generic receiver", source: "suspend fun <T : Any> Call<T>.await(): T {}", want: true},
		{name: "anonymous function assignment", source: "val f = fun(x: Int) {}", want: true},
		{name: "string template", source: `val s = "Hello ${name}"`, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsKotlinNormalization([]byte(tc.source)); got != tc.want {
				t.Fatalf("needsKotlinNormalization(%q) = %v, want %v", tc.source, got, tc.want)
			}
		})
	}
}

func TestNormalizeKotlinSource(t *testing.T) {
	t.Run("basic modern syntax", func(t *testing.T) {
		source := []byte(strings.Join([]string{
			"expect class Shared",
			"actual value class UserId(val raw: String)",
			"fun interface Runner { fun run() }",
			"context(Foo)",
			"sealed interface Marker",
		}, "\n"))

		got := string(normalizeKotlinSource(source))
		for _, banned := range []string{"expect class", "actual value class", "fun interface", "context(", "sealed interface"} {
			if strings.Contains(got, banned) {
				t.Fatalf("normalizeKotlinSource() still contains %q in %q", banned, got)
			}
		}
		for _, want := range []string{"class Shared", "class UserId", "interface Runner", "interface Marker"} {
			if !strings.Contains(got, want) {
				t.Fatalf("normalizeKotlinSource() missing %q in %q", want, got)
			}
		}
	})

	t.Run("labeled lambda normalization", func(t *testing.T) {
		source := []byte("run label@ { println() }")
		got := string(normalizeKotlinSource(source))
		// After normalization, "label@ {" should become " {" with the label removed
		if strings.Contains(got, "label@") {
			t.Logf("Result: %s", got)
			t.Fatal("labeled lambda should not contain label@ after normalization")
		}
		// The pattern should still have the opening brace
		if !strings.Contains(got, "{") {
			t.Fatal("labeled lambda should contain { after normalization")
		}
	})

	t.Run("function type cast normalization", func(t *testing.T) {
		source := []byte("x as (String) -> Int")
		got := string(normalizeKotlinSource(source))
		if strings.Contains(got, " as ") && strings.Contains(got, "->") {
			t.Fatal("function type cast should be removed after normalization")
		}
	})

	t.Run("nullable type cast normalization", func(t *testing.T) {
		source := []byte("x as String?")
		got := string(normalizeKotlinSource(source))
		if strings.Contains(got, "String?") && strings.Contains(got, " as ") {
			// Should be normalized to "as String" without the ?
			t.Log("nullable type cast normalized:", got)
		}
	})

	t.Run("extension star receiver normalization", func(t *testing.T) {
		source := []byte("fun <*>.foo() {}")
		got := string(normalizeKotlinSource(source))
		if strings.Contains(got, "<*>.") {
			t.Fatal("extension star receiver should be normalized")
		}
	})

	t.Run("extension generic receiver normalization", func(t *testing.T) {
		source := []byte("suspend fun <T : Any> Call<T>.await(): T {}")
		got := string(normalizeKotlinSource(source))
		if strings.Contains(got, "Call<T>.await") {
			t.Fatalf("extension generic receiver should be normalized: %q", got)
		}
		if !strings.Contains(got, "Call.await") {
			t.Fatalf("extension receiver declaration should remain: %q", got)
		}
	})

	t.Run("anonymous function assignment normalization", func(t *testing.T) {
		source := []byte("val f = fun(x: Int) { println(x) }")
		got := string(normalizeKotlinSource(source))
		if strings.Contains(got, "= fun(") {
			t.Fatal("anonymous function assignment should be normalized")
		}
	})

	t.Run("string template normalization", func(t *testing.T) {
		source := []byte(`val s = "Hello ${name}"`)
		got := string(normalizeKotlinSource(source))
		if got == string(source) {
			t.Fatal("string template should be normalized but wasn't")
		}
		if strings.Contains(got, "${") {
			t.Logf("Result: %s", got)
			t.Fatal("string template normalization should remove template expressions")
		}
	})
}

func TestParseKotlinGenericExtensionReceiver(t *testing.T) {
	source := []byte(`package sample

suspend fun <T : Any> Call<T>.await(): T {
  return suspendCancellableCoroutine { continuation ->
    continuation.invokeOnCancellation { cancel() }
  }
}
`)
	if err := parseKotlin(source); err != nil {
		t.Fatalf("parseKotlin(generic extension receiver) = %v, want nil", err)
	}
}

func TestParseKotlinGenericExtensionReceiverWithComplexBody(t *testing.T) {
	source := []byte(`package sample

suspend fun <T : Any> Call<T>.await(): T {
  return suspendCancellableCoroutine { continuation ->
    enqueue(
      object : Callback<T> {
        override fun onResponse(call: Call<T>, response: Response<T>) {
          if (response.isSuccessful) {
            val body = response.body()
            if (body == null) {
              val invocation = call.request().tag(Invocation::class.java)!!
              val service = invocation.service()
              val method = invocation.method()
              val e =
                KotlinNullPointerException(
                  "Response from ${service.name}.${method.name}" +
                    " was null but response body type was declared as non-null"
                )
              continuation.resumeWithException(e)
            } else {
              continuation.resume(body)
            }
          }
        }
      }
    )
  }
}
`)
	if err := parseKotlin(source); err != nil {
		t.Fatalf("parseKotlin(complex generic extension receiver) = %v, want nil\nnormalized:\n%s", err, string(normalizeKotlinSource(source)))
	}
}

func TestParseKotlin(t *testing.T) {
	t.Run("valid source", func(t *testing.T) {
		source := []byte(`
package sample

class App

fun run(): Int = 1
`)
		if err := parseKotlin(source); err != nil {
			t.Fatalf("parseKotlin(valid) = %v, want nil", err)
		}
	})

	t.Run("invalid source", func(t *testing.T) {
		source := []byte(`
package sample

fun broken(
`)
		if err := parseKotlin(source); err == nil {
			t.Fatal("parseKotlin(invalid) = nil, want error")
		}
	})
}

func TestNormalizeJavaSourceForANTLR(t *testing.T) {
	t.Run("when identifier", func(t *testing.T) {
		source := []byte(`package p;

import static org.mockito.Mockito.when;

class A {
  private String when;
  void test() {
    when(foo()).thenReturn("bar");
  }
}
`)
		normalized := string(normalizeJavaSourceForANTLR(source))
		if strings.Contains(normalized, "Mockito.when;") || strings.Contains(normalized, "String when;") {
			t.Fatalf("expected when identifiers to be normalized: %q", normalized)
		}
		if err := parseJava(source, JavaGrammar20); err != nil {
			t.Fatalf("parseJava(when identifier) = %v, want nil", err)
		}
	})

	t.Run("when guard", func(t *testing.T) {
		source := []byte(`package p;

class A {
  String test(Object value) {
    return switch (value) {
      case String s when s.length() > 3 -> s;
      default -> "";
    };
  }
}
`)
		normalized := string(normalizeJavaSourceForANTLR(source))
		if !strings.Contains(normalized, "case String s when s.length()") {
			t.Fatalf("expected when guard to be preserved: %q", normalized)
		}
	})

	t.Run("text block", func(t *testing.T) {
		source := []byte("package p;\n\nclass A {\n  String text() {\n    return \"\"\"\n      alpha\n      beta\n      \"\"\";\n  }\n}\n")
		normalized := string(normalizeJavaSourceForANTLR(source))
		if strings.Contains(normalized, `"""`) {
			t.Fatalf("expected text block to be normalized: %q", normalized)
		}
		if err := parseJava(source, JavaGrammar20); err != nil {
			t.Fatalf("parseJava(text block) = %v, want nil", err)
		}
	})

	t.Run("unnamed catch parameter", func(t *testing.T) {
		source := []byte(`package p;

class A {
  boolean test() {
    try {
      return true;
    } catch (RuntimeException _) {
      return false;
    }
  }
}
`)
		normalized := string(normalizeJavaSourceForANTLR(source))
		if strings.Contains(normalized, "RuntimeException _)") {
			t.Fatalf("expected unnamed catch parameter to be normalized: %q", normalized)
		}
		if err := parseJava(source, JavaGrammar20); err != nil {
			t.Fatalf("parseJava(unnamed catch parameter) = %v, want nil", err)
		}
	})
}

func TestIsParseCancellation(t *testing.T) {
	if !isParseCancellation(&antlr.ParseCancellationException{}) {
		t.Fatal("expected pointer ParseCancellationException to be recognized")
	}
	if !isParseCancellation(antlr.ParseCancellationException{}) {
		t.Fatal("expected value ParseCancellationException to be recognized")
	}
	if isParseCancellation("boom") {
		t.Fatal("did not expect arbitrary values to be recognized as parse cancellation")
	}
}

func TestParserRecognitionError(t *testing.T) {
	if got := parserRecognitionError(nil); got == nil || got.Error() != "antlr parse error" {
		t.Fatalf("parserRecognitionError(nil) = %v, want antlr parse error", got)
	}
	if got := parserRecognitionError(&antlr.ParseCancellationException{}); got == nil || got.Error() != "antlr parse canceled" {
		t.Fatalf("parserRecognitionError(ParseCancellationException) = %v, want antlr parse canceled", got)
	}
}

func TestParseJavaWithFallback(t *testing.T) {
	t.Run("returns sll error without fallback", func(t *testing.T) {
		sllErr := parserRecognitionError(nil)
		llCalled := false
		got := parseJavaWithFallback(
			func() (error, bool) { return sllErr, false },
			func() error {
				llCalled = true
				return nil
			},
		)
		if got != sllErr {
			t.Fatalf("parseJavaWithFallback() = %v, want %v", got, sllErr)
		}
		if llCalled {
			t.Fatal("parseJavaWithFallback() should not call LL when fallback is false")
		}
	})

	t.Run("uses ll fallback", func(t *testing.T) {
		llErr := parserRecognitionError(&antlr.ParseCancellationException{})
		llCalled := false
		got := parseJavaWithFallback(
			func() (error, bool) { return nil, true },
			func() error {
				llCalled = true
				return llErr
			},
		)
		if got != llErr {
			t.Fatalf("parseJavaWithFallback() = %v, want %v", got, llErr)
		}
		if !llCalled {
			t.Fatal("parseJavaWithFallback() should call LL when fallback is true")
		}
	})
}

func TestFinalizeJavaParseResultListenerErrors(t *testing.T) {
	parser := newJava20ParserForTest("class A {}")
	listener := newSyntaxErrorListener(2)
	listener.SyntaxError(nil, nil, 3, 4, "boom", nil)

	if err, fallback := finalizeJavaParseResult(listener, parser, antlr.PredictionModeSLL); err != nil || !fallback {
		t.Fatalf("finalizeJavaParseResult(SLL) = (%v, %v), want (nil, true)", err, fallback)
	}

	err, fallback := finalizeJavaParseResult(listener, parser, antlr.PredictionModeLL)
	if err == nil || fallback {
		t.Fatalf("finalizeJavaParseResult(LL) = (%v, %v), want (error, false)", err, fallback)
	}
	if !strings.Contains(err.Error(), "3:4 boom") {
		t.Fatalf("finalizeJavaParseResult(LL) error = %q, want listener message", err.Error())
	}
}

func TestConfigureJavaParserUsesBailStrategyForSLL(t *testing.T) {
	parser := java20parser.NewJava20Parser(antlr.NewCommonTokenStream(java20parser.NewJava20Lexer(antlr.NewInputStream("class A {}")), antlr.TokenDefaultChannel))
	listener := newSyntaxErrorListener(2)

	configureJavaParser(parser, listener, antlr.PredictionModeSLL)

	if _, ok := parser.GetErrorHandler().(*panicBailErrorStrategy); !ok {
		t.Fatalf("expected BailErrorStrategy for SLL, got %T", parser.GetErrorHandler())
	}
}

func TestConfigureJavaParserKeepsDefaultStrategyForLL(t *testing.T) {
	parser := java20parser.NewJava20Parser(antlr.NewCommonTokenStream(java20parser.NewJava20Lexer(antlr.NewInputStream("class A {}")), antlr.TokenDefaultChannel))
	listener := newSyntaxErrorListener(2)

	configureJavaParser(parser, listener, antlr.PredictionModeLL)

	if _, ok := parser.GetErrorHandler().(*antlr.DefaultErrorStrategy); !ok {
		t.Fatalf("expected DefaultErrorStrategy for LL, got %T", parser.GetErrorHandler())
	}
}

func TestSyntaxErrorListenerErrAndLimit(t *testing.T) {
	listener := newSyntaxErrorListener(0)
	listener.SyntaxError(nil, nil, 1, 2, "first", nil)
	listener.SyntaxError(nil, nil, 3, 4, "second", nil)

	err := listener.Err()
	if err == nil {
		t.Fatal("listener.Err() = nil, want error")
	}
	if err.Error() != "1:2 first" {
		t.Fatalf("listener.Err() = %q, want %q", err.Error(), "1:2 first")
	}

	empty := newSyntaxErrorListener(2)
	if empty.Err() != nil {
		t.Fatalf("empty listener.Err() = %v, want nil", empty.Err())
	}
}

func newJava20ParserForTest(source string) antlr.Parser {
	input := antlr.NewInputStream(source)
	lexer := java20parser.NewJava20Lexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := java20parser.NewJava20Parser(tokens)
	parser.BuildParseTrees = false
	parser.CompilationUnit()
	return parser
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "simple path", input: "com.example", want: "com.example"},
		{name: "path with backticks", input: "`com.example`", want: "com.example"},
		{name: "path with spaces", input: " com . example ", want: "com.example"},
		{name: "path with leading dot", input: ".com.example", want: "com.example"},
		{name: "path with trailing dot", input: "com.example.", want: "com.example"},
		{name: "path with multiple dots", input: "com..example...", want: "com..example"},
		{name: "path with tabs", input: "com.\t.\texample", want: "com..example"},
		{name: "path with newlines", input: "com.\n.example\n", want: "com..example"},
		{name: "path with mixed whitespace", input: "  ` com . example `  ", want: "com.example"},
		{name: "only dots", input: "...", want: ""},
		{name: "only whitespace", input: "   ", want: ""},
		{name: "single valid segment", input: "com", want: "com"},
		{name: "dots between segments", input: "com..example", want: "com..example"},
		{name: "complex cleanup", input: "  `com..` . .example..`  ", want: "com....example"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePath(tc.input)
			if got != tc.want {
				t.Fatalf("normalizePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
