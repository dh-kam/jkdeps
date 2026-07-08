package mixedgraph

import (
	"testing"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestGraphBuildContextResolveReference(t *testing.T) {
	ctx := newGraphBuildContext([]FileUnit{
		{PackageName: "com.example"},
		{PackageName: "com.example.spi"},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
			"java.io":   {},
			"java.lang": {},
		},
	})

	file := FileUnit{
		PackageName: "com.example",
		Imports: []string{
			"java.util.List",
			"java.util.Map",
			"java.io.IOException",
			"com.example.spi.Handler",
		},
	}

	tests := []struct {
		name     string
		rawPath  string
		wantPkg  string
		wantKind NodeKind
	}{
		{name: "same package", rawPath: "BaseType", wantPkg: "com.example", wantKind: NodeInternal},
		{name: "explicit import", rawPath: "Handler", wantPkg: "com.example.spi", wantKind: NodeInternal},
		{name: "external import", rawPath: "List", wantPkg: "java.util", wantKind: NodeExternal},
		{name: "nested imported type", rawPath: "Map.Entry", wantPkg: "java.util", wantKind: NodeExternal},
		{name: "throws import", rawPath: "IOException", wantPkg: "java.io", wantKind: NodeExternal},
		{name: "implicit java.lang", rawPath: "Runnable", wantPkg: "java.lang", wantKind: NodeExternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ctx.resolveReference(file, tt.rawPath)
			if got.Package != tt.wantPkg || got.Kind != tt.wantKind {
				t.Fatalf("resolveReference(%q) = %+v, want package=%q kind=%q", tt.rawPath, got, tt.wantPkg, tt.wantKind)
			}
		})
	}
}
