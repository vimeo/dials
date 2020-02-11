package toml

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/static"
)

func TestDecoder(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	tomlData := `
        val1 = "something"
        val2 = 42
`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: tomlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	require.True(t, ok)

	assert.Equal(t, "something", c.Val1)
	assert.Equal(t, 42, c.Val2)
}

func TestDecoderBadMarkup(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	badTOML := `
        val1 = something"
`

	myConfig := &testConfig{}
	_, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: badTOML, Decoder: &Decoder{}},
	)
	require.Error(t, err)
}
