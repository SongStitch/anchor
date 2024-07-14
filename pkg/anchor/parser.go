package anchor

import (
	"bufio"
	"bytes"
	"fmt"
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

type EntryType int

const (
	EntryCommand EntryType = iota
	EntryComment
	EntryEmpty
)

type Entry struct {
	Type      EntryType
	Value     string
	Beginning bool
}

type Nodes []Node

type Node struct {
	Entries     []Entry
	CommandType commandType
	Command     string
}

func (n Nodes) Print() {
	for _, node := range n {
		for _, entry := range node.Entries {
			fmt.Println(entry.Value)
		}
	}
}

func (n Nodes) Write(w io.Writer) {
	for _, node := range n {
		node.Write(w)
	}
}

func (n Node) Write(w io.Writer) {
	b := []byte{}
	for _, entry := range n.Entries {
		b = append(b, []byte(entry.Value)...)
	}
	w.Write(b)
}

func (n *Node) appendLine(line []byte, entryType EntryType, beginning bool, strip ...string) {
	if entryType == EntryCommand {
		l := string(line)
		for _, s := range strip {
			l = strings.TrimLeft(l, s)
			l = strings.TrimSpace(l)
		}
		n.Command += l
	}

	// new lines are trimmed by the scanner so we re-add them here
	line = append(line, '\n')
	n.Entries = append(n.Entries, Entry{Type: entryType, Value: string(line), Beginning: beginning})
}

func Parse(r io.Reader) Nodes {
	scanner := bufio.NewScanner(r)
	node := Node{}
	nodes := make([]Node, 0)
	for scanner.Scan() {
		line := scanner.Bytes()

		if isComment(line) {
			node.appendLine(line, EntryComment, false)
			continue
		}

		if isWhitespace(line) {
			node.appendLine(line, EntryEmpty, false)
			continue
		}
		if bytes.HasPrefix(line, []byte("FROM")) {
			node.appendLine(line, EntryCommand, true)
			node.CommandType = CommandFrom
		} else if bytes.HasPrefix(line, []byte("RUN")) {
			node.appendLine(line, EntryCommand, true, "RUN")
			node.CommandType = CommandRun
		} else {
			node.appendLine(line, EntryCommand, true)
			node.CommandType = CommandOther
		}

		isEndOfLine := isEndOfSection(line)
		for !isEndOfLine && scanner.Scan() {
			nextLine := scanner.Bytes()
			if isWhitespace(nextLine) {
				node.appendLine(nextLine, EntryEmpty, false)
				continue
			}
			if isComment(nextLine) {
				node.appendLine(nextLine, EntryComment, false)
				continue
			}
			node.appendLine(nextLine, EntryCommand, false)

			isEndOfLine = isEndOfSection(nextLine)
		}

		nodes = append(nodes, node)
		node = Node{}
	}
	return nodes
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
