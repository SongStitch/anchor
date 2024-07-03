package anchor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode"
)

type command string

const (
	CommandFrom command = "FROM"
	CommandRun  command = "RUN"
)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

type Node struct {
	startLine     int
	endLine       int
	value         string
	comments      []string
	dockerCommand string
}

func (n Node) Write(w io.Writer) {
	w.Write([]byte(n.value))
}

func Parse(r io.Reader) ([]Node, error) {
	scanner := bufio.NewScanner(r)
	currentLine := 0
	currentNode := Node{startLine: 0}
	nodes := make([]Node, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		if currentLine == 0 {
			line = bytes.TrimPrefix(line, utf8bom)
		}

		if isComment(line) {
			currentNode.value += string(line)
			currentNode.comments = append(currentNode.comments, string(line))
			currentLine++
			continue
		}

		if isWhitespace(line) {
			currentNode.value += string(line)
			currentLine++
			continue
		}

		currentNode.value += string(line)
		if bytes.HasPrefix(line, []byte("FROM")) {
			currentNode.dockerCommand = "FROM"
		} else if bytes.HasPrefix(line, []byte("RUN")) {
			currentNode.dockerCommand = "RUN"
		}

		isEndOfLine := isEndOfSection(line)
		for !isEndOfLine && scanner.Scan() {
			nextLine := scanner.Bytes()
			if isWhitespace(nextLine) {
				currentNode.value += string(nextLine)
				currentLine++
				continue
			}
			currentNode.value += string(nextLine)

			isEndOfLine = isEndOfSection(nextLine)
			currentLine++
		}

		currentNode.endLine = currentLine
		nodes = append(nodes, currentNode)
		currentLine++
		currentNode = Node{startLine: currentLine}
	}
	fmt.Println(nodes)
	return nodes, nil
}

func isWhitespace(line []byte) bool {
	return len(bytes.TrimSpace(line)) == 0
}

func isEndOfSection(line []byte) bool {
	fmt.Println(string(line))
	trimmed := bytes.TrimRightFunc(line, unicode.IsSpace)
	return trimmed[len(trimmed)-1] != '\\'
}

func isComment(line []byte) bool {
	trimmed := bytes.TrimLeftFunc(line, unicode.IsSpace)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '#'
}
