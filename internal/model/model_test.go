package model

import "testing"

func TestSeverityNumberFromText(t *testing.T) {
	cases := []struct {
		text string
		want int32
	}{
		{"TRACE", 1},
		{"DEBUG", 5},
		{"INFO", 9},
		{"WARN", 13},
		{"WARNING", 13},
		{"ERROR", 17},
		{"FATAL", 21},
		{"error", 17}, // case-insensitive
		{" Info ", 9}, // whitespace-tolerant
		{"", 0},       // empty
		{"MEH", 0},    // unknown
	}
	for _, c := range cases {
		if got := SeverityNumberFromText(c.text); got != c.want {
			t.Errorf("SeverityNumberFromText(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}
