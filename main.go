package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/songstitch/anchor/cmd"
)

func run() error {
	err := cmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		color.Red("%s", err)
		os.Exit(1)
	}
}
