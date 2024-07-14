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

func TestParser(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected Nodes
	}{
		{"Simple from node",
			"FROM ubuntu:20.04\n",
			Nodes{
				{
					CommandType: CommandFrom,
					Command:     "FROM ubuntu:20.04",
					Entries: []Entry{{
						Type:      EntryCommand,
						Value:     "FROM ubuntu:20.04\n",
						Beginning: true,
					}},
				},
			},
		},
		{
			"Simple multiline node",
			`# Test Dockerfile for anchor

FROM golang:1.22-bookworm as builder`,
			Nodes{
				{
					CommandType: CommandFrom,
					Command:     "FROM golang:1.22-bookworm as builder",
					Entries: []Entry{
						{
							Type:  EntryComment,
							Value: "# Test Dockerfile for anchor\n",
						},
						{
							Type:  EntryEmpty,
							Value: "\n",
						},
						{
							Type:      EntryCommand,
							Value:     "FROM golang:1.22-bookworm as builder\n",
							Beginning: true,
						},
					},
				},
			},
		},
		{
			"Multiple nodes",
			`FROM golang:1.22-bookworm
# hadolint ignore=DL3008
RUN apt-get update \
  && apt-get install \
    --no-install-recommends -y \
    # We just need curl and wget
    curl wget \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean`,
			Nodes{
				{
					CommandType: CommandFrom,
					Command:     "FROM golang:1.22-bookworm",
					Entries: []Entry{
						{
							Type:      EntryCommand,
							Value:     "FROM golang:1.22-bookworm\n",
							Beginning: true,
						},
					},
				},
				{
					CommandType: CommandRun,
					Command:     "apt-get update \\  && apt-get install \\    --no-install-recommends -y \\    curl wget \\  && rm -rf /var/lib/apt/lists/* \\  && apt-get clean",
					Entries: []Entry{
						{
							Type:  EntryComment,
							Value: "# hadolint ignore=DL3008\n",
						},
						{
							Type:      EntryCommand,
							Value:     "RUN apt-get update \\\n",
							Beginning: true,
						},
						{
							Type:  EntryCommand,
							Value: "  && apt-get install \\\n",
						},
						{
							Type:  EntryCommand,
							Value: "    --no-install-recommends -y \\\n",
						},
						{
							Type:  EntryComment,
							Value: "    # We just need curl and wget\n",
						},
						{
							Type:  EntryCommand,
							Value: "    curl wget \\\n",
						},
						{
							Type:  EntryCommand,
							Value: "  && rm -rf /var/lib/apt/lists/* \\\n",
						},
						{
							Type:  EntryCommand,
							Value: "  && apt-get clean\n",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			input := strings.NewReader(tc.input)
			result := Parse(input)
			if len(result) != len(tc.expected) {
				t.Fatalf("%s: Different lengths, expected:\n%v\n got:\n%v", tc.name, tc.expected, result)
			}
			for i := range result {
				if tc.expected[i].Command != result[i].Command {
					t.Errorf(
						"%s: Different command with index %d, expected:\n%v\ngot:\n%v",
						tc.name,
						i,
						tc.expected[i].Command,
						result[i].Command,
					)
				}
				if tc.expected[i].CommandType != result[i].CommandType {
					t.Errorf(
						"%s: Different command type for index %d, expected:\n%v\ngot:\n%v",
						tc.name,
						i,
						tc.expected[i],
						result[i],
					)
				}
				for j := range tc.expected[i].Entries {
					if !reflect.DeepEqual(tc.expected[i].Entries[j], result[i].Entries[j]) {
						t.Errorf(
							"%s: Different entries for node %d, entry: %d, expected:\n%v\ngot:\n%v",
							tc.name,
							i,
							j,
							tc.expected[i].Entries[j],
							result[i].Entries[j],
						)
					}
				}
			}
		})
	}
}
