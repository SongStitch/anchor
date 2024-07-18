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
		name     string
		entry    Entry
		expected []string
	}{
		{
			name: "simple comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# anchor ignore=curl,wget",
			},
			expected: []string{"curl", "wget"},
		},
		{
			name: "poorly formatted comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "#anchor ignore =curl,     test,wget",
			},
			expected: []string{"curl", "test", "wget"},
		},
		{
			name: "non anchor comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# hadolint ignore=DL3008",
			},
			expected: []string{},
		},
		{
			name: "non anchor ignore comment",
			entry: Entry{
				Type:  EntryComment,
				Value: "# anchor is a tool for anchoring dependencies in dockerfiles",
			},
			expected: []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseComment(tc.entry)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Expected %v but got %v", tc.expected, actual)
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
