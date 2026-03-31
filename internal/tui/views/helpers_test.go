package views

import (
	"testing"
	"time"
)

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0.0%"},
		{0.5, "0.5%"},
		{9.9, "9.9%"},
		{10.0, "10%"},
		{15.3, "15%"},
		{99.9, "100%"},
		{100.0, "100%"},
	}
	for _, tt := range tests {
		got := formatCPU(tt.input)
		if got != tt.want {
			t.Errorf("formatCPU(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1Ki"},
		{1024 * 1024, "1Mi"},
		{128 * 1024 * 1024, "128Mi"},
		{1024 * 1024 * 1024, "1.0Gi"},
		{2 * 1024 * 1024 * 1024, "2.0Gi"},
	}
	for _, tt := range tests {
		got := formatMemory(tt.input)
		if got != tt.want {
			t.Errorf("formatMemory(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"ab", 2, "ab"},
		{"abc", 2, "a…"},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	got := padRight("hi", 5)
	if len(got) != 5 {
		t.Errorf("padRight('hi', 5) len = %d, want 5", len(got))
	}
	if got != "hi   " {
		t.Errorf("padRight('hi', 5) = %q, want %q", got, "hi   ")
	}
}

func TestPadRight_AlreadyWide(t *testing.T) {
	got := padRight("hello", 3)
	if got != "hello" {
		t.Errorf("padRight should not truncate: got %q", got)
	}
}

func TestTimeAgo(t *testing.T) {
	tests := []struct {
		offset time.Duration
		want   string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5 minutes ago"},
		{1 * time.Minute, "1 minute ago"},
		{2 * time.Hour, "2 hours ago"},
		{1 * time.Hour, "1 hour ago"},
		{3 * 24 * time.Hour, "3 days ago"},
		{1 * 24 * time.Hour, "1 day ago"},
		{60 * 24 * time.Hour, "2 months ago"},
		{400 * 24 * time.Hour, "1 year ago"},
	}
	for _, tt := range tests {
		tm := time.Now().Add(-tt.offset)
		got := timeAgo(tm)
		if got != tt.want {
			t.Errorf("timeAgo(-%v) = %q, want %q", tt.offset, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1Ki"},
		{1024 * 1024, "1Mi"},
		{1024 * 1024 * 1024, "1.0Gi"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
