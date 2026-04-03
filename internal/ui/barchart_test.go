package ui

import (
	"strings"
	"testing"
)

func TestBrailleChartEmpty(t *testing.T) {
	result := BrailleChart(nil, 100, 20, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) != 20 {
			t.Errorf("line %d: expected 20 runes, got %d", i, len(runes))
		}
	}
}

func TestBrailleChartFixedWidth(t *testing.T) {
	result := BrailleChart([]float64{50, 100, 25, 75}, 100, 10, 3)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) != 10 {
			t.Errorf("line %d: expected 10 runes, got %d", i, len(runes))
		}
	}
}

func TestBrailleChartZeroMax(t *testing.T) {
	result := BrailleChart([]float64{50}, 0, 10, 5)
	if result == "" {
		t.Error("expected non-empty output")
	}
}

func TestBrailleChartAllZero(t *testing.T) {
	vals := []float64{0, 0, 0, 0, 0}
	result := BrailleChart(vals, 100, 10, 3)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestBrailleChartMaxValues(t *testing.T) {
	vals := []float64{100, 100, 100}
	result := BrailleChart(vals, 100, 10, 3)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}
