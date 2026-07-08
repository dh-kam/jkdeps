package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/parser"
)

func parseGroupByFlag(value string) (mixedgraph.GroupBy, error) {
	group := mixedgraph.GroupBy(strings.TrimSpace(value))
	if !group.IsValid() {
		return "", fmt.Errorf("invalid --group-by value %q (expected package|dir)", value)
	}
	return group, nil
}

func parseMixedRepository(flags mixedParseCommandFlags) (mixedgraph.RepositoryResult, error) {
	return mixedgraph.ParseRepository(flags.repo(), flags.parseOptions())
}

func writeMixedRepositorySummary(w io.Writer, result mixedgraph.RepositoryResult, grammar parser.JavaGrammar, includeSuccessRate bool) {
	cliutil.WriteSummaryLine(w, "Root", "%s", result.Root)
	if grammar != "" {
		cliutil.WriteSummaryLine(w, "Java Grammar", "%s", grammar)
	}
	cliutil.WriteSummaryLine(w, "Files", "total=%d java=%d kotlin=%d", result.TotalFiles, result.JavaFiles, result.KotlinFiles)
	if includeSuccessRate {
		successRate := 100.0
		if result.TotalFiles > 0 {
			successRate = (float64(result.ParsedFiles) / float64(result.TotalFiles)) * 100
		}
		cliutil.WriteSummaryLine(w, "Parse", "parsed=%d failed=%d success=%.2f%%", result.ParsedFiles, result.FailedFiles, successRate)
	} else {
		cliutil.WriteSummaryLine(w, "Parse", "parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	}
	cliutil.WriteSummaryLine(w, "Duration", "%s", result.Duration.Round(0))
}

func writeSlowParseFilesSummary(w io.Writer, result mixedgraph.RepositoryResult, limit int) {
	files := result.SlowestFiles(limit)
	if len(files) == 0 {
		return
	}

	cliutil.WriteSectionHeader(w, fmt.Sprintf("Slow Parse Files (top %d)", len(files)))
	for _, file := range files {
		path := file.Relative
		if path == "" {
			path = file.Path
		}
		status := "parsed"
		if !file.Parsed {
			status = "failed"
		}
		fmt.Fprintf(w, "- %s %s %s\n", file.Duration.Round(0), status, path)
	}
}
