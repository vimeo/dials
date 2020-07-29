package ez

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Config struct {
	// Path will contain the path to the config file and will be set by
	// environment variable
	Path string `dials:"CONFIGPATH"`
	Val1 int    `dials:"Val1"`
	Val2 string `dials:"Val2"`
}

// ConfigPath reflects where the path to config file is stored. Path field
// will be populated from environment variable
func (c *Config) ConfigPath() (string, bool) {
	return c.Path, true
}

// TestYAMLConfigEnvFlag cannot run concurrently with other tests because of
// environment manipulation.
func TestYAMLConfigEnvFlag(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	envErr := os.Setenv("CONFIGPATH", "../testhelper/testconfig.yaml")
	require.NoError(t, envErr)
	defer os.Unsetenv("CONFIGPATH")

	c := &Config{}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c)
	require.NoError(t, dialsErr)

	// Val1 and Val2 come from the config file and Path will be populated from env variable
	expectedConfig := Config{
		Path: "../testhelper/testconfig.yaml",
		Val1: 456,
		Val2: "hello-world",
	}
	populatedConf := view.View().(*Config)
	assert.EqualValues(t, expectedConfig, *populatedConf)
}
