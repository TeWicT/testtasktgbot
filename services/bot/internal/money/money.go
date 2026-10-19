package money

import (
	"fmt"
	"math"
)

// RublesToKopecks converts a ruble amount to integer kopecks.
func RublesToKopecks(rubles float64) (int64, error) {
	if math.IsNaN(rubles) || math.IsInf(rubles, 0) {
		return 0, fmt.Errorf("invalid amount")
	}
	kopecks := int64(math.Round(rubles * 100))
	if kopecks <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}
	return kopecks, nil
}

// KopecksToRubles formats kopecks as a ruble float for display (LLM/UI helpers).
func KopecksToRubles(kopecks int64) float64 {
	return float64(kopecks) / 100
}
