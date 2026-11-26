package database

import (
	"jsfraz/shept-server/utils"
	"testing"
)

func TestEscapeLike(t *testing.T) {
	tests := map[string]string{
		"plain":         "plain",
		"100%":          `100\%`,
		"a_b":           `a\_b`,
		`a\b`:           `a\\b`,
		`50%_off\today`: `50\%\_off\\today`,
	}
	for input, expected := range tests {
		if actual := escapeLike(input); actual != expected {
			t.Errorf("escapeLike(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSearchExcludesSelfOnlyInRelease(t *testing.T) {
	previous := utils.GetSingleton().Config.GinMode
	defer func() { utils.GetSingleton().Config.GinMode = previous }()

	utils.GetSingleton().Config.GinMode = "debug"
	if searchExcludesSelf() {
		t.Fatal("debug search must include the current user")
	}

	utils.GetSingleton().Config.GinMode = "release"
	if !searchExcludesSelf() {
		t.Fatal("release search must exclude the current user")
	}
}
