package anchor

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
)

type commandType int

const (
	CommandFrom commandType = iota
	CommandRun
	CommandOther
)

type Node struct {
	startLine   int
	endLine     int
	value       []byte
	comments    []string
	CommandType commandType
	Command     string
}

func (n Node) Write(w io.Writer) {
	w.Write([]byte(n.value))
}

func (n *Node) appendLine(line []byte) {
	n.value = append(n.value, line...)
	n.value = append(n.value, '\n')
}

func (n *Node) appendComment(line []byte) {
	n.comments = append(n.comments, string(line))
}

func (n *Node) appendCommand(command []byte) {
	trimmed := bytes.TrimLeftFunc(command, func(r rune) bool {
		return r == '\\'
	})
	n.Command += string(trimmed)
}

func Parse(r io.Reader) ([]Node, error) {
	scanner := bufio.NewScanner(r)
	currentLine := 0
	node := Node{startLine: 0}
	nodes := make([]Node, 0)
	for scanner.Scan() {
		line := scanner.Bytes()

		if isComment(line) {
			node.appendLine(line)
			node.appendComment(line)
			currentLine++
			continue
		}

		if isWhitespace(line) {
			node.appendLine(line)
			currentLine++
			continue
		}

		node.appendLine(line)
		node.appendCommand(line)
		if bytes.HasPrefix(line, []byte("FROM")) {
			node.CommandType = CommandFrom
		} else if bytes.HasPrefix(line, []byte("RUN")) {
			node.CommandType = CommandRun
		} else {
			node.CommandType = CommandOther
		}

		isEndOfLine := isEndOfSection(line)
		for !isEndOfLine && scanner.Scan() {
			nextLine := scanner.Bytes()
			if isWhitespace(nextLine) {
				node.appendLine(nextLine)
				currentLine++
				continue
			}
			if isComment(nextLine) {
				node.appendLine(nextLine)
				currentLine++
				continue
			}
			node.appendLine(nextLine)
			node.appendCommand(nextLine)

			isEndOfLine = isEndOfSection(nextLine)
			currentLine++
		}

		node.endLine = currentLine
		nodes = append(nodes, node)
		currentLine++
		node = Node{startLine: currentLine}
	}
	return nodes, nil
}

func isWhitespace(line []byte) bool {
	return len(bytes.TrimSpace(line)) == 0
}

func isEndOfSection(line []byte) bool {
	trimmed := bytes.TrimRightFunc(line, unicode.IsSpace)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[len(trimmed)-1] != '\\'
}

func isComment(line []byte) bool {
	trimmed := bytes.TrimLeftFunc(line, unicode.IsSpace)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '#'
}
