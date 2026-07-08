package kotlincompilergolang

import "testing"

func TestIsKnownExternalImport(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "java.util.concurrent.*", want: true},
		{path: "org.junit.Test", want: true},
		{path: "platform.posix.*", want: true},
		{path: "org.jetbrains.kotlinx.lincheck.annotations.*", want: true},
		{path: "kotlinx.cinterop.*", want: true},
		{path: "org.gradle.kotlin.dsl.*", want: true},
		{path: "com.example.internal", want: false},
		{path: "", want: false},
	}

	for _, tc := range cases {
		got := isKnownExternalImport(tc.path)
		if got != tc.want {
			t.Fatalf("isKnownExternalImport(%q)=%v, want %v", tc.path, got, tc.want)
		}
	}
}
