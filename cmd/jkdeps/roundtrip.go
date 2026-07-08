package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

type roundTripStatus string

const (
	roundTripStatusPass        roundTripStatus = "pass"
	roundTripStatusDiff        roundTripStatus = "diff"
	roundTripStatusParseFailed roundTripStatus = "parse-failed"
	roundTripStatusUnsupported roundTripStatus = "unsupported"
	roundTripStatusFormatError roundTripStatus = "format-error"
)

type roundTripRewriteMode string

const (
	roundTripRewriteModeHeader   roundTripRewriteMode = "header"
	roundTripRewriteModeLossless roundTripRewriteMode = "lossless"
)

type roundTripCheckConfig struct {
	formatter   sourceFormatter
	rewriteMode roundTripRewriteMode
	limit       int
	printJSON   bool
}

type roundTripEvaluation struct {
	Status roundTripStatus
	Reason string
}

type roundTripFileResult struct {
	Path     string                    `json:"path"`
	Relative string                    `json:"relative_path"`
	Language mixedgraph.SourceLanguage `json:"language"`
	Status   roundTripStatus           `json:"status"`
	Reason   string                    `json:"reason,omitempty"`
}

type roundTripSummary struct {
	Root                 string                `json:"root"`
	TotalFiles           int                   `json:"total_files"`
	CheckedFiles         int                   `json:"checked_files"`
	PassedFiles          int                   `json:"passed_files"`
	DiffFiles            int                   `json:"diff_files"`
	ParseFailed          int                   `json:"parse_failed_files"`
	Unsupported          int                   `json:"unsupported_files"`
	FormatErrors         int                   `json:"format_error_files"`
	RewriteMode          roundTripRewriteMode  `json:"rewrite_mode"`
	ParseDurationSeconds float64               `json:"parse_duration_seconds"`
	DurationSeconds      float64               `json:"duration_seconds"`
	JavaFormatCmd        string                `json:"java_format_cmd,omitempty"`
	KotlinFormatCmd      string                `json:"kotlin_format_cmd,omitempty"`
	Files                []roundTripFileResult `json:"files,omitempty"`
}

func runRoundTripCheck(args []string) int {
	fs := cliutil.NewFlagSet("roundtrip-check", args)
	parseFlags := addMixedParseCommandFlags(fs)
	javaFormatCmd := fs.String("java-format-cmd", "", "Shell command used to format Java sources; use {file} placeholder for the temp file path")
	kotlinFormatCmd := fs.String("kotlin-format-cmd", "", "Shell command used to format Kotlin sources; use {file} placeholder for the temp file path")
	rewriteMode := fs.String("rewrite-mode", string(roundTripRewriteModeHeader), "Rewrite mode: header|lossless")
	limit := fs.Int("limit", 0, "Maximum number of parsed files to compare (0 means all)")
	printJSON := fs.Bool("json", false, "Print result JSON to stdout")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	mode := roundTripRewriteMode(strings.TrimSpace(*rewriteMode))
	if mode == "" {
		mode = roundTripRewriteModeHeader
	}
	if mode != roundTripRewriteModeHeader && mode != roundTripRewriteModeLossless {
		fmt.Fprintf(os.Stderr, "roundtrip-check failed: invalid rewrite mode %q\n", *rewriteMode)
		return 2
	}

	started := time.Now()
	repo, err := parseFlags.parseRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "roundtrip-check failed: %v\n", err)
		return 1
	}

	cfg := roundTripCheckConfig{
		formatter:   newCommandSourceFormatter(strings.TrimSpace(*javaFormatCmd), strings.TrimSpace(*kotlinFormatCmd)),
		rewriteMode: mode,
		limit:       *limit,
		printJSON:   *printJSON,
	}

	summary, err := runRoundTripRepositoryCheck(repo, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "roundtrip-check failed: %v\n", err)
		return 1
	}
	summary.RewriteMode = mode
	summary.ParseDurationSeconds = repo.Duration.Seconds()
	summary.DurationSeconds = time.Since(started).Seconds()

	if *printJSON {
		if err := json.NewEncoder(os.Stdout).Encode(summary); err != nil {
			fmt.Fprintf(os.Stderr, "encode roundtrip result: %v\n", err)
			return 1
		}
	} else {
		writeRoundTripSummary(os.Stdout, summary)
	}

	if summary.DiffFiles > 0 || summary.ParseFailed > 0 || summary.Unsupported > 0 || summary.FormatErrors > 0 {
		return 1
	}
	return 0
}

