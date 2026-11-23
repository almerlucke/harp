package main

import "testing"

func TestParseTime(t *testing.T) {
	for text, want := range map[string]float64{"": 0, "0": 0, "264": 264, "12.5": 12.5, "4:24": 264, "0:30": 30, "1:02:03": 3723} {
		got, err := parseTime(text)
		if err != nil || got != want {
			t.Errorf("%q: %v, %v (want %v)", text, got, err, want)
		}
	}
	for _, text := range []string{"abc", "4:", ":30", "-5", "4:2x", "NaN", "Inf"} {
		if _, err := parseTime(text); err == nil {
			t.Errorf("%q is not a time and was taken", text)
		}
	}
}
