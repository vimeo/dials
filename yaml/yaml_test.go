package yaml

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/static"
)

func TestYAML(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	yamlData := `---
        val1: something
        val2: 42
`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: yamlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.Val1)
	assert.Equal(t, 42, c.Val2)
}

func TestShallowlyNestedYAML(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something string        `dials:"something"`
				IPAddress net.IP        `dials:"ip_address"`
				Timeout   time.Duration `dials:"timeout"`
			} `dials:"other_stuff"`
		} `dials:"database_user"`
	}

	yamlData := `{
	    "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"other_stuff": {
				"something": "asdf",
				"ip_address": "123.10.11.121",
				"timeout": "10s",
			}
		}
	}`

	myConfig := &testConfig{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := dials.Config(
		ctx,
		myConfig,
		&static.StringSource{Data: yamlData, Decoder: &Decoder{}},
	)

	require.NoError(t, err)
	c := d.View()

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something)
	assert.Equal(t, net.IPv4(123, 10, 11, 121), c.DatabaseUser.OtherStuff.IPAddress)
	assert.Equal(t, time.Duration(10*time.Second), c.DatabaseUser.OtherStuff.Timeout)
}

func TestMoreDeeplyNestedYAML(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something struct {
					AnotherField string        `dials:"another_field"`
					IPAddress    net.IP        `dials:"ip_address"`
					Timeout      time.Duration `dials:"timeout"`
				} `dials:"something"`
			} `dials:"other_stuff"`
		} `dials:"database_user"`
	}

	yamlData := `{
	    "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"other_stuff": {
				"something": {
					"another_field": "asdf",
					"ip_address": "123.10.11.121",
					"timeout": "10s", 
				}
			}
		}
	}`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: yamlData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
	assert.Equal(t, net.IPv4(123, 10, 11, 121), c.DatabaseUser.OtherStuff.Something.IPAddress)
	assert.Equal(t, time.Duration(10*time.Second), c.DatabaseUser.OtherStuff.Something.Timeout)
}

func TestDecoderBadMarkup(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	yamlData := `---
        val1 something
        val 2: 42
`

	myConfig := &testConfig{}
	_, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: yamlData, Decoder: &Decoder{}},
	)
	require.Error(t, err)
}
