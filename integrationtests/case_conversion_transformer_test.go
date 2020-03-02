package integrationtests

import (
	"context"
	"net"
	"testing"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/json"
	"github.com/vimeo/dials/static"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReformatDialsTags(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
		IPAddress       net.IP `dials:"ip_address"`
	}

	data := `{
		"databaseName": "something",
		"databaseAddress": "127.0.0.1",
		"ipAddress":"127.0.0.1"
	}`

	testCases := []struct {
		description string
		decoder     dials.Decoder
	}{
		{
			description: "JSON",
			decoder:     &json.Decoder{},
		},
		{
			description: "YAML",
			decoder:     &yaml.Decoder{},
		},
	}

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			myConfig := &testConfig{}
			view, err := dials.Config(
				context.Background(),
				myConfig,
				tagformat.ReformatDialsTagSource(&static.StringSource{Data: data, Decoder: tc.decoder}, tagformat.DecodeLowerSnakeCase, tagformat.EncodeLowerCamelCase),
			)
			require.NoError(t, err)

			c, ok := view.Get().(*testConfig)
			assert.True(t, ok)
			assert.Equal(t, "something", c.DatabaseName)
			assert.Equal(t, "127.0.0.1", c.DatabaseAddress)
			assert.Equal(t, net.IPv4(127, 0, 0, 1), c.IPAddress)
		})
	}
}

func TestReformatDialsTagsInNestedStruct(t *testing.T) {
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

	data := `{
		"databaseName": "something",
		"databaseAddress": "127.0.0.1",
		"databaseUser": {
			"username": "test",
			"password": "password",
			"otherStuff": {
				"something": {
					"anotherField": "asdf",
					"ipAddress":"127.0.0.1"
				}
			}
		}
	}`

	testCases := []struct {
		description string
		decoder     dials.Decoder
	}{
		{
			description: "JSON",
			decoder:     &json.Decoder{},
		},
		{
			description: "YAML",
			decoder:     &yaml.Decoder{},
		},
	}

	for _, testcase := range testCases {
		tc := testcase

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			myConfig := &testConfig{}
			view, err := dials.Config(
				context.Background(),
				myConfig,
				tagformat.ReformatDialsTagSource(&static.StringSource{Data: data, Decoder: tc.decoder}, tagformat.DecodeLowerSnakeCase, tagformat.EncodeLowerCamelCase),
			)
			require.NoError(t, err)

			c, ok := view.Get().(*testConfig)
			assert.True(t, ok)
			assert.Equal(t, "something", c.DatabaseName)
			assert.Equal(t, "test", c.DatabaseUser.Username)
			assert.Equal(t, "password", c.DatabaseUser.Password)
			assert.Equal(t, "asdf", c.DatabaseUser.OtherStuff.Something.AnotherField)
			assert.Equal(t, net.IPv4(127, 0, 0, 1), c.DatabaseUser.OtherStuff.Something.IPAddress)
		})
	}
}
