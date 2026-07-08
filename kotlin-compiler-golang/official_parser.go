package kotlincompilergolang

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type officialSnapshot struct {
	Files []officialSnapshotFile `json:"files"`
}

type officialSnapshotFile struct {
	Path         string                 `json:"path"`
	PackageName  string                 `json:"package_name"`
	Imports      []string               `json:"imports"`
	Declarations []officialSnapshotDecl `json:"declarations"`
	ErrorCount   int                    `json:"error_count"`
	Errors       []string               `json:"errors"`
}

type officialSnapshotDecl struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (c *Compiler) parseRepositoryWithEmbeddable(root string) (RepositoryResult, error) {
	startedAt := time.Now()

	snapshot, err := c.loadOfficialSnapshot(root)
	if err != nil {
		return RepositoryResult{}, err
	}

	files := make([]FileUnit, 0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}

		absPath := filepath.FromSlash(path)
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(root, absPath)
		}
		if !officialShouldIncludeFile(absPath, c.config.IncludeKTS, c.config.IncludeBuildScripts) {
			continue
		}

		declarations := officialDeclarationsToTopLevel(file.Declarations)
		imports := mergeUniqueStrings(nil, file.Imports)
		for i := range imports {
			imports[i] = normalizeQualifiedName(imports[i])
		}

		diagnostics := officialDiagnostics(file.ErrorCount, file.Errors)
		parsed := file.ErrorCount == 0
		if c.config.LenientSyntax {
			parsed = true
		}

		files = append(files, FileUnit{
			Path:         absPath,
			PackageName:  strings.TrimSpace(file.PackageName),
			Imports:      imports,
			Declarations: declarations,
			Parsed:       parsed,
			Diagnostics:  diagnostics,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	parsedFiles := 0
	failedFiles := 0
	for _, file := range files {
		if file.Parsed {
			parsedFiles++
		} else {
			failedFiles++
		}
	}

	return RepositoryResult{
		Root:        root,
		TotalFiles:  len(files),
		ParsedFiles: parsedFiles,
		FailedFiles: failedFiles,
		Files:       files,
		Duration:    time.Since(startedAt),
	}, nil
}

func (c *Compiler) loadOfficialSnapshot(root string) (officialSnapshot, error) {
	var snapshot officialSnapshot
	toolDir, err := c.embeddableToolsDir()
	if err != nil {
		return snapshot, err
	}

	embeddable, err := resolveEmbeddableJar(toolDir)
	if err != nil {
		return snapshot, err
	}
	runtimeJars, err := resolveRuntimeJars(toolDir)
	if err != nil {
		return snapshot, err
	}

	classPath := append([]string{embeddable}, runtimeJars...)
	classDir, err := compileOfficialSnapshotClasses(classPath, toolDir)
	if err != nil {
		return snapshot, err
	}

	outPath, err := os.CreateTemp("", "kcg-official-snapshot-*.json")
	if err != nil {
		return snapshot, fmt.Errorf("create temp official snapshot output: %w", err)
	}
	outFilePath := outPath.Name()
	if err := outPath.Close(); err != nil {
		_ = os.Remove(outFilePath)
		return snapshot, fmt.Errorf("close temp official snapshot output: %w", err)
	}
	defer func() {
		_ = os.Remove(outFilePath)
	}()

	javaClassPath := make([]string, 0, len(classPath)+1)
	javaClassPath = append(javaClassPath, classDir)
	javaClassPath = append(javaClassPath, classPath...)

	args := []string{
		"-cp", strings.Join(javaClassPath, string(filepath.ListSeparator)),
		"KotlinOfficialSnapshot",
		"--repo", root,
		"--out", outFilePath,
		"--include-kts", strconv.FormatBool(c.config.IncludeKTS),
	}
	if err := runCommand("java", args...); err != nil {
		return snapshot, fmt.Errorf("run official snapshot: %w", err)
	}

	data, err := os.ReadFile(outFilePath)
	if err != nil {
		return snapshot, fmt.Errorf("read official snapshot output: %w", err)
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, fmt.Errorf("parse official snapshot JSON: %w", err)
	}
	return snapshot, nil
}

func compileOfficialSnapshotClasses(classPaths []string, toolDir string) (string, error) {
	sourceRoot := filepath.Join(toolDir, "kotlin-official-parity")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	classDir := filepath.Join(toolDir, ".kcg-official-classes")
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")

	needCompile := true
	srcInfo, srcErr := os.Stat(sourcePath)
	if srcErr != nil {
		return "", fmt.Errorf("official snapshot source missing: %w", srcErr)
	}
	if srcInfo != nil {
		if classInfo, classErr := os.Stat(classFile); classErr == nil && !classInfo.ModTime().Before(srcInfo.ModTime()) {
			needCompile = false
		}
	}

	if !needCompile {
		return classDir, nil
	}

	if err := os.MkdirAll(classDir, 0o755); err != nil {
		return "", fmt.Errorf("create official snapshot classes dir: %w", err)
	}

	args := []string{
		"-proc:none",
		"-cp", strings.Join(classPaths, string(filepath.ListSeparator)),
		"-d", classDir,
		sourcePath,
	}
	if err := runCommand("javac", args...); err != nil {
		return "", fmt.Errorf("compile official snapshot: %w", err)
	}
	return classDir, nil
}

func resolveEmbeddableJar(toolDir string) (string, error) {
	scriptPath := filepath.Join(toolDir, "..", "scripts", "fetch_kotlin_compiler_embeddable.sh")
	scriptPath = filepath.Clean(scriptPath)
	out, err := runCommandOutput(scriptPath, toolDir)
	if err != nil {
		return "", fmt.Errorf("fetch kotlin-compiler-embeddable: %w", err)
	}
	jars := parseJarLines(out)
	if len(jars) == 0 {
		return "", fmt.Errorf("could not resolve kotlin-compiler-embeddable jar from %q", scriptPath)
	}
	return jars[0], nil
}

func resolveRuntimeJars(toolDir string) ([]string, error) {
	scriptPath := filepath.Join(toolDir, "..", "scripts", "fetch_kotlin_compiler_runtime_jars.sh")
	scriptPath = filepath.Clean(scriptPath)
	out, err := runCommandOutput(scriptPath, toolDir)
	if err != nil {
		return nil, fmt.Errorf("fetch kotlin runtime jars: %w", err)
	}
	jars := parseJarLines(out)
	if len(jars) == 0 {
		return nil, fmt.Errorf("could not resolve kotlin runtime jars from %q", scriptPath)
	}
	return jars, nil
}

func parseJarLines(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	jars := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if filepath.Ext(line) != ".jar" {
			continue
		}
		info, err := os.Stat(line)
		if err != nil || info.IsDir() {
			continue
		}
		jars = append(jars, line)
	}
	return jars
}