func runRoundTripRepositoryCheck(repo mixedgraph.RepositoryResult, cfg roundTripCheckConfig) (roundTripSummary, error) {
	files := append([]mixedgraph.FileUnit(nil), repo.Files...)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Relative < files[j].Relative
	})

	summary := roundTripSummary{
		Root:        repo.Root,
		TotalFiles:  len(files),
		RewriteMode: normalizedRoundTripRewriteMode(cfg.rewriteMode),
		Files:       make([]roundTripFileResult, 0, len(files)),
	}
	if cfg.formatter != nil {
		formatterSummary := cfg.formatter.Summary()
		summary.JavaFormatCmd = formatterSummary.JavaCommand
		summary.KotlinFormatCmd = formatterSummary.KotlinCommand
	}

	for _, file := range files {
		if cfg.limit > 0 && summary.CheckedFiles >= cfg.limit {
			break
		}
		result, err := roundTripCheckFile(file, cfg)
		if err != nil {
			return roundTripSummary{}, err
		}
		summary.Files = append(summary.Files, result)
		switch result.Status {
		case roundTripStatusPass:
			summary.CheckedFiles++
			summary.PassedFiles++
		case roundTripStatusDiff:
			summary.CheckedFiles++
			summary.DiffFiles++
		case roundTripStatusParseFailed:
			summary.ParseFailed++
		case roundTripStatusUnsupported:
			summary.Unsupported++
		case roundTripStatusFormatError:
			summary.CheckedFiles++
			summary.FormatErrors++
		}
	}

	return summary, nil
}

func roundTripCheckFile(file mixedgraph.FileUnit, cfg roundTripCheckConfig) (roundTripFileResult, error) {
	result := roundTripFileResult{
		Path:     file.Path,
		Relative: file.Relative,
		Language: file.Language,
	}

	if !file.Parsed {
		result.Status = roundTripStatusParseFailed
		result.Reason = "file did not parse successfully"
		return result, nil
	}

	original, err := os.ReadFile(file.Path)
	if err != nil {
		return roundTripFileResult{}, fmt.Errorf("read source %s: %w", file.Path, err)
	}

	if reason := unsupportedRoundTripReason(file.Language, original); reason != "" {
		result.Status = roundTripStatusUnsupported
		result.Reason = reason
		return result, nil
	}

	var rewritten []byte
	var changed bool
	switch normalizedRoundTripRewriteMode(cfg.rewriteMode) {
	case roundTripRewriteModeLossless:
		rewritten = append([]byte(nil), original...)
		changed = false
	default:
		rewritten, changed = rewriteSourceHeader(original, file)
	}
	if !changed && (cfg.formatter == nil || !cfg.formatter.HasFormatter(file.Language)) {
		result.Status = roundTripStatusPass
		return result, nil
	}
	evaluation, err := evaluateRoundTripRewrite(file, original, rewritten, cfg)
	if err != nil {
		return roundTripFileResult{}, err
	}
	result.Status = evaluation.Status
	result.Reason = evaluation.Reason
	return result, nil
}

func normalizedRoundTripRewriteMode(mode roundTripRewriteMode) roundTripRewriteMode {
	if mode == roundTripRewriteModeLossless {
		return roundTripRewriteModeLossless
	}
	return roundTripRewriteModeHeader
}

func unsupportedRoundTripReason(lang mixedgraph.SourceLanguage, source []byte) string {
	return ""
}

func normalizeRoundTripText(source []byte) string {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.Join(lines, "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return text + "\n"
}

func writeRoundTripSummary(w io.Writer, summary roundTripSummary) {
	cliutil.WriteSummaryLine(w, "Root", "%s", summary.Root)
	cliutil.WriteSummaryLine(w, "Files", "total=%d checked=%d", summary.TotalFiles, summary.CheckedFiles)
	cliutil.WriteSummaryLine(w, "Rewrite", "%s", normalizedRoundTripRewriteMode(summary.RewriteMode))
	if summary.ParseDurationSeconds > 0 || summary.DurationSeconds > 0 {
		cliutil.WriteSummaryLine(w, "Duration", "parse=%.2fs total=%.2fs", summary.ParseDurationSeconds, summary.DurationSeconds)
	}
	if summary.JavaFormatCmd != "" || summary.KotlinFormatCmd != "" {
		javaCmd := summary.JavaFormatCmd
		if javaCmd == "" {
			javaCmd = "not-configured"
		}
		kotlinCmd := summary.KotlinFormatCmd
		if kotlinCmd == "" {
			kotlinCmd = "not-configured"
		}
		cliutil.WriteSummaryLine(w, "Formatter", "java=%s kotlin=%s", javaCmd, kotlinCmd)
	}
	cliutil.WriteSummaryLine(
		w,
		"RoundTrip",
		"pass=%d diff=%d parse-failed=%d unsupported=%d format-error=%d",
		summary.PassedFiles,
		summary.DiffFiles,
		summary.ParseFailed,
		summary.Unsupported,
		summary.FormatErrors,
	)
	if len(summary.Files) == 0 {
		return
	}

	cliutil.WriteSectionHeader(w, "Results")
	for _, file := range summary.Files {
		if file.Status == roundTripStatusPass {
			continue
		}
		if file.Reason != "" {
			fmt.Fprintf(w, "- [%s] %s: %s\n", file.Status, file.Relative, file.Reason)
			continue
		}
		fmt.Fprintf(w, "- [%s] %s\n", file.Status, file.Relative)
	}
}
