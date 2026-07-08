package kotlincompilergolang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDebugSamplePath(t *testing.T) {
	t.Parallel()

	origRoot := os.Getenv("KCG_SAMPLE_ROOT")
	t.Cleanup(func() {
		_ = os.Setenv("KCG_SAMPLE_ROOT", origRoot)
	})

	cases := []struct {
		name string
		root string
		in   string
		want string
	}{
		{
			name: "default-root-absent",
			root: "",
			in:   debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
			want: debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
		},
		{
			name: "mapped-root",
			root: "/workspace/samples",
			in:   debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
			want: filepath.Join("/workspace/samples", "kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
		},
		{
			name: "non-prefixed-path-unchanged",
			root: "/workspace/samples",
			in:   "/tmp/other-samples/foo.kt",
			want: "/tmp/other-samples/foo.kt",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()
			_ = os.Setenv("KCG_SAMPLE_ROOT", tc.root)
			got := resolveDebugSamplePath(tc.in)
			if got != tc.want {
				t.Fatalf("resolveDebugSamplePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
