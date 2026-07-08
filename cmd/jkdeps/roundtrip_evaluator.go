package main

import "github.com/dh-kam/jkdeps/internal/mixedgraph"

const formatterRequiredComparisonReason = "formatter command is required for reliable rewritten-source comparison"
const normalizedDiffComparisonReason = "normalized source differs after header rewrite"

func evaluateRoundTripRewrite(file mixedgraph.FileUnit, original []byte, rewritten []byte, cfg roundTripCheckConfig) (roundTripEvaluation, error) {
	if cfg.formatter == nil || !cfg.formatter.HasFormatter(file.Language) {
		if isSemanticallyEquivalentHeaderRewrite(original, rewritten, file) {
			return roundTripEvaluation{Status: roundTripStatusPass}, nil
		}
		return roundTripEvaluation{
			Status: roundTripStatusUnsupported,
			Reason: formatterRequiredComparisonReason,
		}, nil
	}

	originalFormatted, err := cfg.formatter.Format(file.Language, file.Path, original)
	if err != nil {
		return roundTripEvaluation{
			Status: roundTripStatusFormatError,
			Reason: err.Error(),
		}, nil
	}
	rewrittenFormatted, err := cfg.formatter.Format(file.Language, file.Path, rewritten)
	if err != nil {
		return roundTripEvaluation{
			Status: roundTripStatusFormatError,
			Reason: err.Error(),
		}, nil
	}

	if normalizeRoundTripText(originalFormatted) == normalizeRoundTripText(rewrittenFormatted) {
		return roundTripEvaluation{Status: roundTripStatusPass}, nil
	}

	return roundTripEvaluation{
		Status: roundTripStatusDiff,
		Reason: normalizedDiffComparisonReason,
	}, nil
}
