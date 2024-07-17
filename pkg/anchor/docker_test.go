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
