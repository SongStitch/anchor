package anchor

import (
	"reflect"
	"strings"
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

func TestBasicParse(t *testing.T) {
	input := `# Test Dockerfile for anchor
FROM golang:1.22-bookworm as builder

# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install --no-install-recommends -y curl wget \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

FROM golang:1.22-bookworm
# anchor ignore=curl
# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install --no-install-recommends -y curl wget \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
`

	nodes, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Errorf("Error parsing Dockerfile: %v", err)
	}

	expected := []Node{
		{
			startLine:      1,
			endLine:        2,
			Entries:        []string{"# Test Dockerfile for anchor\n", "FROM golang:1.22-bookworm as builder\n"},
			commentIndexes: []int{0},
			Command:        "FROM golang:1.22-bookworm as builder",
			CommandType:    CommandFrom,
		},
		{
			startLine:      4,
			endLine:        8,
			commentIndexes: []int{0},
			Entries: []string{
				"# hadolint ignore=DL3008\n", "RUN apt-get update \\\n", "    && apt-get install --no-install-recommends -y curl wget \\\n", "    && rm -rf /var/lib/apt/lists/* \\\n", "    && apt-get clean\n",
			},
			Command:     "RUN apt-get update \\    && apt-get install --no-install-recommends -y curl wget \\    && rm -rf /var/lib/apt/lists/* \\    && apt-get clean",
			CommandType: CommandRun,
		},
		{
			startLine:      10,
			endLine:        11,
			commentIndexes: []int{},
			Entries:        []string{"FROM golang:1.22-bookworm\n"},
			Command:        "FROM golang:1.22-bookworm",
			CommandType:    CommandFrom,
		},
		{
			startLine:      13,
			endLine:        18,
			commentIndexes: []int{0, 1},
			Entries: []string{
				"# hadolint ignore=DL3008\n", "RUN apt-get update \\    && apt-get install --no-install-recommends -y curl wget \\    && rm -rf /var/lib/apt/lists/* \\    && apt-get clean\n",
			},
			Command:     "RUN apt-get update \\    && apt-get install --no-install-recommends -y curl wget \\    && rm -rf /var/lib/apt/lists/* \\    && apt-get clean",
			CommandType: CommandRun,
		},
	}

	if len(nodes) != len(expected) {
		t.Errorf("Expected %d nodes, got %d", len(expected), len(nodes))
	}

	for i := range expected {
		if !reflect.DeepEqual(expected[i], nodes[i]) {
			t.Errorf("Expected %v, got %v", expected[i], nodes[i])
		}
	}
}
