package logging

import (
	"fmt"
	"strings"
)

// formatCoreMetrics formats per-core percentages with consistent width
func formatCoreMetrics(percentages []float64) string {
	var builder strings.Builder
	for i, pct := range percentages {
		builder.WriteString(fmt.Sprintf("Core%d: %5.1f%%  ", i+1, pct))
	}
	return builder.String() + "\n"
}

// formatHistorical formats historical data in columns
func FormatHistorical(history [][]float64) string {
	if len(history) == 0 {
		return "No historical data"
	}

	var builder strings.Builder
	numCores := len(history[0])

	// Header
	builder.WriteString("\nLatest Metrics:\n")
	for i := 0; i < numCores; i++ {
		builder.WriteString(fmt.Sprintf("Core%d\t", i+1))
	}
	builder.WriteString("\n")

	// Data columns
	for _, snapshot := range history {
		for _, pct := range snapshot {
			builder.WriteString(fmt.Sprintf("%5.1f%%\t", pct))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

/*
func formatVertical(history [][]float64) string {
	if len(history) == 0 {
		return "No historical data"
	}

	var builder strings.Builder
	builder.WriteString("\nLatest Metrics:\n")

	for coreIdx := range history[0] {
		builder.WriteString(fmt.Sprintf("Core%d:\t", coreIdx+1))
		for _, snapshot := range history {
			if coreIdx < len(snapshot) {
				builder.WriteString(fmt.Sprintf("%5.1f%%\t", snapshot[coreIdx]))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
*/
