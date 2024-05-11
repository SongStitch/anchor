package main

import "github.com/moby/buildkit/frontend/dockerfile/parser"

func printNode(node *parser.Node) {
	for _, child := range node.Children {
		printNode(child)
	}

	if node.Next != nil {
		printNode(node.Next)
	}
}

func parseNode(node *parser.Node, architecture string) error {
	if node == nil {
		return nil
	}

	if node.Value == "FROM" {
		var err error
		image, err = attachDockerSha(node.Next)
		if err != nil {
			return err
		}
	} else if node.Value == "RUN" {
		err := parseRunCommand(node.Next, architecture)
		if err != nil {
			return err
		}
	} else if node.Next != nil {
		parseNode(node.Next, architecture)
	}

	for _, child := range node.Children {
		parseNode(child, architecture)
	}
	return nil
}
