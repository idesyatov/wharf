package ui

// Sparkline renders a mini sparkline chart from a slice of values.
func Sparkline(values []float64, maxVal float64) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	if maxVal <= 0 {
		maxVal = 1
	}
	result := make([]rune, len(values))
	for i, v := range values {
		idx := int(v / maxVal * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		result[i] = bars[idx]
	}
	return string(result)
}
