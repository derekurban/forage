package main

import (
	"os"

	"github.com/derekurban/forage/internal/cli"
)

func main() {
	if code := cli.Execute(); code != 0 {
		os.Exit(code)
	}
}
