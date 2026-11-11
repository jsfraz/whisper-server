package utils

import "testing"

func TestSanitizeLogValue(t *testing.T) {
	cases := map[string]string{
		"plain":          "plain",
		"line\nbreak":    "linebreak",
		"carriage\rhere": "carriagehere",
		"both\r\n":       "both",
	}
	for input, want := range cases {
		if got := SanitizeLogValue(input); got != want {
			t.Errorf("SanitizeLogValue(%q) = %q, want %q", input, got, want)
		}
	}
}
