package anchor

import (
	"reflect"
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
