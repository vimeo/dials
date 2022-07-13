package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vimeo/dials/panels"
)

var p = panels.NewPanel(&animal{}, nil, panels.SetupParams[animal]{})

func main() {
	if err := p.Process(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run subcommand: %s\n", err)
		os.Exit(1)
	}
}
