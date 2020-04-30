package dials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	type testConfig struct {
		Foo string
	}

	tc := testConfig{
		Foo: "foo",
	}

	// no sources, just passing the defaults
	d, err := Config(context.Background(), &tc)
	require.NoError(t, err)

	filledConfig := &testConfig{}
	d.Fill(filledConfig)
	assert.Equal(t, "foo", filledConfig.Foo)
}
