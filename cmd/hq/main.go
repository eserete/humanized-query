package main

import (
	"os"

	"github.com/eduardoserete/humanized-query/cmd/hq/commands"
)

func main() {
	if err := commands.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
