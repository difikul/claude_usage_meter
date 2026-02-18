package main

import "strings"

// getPrices returns (input, output, cacheRead, cacheCreate) prices per 1M tokens
func getPrices(model string) (float64, float64, float64, float64) {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		return 15.0, 75.0, 1.50, 18.75
	case strings.Contains(m, "haiku"):
		return 0.80, 4.0, 0.08, 1.0
	default:
		// Default to Sonnet pricing (most common in Claude Code)
		return 3.0, 15.0, 0.30, 3.75
	}
}

func calculateCost(model string, inputTokens, outputTokens, cacheReadTokens, cacheCreateTokens uint64) float64 {
	inputPrice, outputPrice, cacheReadPrice, cacheCreatePrice := getPrices(model)
	return (float64(inputTokens)*inputPrice +
		float64(outputTokens)*outputPrice +
		float64(cacheReadTokens)*cacheReadPrice +
		float64(cacheCreateTokens)*cacheCreatePrice) / 1_000_000.0
}