func (c *Compiler) embeddableToolsDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("JKTDEPS_TOOLS_DIR")); override != "" {
		return override, nil
	}

	root, err := findRepoRootFromCWD()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "tools"), nil
}

func findRepoRootFromCWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for i := 0; i < 16; i++ {
		modPath := filepath.Join(cwd, "go.mod")
		modText, err := os.ReadFile(modPath)
		if err == nil && strings.Contains(string(modText), "module github.com/dh-kam/jkdeps") {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	return "", fmt.Errorf("unable to locate jkdeps module root; set JKTDEPS_TOOLS_DIR")
}

func officialShouldIncludeFile(path string, includeKTS bool, includeBuildScripts bool) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".kt" {
		return true
	}
	if ext == ".kts" {
		base := strings.ToLower(filepath.Base(path))
		isBuildScript := base == "build.gradle.kts" || base == "settings.gradle.kts"
		if isBuildScript {
			return includeBuildScripts
		}
		return includeKTS
	}
	return false
}

func officialDeclarationsToTopLevel(rawDecls []officialSnapshotDecl) []TopLevelDeclaration {
	out := make([]TopLevelDeclaration, 0, len(rawDecls))
	for _, decl := range rawDecls {
		kind := DeclarationKind(decl.Kind)
		if kind != DeclClass && kind != DeclInterface && kind != DeclObject && kind != DeclFunction && kind != DeclProperty && kind != DeclTypeAlias {
			continue
		}
		name := normalizeIdentifier(decl.Name)
		if name == "" {
			continue
		}
		out = append(out, TopLevelDeclaration{
			Kind: kind,
			Name: name,
			Line: 0,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Line < out[j].Line
	})
	return out
}

func officialDiagnostics(errorCount int, messages []string) []Diagnostic {
	if errorCount <= 0 {
		return nil
	}
	out := make([]Diagnostic, 0, maxInt(errorCount, len(messages)))
	for _, message := range messages {
		msg := strings.TrimSpace(message)
		if msg == "" {
			msg = "kotlin parse error"
		}
		out = append(out, Diagnostic{
			Path:     "",
			Line:     0,
			Column:   0,
			Message:  msg,
			Severity: SeverityError,
		})
	}
	for len(out) < errorCount {
		out = append(out, Diagnostic{
			Line:     0,
			Column:   0,
			Message:  "kotlin parse error",
			Severity: SeverityError,
		})
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func runCommand(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %q failed: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runCommandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command %q failed: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
