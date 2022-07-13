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

type person struct {
	Name string
}

func (h *helloCmd) DefaultConfig() *struct{} {
	return &struct{}{}
}

func (h *helloCmd) SetupParams() panels.SetupParams[struct{}] {
	return panels.SetupParams[struct{}]{}
}

func (h *helloCmd) Run(ctx context.Context, s *panels.Handle[struct{}, struct{}]) error {
	switch s.SCPath[1] {
	case "hello":
		fmt.Fprintf(s.W, "Hello World!\n")
	case "hola":
		fmt.Fprintf(s.W, "Hola Mundo!\n")
	case "привет":
		fmt.Fprintf(s.W, "Превет мир!\n")
	case "नमस्ते":
		fmt.Fprintf(s.W, "नमस्ते दुनिया!\n")
	case "שָׁלוֹם":
		fmt.Fprintf(s.W, "!שָׁלוֹם עוֹלָם\n")
	default:
		return fmt.Errorf("unknown language: %q", s.SCPath[1])
	}

	fmt.Fprintf(s.W, "I was called as %q\n", s.SCPath)
	return nil
}

type ruДобрийДен struct {
	helloCmd
}

func (r *ruДобрийДен) Run(ctx context.Context, s *panels.Handle[struct{}, person]) error {
	p := s.Dials.View()
	fmt.Fprintf(s.W, "Добрий ден %s!\n", p.Name)

	return nil
}

func (r *ruДобрийДен) SetupParams() panels.SetupParams[person] {
	return panels.SetupParams[person]{}
}

func (r *ruДобрийДен) DefaultConfig() *person {
	return &person{
		Name: "мир",
	}
}

type hiCmd struct {
	helloCmd
}

func (h *hiCmd) SetupParams() panels.SetupParams[person] {
	return panels.SetupParams[person]{}
}

func (h *hiCmd) Run(ctx context.Context, s *panels.Handle[struct{}, person]) error {
	p := s.Dials.View()
	fmt.Fprintf(s.W, "नमस्ते %s!\n", p.Name)

	return nil
}

func (h *hiCmd) DefaultConfig() *person {
	return &person{
		Name: "दुनिया",
	}
}

type iwCmd struct {
	helloCmd
}

func (i *iwCmd) SetupParams() panels.SetupParams[person] {
	return panels.SetupParams[person]{}
}

func (i *iwCmd) Run(ctx context.Context, s *panels.Handle[struct{}, person]) error {
	p := s.Dials.View()
	fmt.Fprintf(s.W, "!%s שָׁלוֹם\n", p.Name)

	return nil
}

func (i *iwCmd) DefaultConfig() *person {
	return &person{
		Name: "עוֹלָם",
	}
}

type greeting struct {
	Phrase string
}

type esMundoCmd struct {
	helloCmd
}

func (e *esMundoCmd) SetupParams() panels.SetupParams[greeting] {
	return panels.SetupParams[greeting]{}
}

func (e *esMundoCmd) DefaultConfig() *greeting {
	return &greeting{
		Phrase: "Buenos días",
	}
}

func (e *esMundoCmd) Run(ctx context.Context, s *panels.Handle[struct{}, greeting]) error {
	g := s.Dials.View()

	fmt.Fprintf(s.W, "%s Mundo!\n", g.Phrase)

	return nil
}

type enWorldCmd struct {
	helloCmd
}

func (e *enWorldCmd) SetupParams() panels.SetupParams[greeting] {
	return panels.SetupParams[greeting]{}
}
func (e *enWorldCmd) DefaultConfig() *greeting {
	return &greeting{
		Phrase: "Good Morning",
	}
}

func (e *enWorldCmd) Run(ctx context.Context, s *panels.Handle[struct{}, greeting]) error {
	g := s.Dials.View()

	fmt.Fprintf(s.W, "%s World!\n", g.Phrase)

	return nil
}

func init() {
	panels.Register[struct{}, struct{}](&p, "hello", &helloCmd{})
	panels.Register[struct{}, struct{}](&p, "hola", &helloCmd{})
	panels.Register[struct{}, struct{}](&p, "привет", &helloCmd{})
	panels.Register[struct{}, struct{}](&p, "नमस्ते", &helloCmd{})
	panels.Register[struct{}, struct{}](&p, "שָׁלוֹם", &helloCmd{})
	panels.Register[struct{}, person](&p, "ru", &ruДобрийДен{})
	panels.Register[struct{}, person](&p, "hi", &hiCmd{})
	panels.Register[struct{}, greeting](&p, "es", &esMundoCmd{})
	panels.Register[struct{}, greeting](&p, "en", &enWorldCmd{})
	panels.Register[struct{}, person](&p, "iw", &iwCmd{})
}
