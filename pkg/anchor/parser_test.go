package anchor

import (
	"testing"
)

func TestIsEndOfSection(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"FROM ubuntu:20.04 \n", true},
		{"RUN apt-get update \\ \n", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := isEndOfSection([]byte(tc.input))
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		},
		)
	}
}
