package json

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/static"
	"github.com/vimeo/dials/tagformat"
)

func TestJSON(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	jsonData := `{
        "val1": "something",
        "val2": 42
    }`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.Val1)
	assert.Equal(t, 42, c.Val2)
}

func TestShallowlyNestedJSON(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username string `dials:"username"`
			Password string `dials:"password"`
		} `dials:"database_user"`
	}

	jsonData := `{
        "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password"
		}
    }`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
}

func TestDeeplyNestedJSON(t *testing.T) {
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

	jsonData := `{
	    "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"other_stuff": {
				"something": "asdf"
			}
		}
	}`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something)
}

func TestMoreDeeplyNestedJSON(t *testing.T) {
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

	jsonData := `{
	    "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"other_stuff": {
				"something": {
					"another_field": "asdf"
				}
			}
		}
	}`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
}

func TestReformatdialsToJSONTags(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
	}
	jsonData := `{
        "databaseName": "something",
        "databaseAddress": "127.0.0.1"
    }`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		tagformat.ReformatdialsTagSource(&static.StringSource{Data: jsonData, Decoder: &Decoder{}}, tagformat.DecodeLowerSnakeCase, tagformat.EncodeLowerCamelCase),
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "127.0.0.1", c.DatabaseAddress)
}

func TestReformatdialsTagsInNestedJSON(t *testing.T) {
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

	jsonData := `{
	    "databaseName": "something",
		"databaseAddress": "127.0.0.1",
		"databaseUser": {
			"username": "test",
			"password": "password",
			"otherStuff": {
				"something": {
					"anotherField": "asdf"
				}
			}
		}
	}`

	myConfig := &testConfig{}
	view, err := dials.Config(
		context.Background(),
		myConfig,
		tagformat.ReformatdialsTagSource(&static.StringSource{Data: jsonData, Decoder: &Decoder{}}, tagformat.DecodeLowerSnakeCase, tagformat.EncodeLowerCamelCase),
	)
	require.NoError(t, err)

	c, ok := view.Get().(*testConfig)
	assert.True(t, ok)

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
}
