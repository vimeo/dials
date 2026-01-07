//go:build go1.23

package integrationtests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/decoders/json"
	"github.com/vimeo/dials/sources/static"
)

type protocol int

const (
	HTTP protocol = iota
	HTTPS
	GitPlusSSH
)

func (p protocol) DialsValueMap() map[string]protocol {
	return map[string]protocol{
		"http":    HTTP,
		"https":   HTTPS,
		"git+ssh": GitPlusSSH,
	}
}

func TestEnum(t *testing.T) {
	type TestConfig struct {
		Proto  dials.Enum[protocol]
		Backup dials.Enum[protocol]
	}

	t.Run("ok", func(t *testing.T) {
		source := &static.StringSource{
			Data:    `{"proto": "git+ssh"}`,
			Decoder: &json.Decoder{},
		}

		c := &TestConfig{
			Backup: dials.EnumValue(HTTPS),
		}

		d, err := dials.Config(context.Background(), c, source)
		require.NoError(t, err)

		assert.Equal(t, GitPlusSSH, d.View().Proto.Value)
		assert.Equal(t, HTTPS, d.View().Backup.Value)
	})

	t.Run("case-sensitive", func(t *testing.T) {
		source := &static.StringSource{
			Data:    `{"proto": "HTTP"}`,
			Decoder: &json.Decoder{},
		}

		c := &TestConfig{}

		_, err := dials.Config(context.Background(), c, source)
		require.Error(t, err)
	})

	t.Run("no-mapping", func(t *testing.T) {
		source := &static.StringSource{
			Data:    `{"proto": "foo"}`,
			Decoder: &json.Decoder{},
		}

		c := &TestConfig{}

		_, err := dials.Config(context.Background(), c, source)
		require.Error(t, err)
	})
}

type instrument string

const (
	Piano instrument = "Piano"
	Bass  instrument = "Bass"
	Drums instrument = "Drums"
)

func (i instrument) DialsValueMap() map[string]instrument {
	return dials.StringValueMap(Piano, Bass, Drums)
}

func TestStringEnum(t *testing.T) {
	type TestConfig struct {
		Inst    dials.Enum[instrument]
		Other   dials.FuzzyEnum[instrument]
		Band    []dials.Enum[instrument]
		Members map[string]dials.Enum[instrument]
	}

	source := &static.StringSource{
		Data:    `{"inst": "Piano", "other": "drUMS", "band": ["Bass", "Drums"], "members": {"Paul": "Bass", "Ringo": "Drums"}}`,
		Decoder: &json.Decoder{},
	}

	c := &TestConfig{}

	d, err := dials.Config(context.Background(), c, source)
	require.NoError(t, err)

	v := d.View()
	assert.Equal(t, Piano, v.Inst.Value)
	assert.Equal(t, Drums, v.Other.Value)
	assert.Equal(t, Bass, v.Members["Paul"].Value)
	assert.Equal(t, Drums, v.Members["Ringo"].Value)
}

type genre int

const (
	Rock genre = iota
	Pop
	Jazz
)

func (g genre) String() string {
	switch g {
	case Rock:
		return "Rock"
	case Pop:
		return "Pop"
	case Jazz:
		return "Jazz"
	}
	return "Unknown"
}

func (g genre) DialsValueMap() map[string]genre {
	return dials.StringerValueMap(Rock, Pop, Jazz)
}

func TestStringerEnum(t *testing.T) {
	type TestConfig struct {
		Genre dials.Enum[genre]
	}

	source := &static.StringSource{
		Data:    `{"genre": "Jazz"}`,
		Decoder: &json.Decoder{},
	}

	c := &TestConfig{}

	d, err := dials.Config(context.Background(), c, source)
	require.NoError(t, err)

	v := d.View()
	assert.Equal(t, Jazz, v.Genre.Value)
}
