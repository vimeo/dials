package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vimeo/dials/panels"
)

type helloCmd struct{}

func (h *helloCmd) Description(scPath []string) string {
	return fmt.Sprintf("%s %s says %q", scPath[0], scPath[1], scPath[1])
}

// scPath is the subcommand-path, including the binary-name (args up to
// this subcommand with flags stripped out)
func (h *helloCmd) ShortUsage(scPath []string) string {
	return strings.Join(scPath, " ")
}

// scPath is the subcommand-path, including the binary-name (args up to
// this subcommand with flags stripped out)
func (h *helloCmd) LongUsage(scPath []string) string {
	return strings.Join(scPath, " ") + "\n\tand wave"
}

func (h *helloCmd) DefaultConfig() interface{} {
	return &struct{}{}
}

func (h *helloCmd) SetupParams() panels.SetupParams {
	return panels.SetupParams{}
}

func (h *helloCmd) Run(ctx context.Context, s *panels.Something) error {
	switch s.SCPath[1] {
	case "hello":
		fmt.Fprintf(s.W, "Hello World!\n")
	case "hola":
		fmt.Fprintf(s.W, "Hola Mundo!\n")
	case "превет":
		fmt.Fprintf(s.W, "Превет мир!\n")

	default:
		return fmt.Errorf("unknown language: %q", s.SCPath[1])
	}
	fmt.Fprintf(s.W, "I was called as %q\n", s.SCPath)
	return nil
}

func init() {
	p.Register("hello", &helloCmd{})
	p.Register("hola", &helloCmd{})
	p.Register("превет", &helloCmd{})
}
