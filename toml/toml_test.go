package toml

import (
	"context"
	"testing"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/static"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestShallowlyNestedTOML(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username string `dials:"username"`
			Password string `dials:"password"`
		} `dials:"database_user"`
	}

	tomlData := `
	database_address = "127.0.0.1"
	database_name = "something"
	[database_user]
		password = "password"
		username = "test"
`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: tomlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
}

func TestDeeplyNestedTOML(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something string `dials:"something"`
			} `dials:"other_stuff"`
		} `dials:"database_user"`
	}

	tomlData := `
	    database_name = "something"
		database_address = "127.0.0.1"
		[database_user]
			username = "test"
			password = "password"
			[database_user.other_stuff]
				something = "asdf"
	`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: tomlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something)
}

func TestMoreDeeplyNestedTOML(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something struct {
					AnotherField string `dials:"another_field"`
				} `dials:"something"`
			} `dials:"other_stuff"`
		} `dials:"database_user"`
	}

	tomlData := `
	    database_name = "something"
		database_address = "127.0.0.1"
		[database_user]
			username = "test"
			password = "password"
			[database_user.other_stuff.something]
			another_field = "asdf"
	`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: tomlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
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
