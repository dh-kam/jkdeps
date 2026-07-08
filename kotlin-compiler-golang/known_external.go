package kotlincompilergolang

import "strings"

var knownExternalRoots = []string{
	"java",
	"javax",
	"jdk",
	"sun",
	"android",
}

var knownExternalPrefixes = []string{
	"org.codehaus.mojo.animal_sniffer",
	"org.junit",
	"junit.framework",
	"org.jetbrains.kotlinx.lincheck",
	"org.openjdk.jmh",
	"org.openjdk.jol",
	"platform",
	"kotlinx.cinterop",
	"kotlinx.knit",
	"kotlinx.benchmark",
	"org.gradle",
	"ru.vyarus.gradle.plugin.animalsniffer",
}

func isKnownExternalImport(importPath string) bool {
	path := normalizeImportPath(importPath)
	if path == "" {
		return false
	}
	if strings.HasSuffix(path, ".*") {
		path = strings.TrimSuffix(path, ".*")
	}
	if path == "" {
		return false
	}

	for _, root := range knownExternalRoots {
		if path == root || strings.HasPrefix(path, root+".") {
			return true
		}
	}
	for _, prefix := range knownExternalPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+".") {
			return true
		}
	}
	return false
}
