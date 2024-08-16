package anchor

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	expected := []string{"curl", "wget"}
	command := "apt-get install -y curl    wget"
	actual := parseCommand(command)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestAppendPackageVersions(t *testing.T) {
	file := `# hadolint ignore=DL3008
RUN apt-get update \
  && apt-get install \
    --no-install-recommends -y \
    # We just need curl and wget
    curl wget \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean`
	input := strings.NewReader(file)
	nodes := Parse(input)
	architecture := "amd64"

	packageMap := map[string]string{
		"curl": "7.68.0",
		"wget": "1.20.3",
	}

	expected := fmt.Sprintf(`# hadolint ignore=DL3008
RUN dpkg --add-architecture %s && apt-get update && apt-get update \
  && apt-get install \
    --no-install-recommends -y \
    # We just need curl and wget
    curl=%s wget=%s \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean
`, architecture, packageMap["curl"], packageMap["wget"])

	node := nodes[0]
	appendPackageVersions(&node, packageMap, architecture)
	nodes[0] = node

	w := &strings.Builder{}
	nodes.Write(w)
	result := w.String()
	if result != expected {
		t.Errorf("Expected:\n%v\ngot:\n%v", expected, result)
	}
}

func TestParseComment(t *testing.T) {
	cases := []struct {
		name            string
		entry           Entry
		expectedIgnored []string
		expectedAll     bool
	}{
		{
			name: "simple comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# anchor ignore=curl,wget",
			},
			expectedIgnored: []string{"curl", "wget"},
			expectedAll:     false,
		},
		{
			name: "poorly formatted comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "#anchor ignore =curl,     test,wget",
			},
			expectedIgnored: []string{"curl", "test", "wget"},
			expectedAll:     false,
		},
		{
			name: "non anchor comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# hadolint ignore=DL3008",
			},
			expectedIgnored: []string{},
			expectedAll:     false,
		},
		{
			name: "non anchor ignore comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# anchor is a tool for anchoring dependencies in dockerfiles",
			},
			expectedIgnored: []string{},
			expectedAll:     false,
		},
		{
			name: "basic ignore all",
			entry: Entry{
				Type:  EntryComment,
				Value: "# anchor ignore",
			},
			expectedIgnored: []string{},
			expectedAll:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, all := parseComment(tc.entry)
			if !reflect.DeepEqual(actual, tc.expectedIgnored) {
				t.Errorf("Expected %v but got %v", tc.expectedIgnored, actual)
			}
			if all != tc.expectedAll {
				t.Errorf("Expected %v but got %v", tc.expectedAll, all)
			}
		})
	}
}

func TestAppendPackageVersionsWithIgnore(t *testing.T) {
	file := `# hadolint ignore=DL3008
# anchor ignore=curl
RUN apt-get update \
  && apt-get install \
    --no-install-recommends -y \
    # We just need curl and wget
    curl wget \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean`
	input := strings.NewReader(file)
	nodes := Parse(input)
	architecture := "amd64"

	packageMap := map[string]string{
		"curl": "7.68.0",
		"wget": "1.20.3",
	}

	expected := fmt.Sprintf(`# hadolint ignore=DL3008
# anchor ignore=curl
RUN dpkg --add-architecture %s && apt-get update && apt-get update \
  && apt-get install \
    --no-install-recommends -y \
    # We just need curl and wget
    curl wget=%s \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean
`, architecture, packageMap["wget"])

	node := nodes[0]
	appendPackageVersions(&node, packageMap, architecture)
	nodes[0] = node

	w := &strings.Builder{}
	nodes.Write(w)
	result := w.String()
	if result != expected {
		t.Errorf("Expected:\n%v\ngot:\n%v", expected, result)
	}
}

func TestImageIgnore(t *testing.T) {
	file := `# hadolint ignore=DL3008
  # anchor ignore=golang:1.23-bookworm
FROM golang:1.23-bookworm as builder
`

	input := strings.NewReader(file)
	nodes := Parse(input)
	image, err := processFromCommand(&nodes[0])
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
	if image != "golang:1.23-bookworm" {
		t.Errorf("Expected golang:1.23-bookworm but got %v", image)
	}
}
