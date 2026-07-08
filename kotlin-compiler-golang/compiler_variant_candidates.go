package kotlincompilergolang

import "bytes"

func buildRuleVariantCandidates(source []byte) [][]byte {
	candidates := make([][]byte, 0, 12)
	appendCandidate := func(candidate []byte) {
		if len(candidate) == 0 || bytes.Equal(candidate, source) {
			return
		}
		for _, existing := range candidates {
			if bytes.Equal(existing, candidate) {
				return
			}
		}
		candidates = append(candidates, candidate)
	}

	appendCandidate(normalizeParenthesizedLambdaBodies(source))
	appendCandidate(normalizeObjectLiteralExpressions(source))

	trailingAdjusted := normalizeKnownTrailingLambdaCalls(source)
	appendCandidate(trailingAdjusted)
	if !bytes.Equal(trailingAdjusted, source) {
		appendCandidate(normalizeParenthesizedLambdaBodies(trailingAdjusted))
	}

	genericTrailingAdjusted := normalizeGenericTrailingLambdaCalls(source)
	appendCandidate(genericTrailingAdjusted)
	if !bytes.Equal(genericTrailingAdjusted, source) {
		appendCandidate(normalizeParenthesizedLambdaBodies(genericTrailingAdjusted))
	}

	genericSimpleRunTestTrailingAdjusted := normalizeGenericTrailingLambdaCallsSimpleRunTest(source)
	appendCandidate(genericSimpleRunTestTrailingAdjusted)
	if !bytes.Equal(genericSimpleRunTestTrailingAdjusted, source) {
		appendCandidate(normalizeParenthesizedLambdaBodies(genericSimpleRunTestTrailingAdjusted))
	}

	genericNoRunTestTrailingAdjusted := normalizeGenericTrailingLambdaCallsWithoutRunTest(source)
	appendCandidate(genericNoRunTestTrailingAdjusted)
	if !bytes.Equal(genericNoRunTestTrailingAdjusted, source) {
		appendCandidate(normalizeParenthesizedLambdaBodies(genericNoRunTestTrailingAdjusted))
	}

	delegationAndUnsignedAdjusted := normalizeDelegationConstructorsAndUnsignedLiterals(source)
	appendCandidate(delegationAndUnsignedAdjusted)
	if !bytes.Equal(delegationAndUnsignedAdjusted, source) {
		genericAfterDelegation := normalizeGenericTrailingLambdaCalls(delegationAndUnsignedAdjusted)
		appendCandidate(genericAfterDelegation)
		appendCandidate(normalizeParenthesizedLambdaBodies(genericAfterDelegation))

		genericSimpleRunTestAfterDelegation := normalizeGenericTrailingLambdaCallsSimpleRunTest(delegationAndUnsignedAdjusted)
		appendCandidate(genericSimpleRunTestAfterDelegation)
		appendCandidate(normalizeParenthesizedLambdaBodies(genericSimpleRunTestAfterDelegation))

		genericNoRunTestAfterDelegation := normalizeGenericTrailingLambdaCallsWithoutRunTest(delegationAndUnsignedAdjusted)
		appendCandidate(genericNoRunTestAfterDelegation)
		appendCandidate(normalizeParenthesizedLambdaBodies(genericNoRunTestAfterDelegation))
	}

	return candidates
}
