package anchor

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"
)

type commandType int

const (
	CommandFrom commandType = iota
	CommandRun
	CommandOther
)

type Node struct {
	startLine        int
	endLine          int
	Entries          []string
	commentIndexes   []int
	CommandType      commandType
	Command          string
	nextCommentIndex int
}

func (n Node) NextCommentIndex() int {
	if len(n.commentIndexes) < n.nextCommentIndex+1 {
		return -1
	}
	n.nextCommentIndex += 1
	return n.commentIndexes[n.nextCommentIndex-1]
}

func (n Node) Write(w io.Writer) {
	w.Write([]byte(strings.Join(n.Entries, "")))
}

func (n *Node) appendLine(line []byte, lineNumber int) {
	line = append(line, '\n')
	n.Entries = append(n.Entries, string(line))
	if n.startLine == -1 {
		n.startLine = lineNumber
	}
}

func (n *Node) appendComment(comment []byte, lineNumber int) {
	n.appendLine(comment, lineNumber)
	n.commentIndexes = append(n.commentIndexes, len(n.Entries)-1)
	if n.startLine == -1 {
		n.startLine = lineNumber
	}
}

func (n *Node) appendCommand(command []byte, lineNumber int) {
	trimmed := bytes.TrimLeftFunc(command, func(r rune) bool {
		return r == '\\'
	})
	n.Command += string(trimmed)
	if n.startLine == -1 {
		n.startLine = lineNumber
	}
}

func Parse(r io.Reader) ([]Node, error) {
	scanner := bufio.NewScanner(r)
	currentLine := 1
	node := Node{startLine: -1}
	nodes := make([]Node, 0)
	for scanner.Scan() {
		line := scanner.Bytes()

		if isComment(line) {
			node.appendComment(line, currentLine)
			currentLine++
			continue
		}

		if isWhitespace(line) {
			// node.appendLine(line, currentLine)
			currentLine++
			continue
		}

		node.appendLine(line, currentLine)
		node.appendCommand(line, currentLine)
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
				// node.appendLine(nextLine, currentLine)
				currentLine++
				continue
			}
			if isComment(nextLine) {
				node.appendComment(nextLine, currentLine)
				currentLine++
				continue
			}
			node.appendLine(nextLine, currentLine)
			node.appendCommand(nextLine, currentLine)

			isEndOfLine = isEndOfSection(nextLine)
			currentLine++
		}

		node.endLine = currentLine
		nodes = append(nodes, node)
		currentLine++
		node = Node{startLine: -1}
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
