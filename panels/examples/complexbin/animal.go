package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vimeo/dials/panels"
)

type animal struct {
	Type  string // dog, lion, duck
	Legs  int8
	Group string // mammal, reptile, marsupial
}

type animals struct {
	m map[string]animal
}

type animalTraits struct {
	Name string
}

func (a *animals) Description(scPath []string) string {
	return "Get a list of all the animals in the directory that match the criteria"
}

func (a *animals) ShortUsage(scPath []string) string {
	return strings.Join(scPath, " ")
}

func (a *animals) LongUsage(scPath []string) string {
	return strings.Join(scPath, " ") + "\n\tget all the animals"
}

func (a *animals) DefaultConfig() *animalTraits { return &animalTraits{} }
func (a *animals) SetupParams() panels.SetupParams[animalTraits] {
	return panels.SetupParams[animalTraits]{}
}

func (a *animals) Run(ctx context.Context, s *panels.Handle[animal, animalTraits]) error {

	rootDial := s.RootDials
	c := rootDial.View()
	an := s.Dials.View()

	foundMatch := false
	for n, animal := range a.m {
		if an.Name != "" && n != an.Name {
			continue
		}

		if c.Group != "" && c.Group != animal.Group {
			continue
		}

		if c.Legs != 0 && c.Legs != animal.Legs {
			continue
		}

		if c.Type != "" && c.Type != animal.Type {
			continue
		}

		foundMatch = true
		fmt.Fprintf(s.W, "Name: %s\tType:%s\tLegs: %d\n", n, animal.Type, animal.Legs)
	}
	if !foundMatch {
		fmt.Fprintf(s.W, "No animals found matching the specified requirements\n")
	}
	return nil
}

func defaultAnimals() *animals {
	return &animals{
		m: map[string]animal{
			"Tom": {
				Type:  "cat",
				Legs:  4,
				Group: "mammal",
			},
			"Jerry": {
				Type:  "mouse",
				Legs:  4,
				Group: "mammal",
			},
			"Mickey": {
				Type:  "mouse",
				Legs:  4,
				Group: "mammal",
			},
			"Minnie": {
				Type:  "mouse",
				Legs:  4,
				Group: "mammal",
			},
			"Donald": {
				Type:  "duck",
				Legs:  2,
				Group: "reptile",
			},
			"Goofy": {
				Type:  "dog",
				Legs:  4,
				Group: "mammal",
			},
		},
	}
}

var _ = panels.Must(panels.Register[animal, animalTraits](p, "animals", defaultAnimals()))
