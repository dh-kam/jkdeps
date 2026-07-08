package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaGrammarAliasesAreValid(t *testing.T) {
	valid := []JavaGrammar{
		JavaGrammarOrig,
		JavaGrammar7,
		JavaGrammar8,
		JavaGrammar9,
		JavaGrammar11,
		JavaGrammar17,
		JavaGrammar20,
		JavaGrammar21,
		JavaGrammar25,
	}
	for _, grammar := range valid {
		if !grammar.IsValid() {
			t.Fatalf("expected grammar %q to be valid", grammar)
		}
	}
}

func TestJavaGrammarVersionsParseRepresentativeFiles(t *testing.T) {
	type fixture struct {
		grammar JavaGrammar
		file    string
		body    string
	}

	cases := []fixture{
		{
			grammar: JavaGrammarOrig,
			file:    "JavaOrigFeature.java",
			body: `
package sample;

public class JavaOrigFeature {
  public int value() {
    return 1;
  }
}
`,
		},
		{
			grammar: JavaGrammar7,
			file:    "Java7Feature.java",
			body: `
package sample;

import java.io.ByteArrayInputStream;

public class Java7Feature {
  public int size() throws Exception {
    try (ByteArrayInputStream input = new ByteArrayInputStream(new byte[] {1, 2, 3})) {
      return input.available();
    }
  }
}
`,
		},
		{
			grammar: JavaGrammar8,
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
			grammar: JavaGrammar9,
			file:    "Java9Feature.java",
			body: `
package sample;

interface Java9Feature {
  private static void ping() {}
}
`,
		},
		{
			grammar: JavaGrammar11,
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
			grammar: JavaGrammar17,
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
			grammar: JavaGrammar21,
			file:    "Java21Feature.java",
			body: `
package sample;

public record Java21Feature(String id, int count) {
}
`,
		},
		{
			grammar: JavaGrammar25,
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
		t.Run(string(tc.grammar), func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "src", "main", "java", tc.file)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write sample: %v", err)
			}

			summary, err := ParseRepository(root, ParseOptions{
				JavaGrammar: tc.grammar,
				Workers:     1,
				IncludeKTS:  false,
			})
			if err != nil {
				t.Fatalf("parse failed for %s: %v", tc.grammar, err)
			}
			if summary.TotalFiles != 1 || summary.JavaFiles != 1 || summary.ParsedFiles != 1 || summary.FailedFiles != 0 {
				t.Fatalf("unexpected parse summary for %s: total=%d java=%d parsed=%d failed=%d", tc.grammar, summary.TotalFiles, summary.JavaFiles, summary.ParsedFiles, summary.FailedFiles)
			}
		})
	}
}
