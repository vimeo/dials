package cue

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/static"
)

func TestCueJSON(t *testing.T) {
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
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.Val1)
	assert.Equal(t, 42, c.Val2)
}

func TestShallowlyNestedCueJSON(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username  string `dials:"username"`
			Password  string `dials:"password"`
			IPAddress net.IP `dials:"ip_address"`
		} `dials:"database_user"`
	}

	jsonData := `{
        "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"ip_address": "123.10.11.121"
		}
    }`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, net.IPv4(123, 10, 11, 121), c.DatabaseUser.IPAddress)
}

func TestDeeplyNestedCueJSON(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something        string        `dials:"something"`
				IPAddress        net.IP        `dials:"ip_address"`
				SomeTimeout      time.Duration `dials:"some_timeout"`
				SomeOtherTimeout time.Duration `dials:"some_other_timeout"`
				SomeLifetime     time.Duration `dials:"some_lifetime_ns"`
			} `dials:"other_stuff"`
		} `dials:"database_user"`
	}

	cueData := `
	    import "time"
	    "database_name": "something",
		"database_address": "127.0.0.1",
		"database_user": {
			"username": "test",
			"password": "password",
			"other_stuff": {
				"something": "asdf",
				"ip_address": "123.10.11.121"
				"some_timeout": "13s"
				"some_other_timeout": 87 * time.Second,
				"some_lifetime_ns": 378,
			}
		}
	`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: cueData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something)
	assert.Equal(t, time.Second*13, c.DatabaseUser.OtherStuff.SomeTimeout)
	assert.Equal(t, time.Second*87, c.DatabaseUser.OtherStuff.SomeOtherTimeout)
	assert.Equal(t, time.Nanosecond*378, c.DatabaseUser.OtherStuff.SomeLifetime)
	assert.Equal(t, net.IPv4(123, 10, 11, 121), c.DatabaseUser.OtherStuff.IPAddress)

}

func TestMoreDeeplyNestedCueJSON(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		DatabaseUser    struct {
			Username   string `dials:"username"`
			Password   string `dials:"password"`
			OtherStuff struct {
				Something struct {
					AnotherField string `dials:"another_field"`
					IPAddress    net.IP `dials:"ip_address"`
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
					"another_field": "asdf",
					"ip_address": "123.10.11.121"
				}
			}
		}
	}`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.DatabaseName)
	assert.Equal(t, "test", c.DatabaseUser.Username)
	assert.Equal(t, "password", c.DatabaseUser.Password)
	assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
	assert.Equal(t, net.IPv4(123, 10, 11, 121), c.DatabaseUser.OtherStuff.Something.IPAddress)

}

func TestCueSimpleWithRef(t *testing.T) {
	type testConfig struct {
		Val1 string
		C    chan struct{}
		Val2 int
	}
	jsonData := `
_foobar: {
	j: "something"
}
"Val1": _foobar.j
"Val2": 42
`

	myConfig := &testConfig{}
	d, err := dials.Config(
		context.Background(),
		myConfig,
		&static.StringSource{Data: jsonData, Decoder: &Decoder{}},
	)
	require.NoError(t, err)

	c := d.View()

	assert.Equal(t, "something", c.Val1)
	assert.Equal(t, 42, c.Val2)
}
