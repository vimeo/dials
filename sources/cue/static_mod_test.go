package cue_test

import (
	"context"
	"testing"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/cue"
)

func TestTwoFileConfigWithSubPathAndSchema(t *testing.T) {
	t.Parallel()
	cs, initErr := cue.NewSource("./examples/simple_twofiles", ".:fizzlebat", "abcd")
	if initErr != nil {
		t.Fatalf("failed to init one-shot source: %s", initErr)
	}
	type cfgT struct {
		A string
		B int
	}

	ctx := context.Background()

	d, dErr := dials.Config(ctx, &cfgT{}, cs)
	if dErr != nil {
		t.Fatalf("failed to init dials: %s", dErr)
	}
	c := d.View()
	if exp := "some string"; c.A != exp {
		t.Errorf("config incorrectly parsed for field abcd.a; got %q; want %q", c.A, exp)
	}
	if exp := 2345; c.B != exp {
		t.Errorf("config incorrectly parsed for field abcd.b; got %d; want %d", c.B, exp)
	}
}

func TestTwoFileConfigWithSubPathUnclosed(t *testing.T) {
	t.Parallel()
	cs, initErr := cue.NewSource("./examples/simple_twofiles", ".:fizzlebat", "bcde")
	if initErr != nil {
		t.Fatalf("failed to init one-shot source: %s", initErr)
	}
	type cfgT struct {
		F string
	}

	ctx := context.Background()

	d, dErr := dials.Config(ctx, &cfgT{}, cs)
	if dErr != nil {
		t.Fatalf("failed to init dials: %s", dErr)
	}
	c := d.View()
	if exp := "abcd"; c.F != exp {
		t.Errorf("config incorrectly parsed for field bcde.F; got %q; want %q", c.F, exp)
	}
}
