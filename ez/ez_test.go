package ez

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type config struct {
	// Path will contain the path to the config file and will be set by
	// environment variable
	Path string              `dials:"CONFIGPATH"`
	Val1 int                 `dials:"Val1"`
	Val2 string              `dials:"Val2"`
	Set  map[string]struct{} `dials:"Set"`
}

// ConfigPath reflects where the path to config file is stored. Path field
// will be populated from environment variable
func (c *config) ConfigPath() (string, bool) {
	return c.Path, true
}

// TestYAMLConfigEnvFlag cannot run concurrently with other tests because of
// environment manipulation.
func TestYAMLConfigEnvFlagWithValidConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	envErr := os.Setenv("CONFIGPATH", "../testhelper/testconfig.yaml")
	require.NoError(t, envErr)
	defer os.Unsetenv("CONFIGPATH")

	c := &config{}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c)
	require.NoError(t, dialsErr)

	// Val1 and Val2 come from the config file and Path will be populated from env variable
	expectedConfig := config{
		Path: "../testhelper/testconfig.yaml",
		Val1: 456,
		Val2: "hello-world",
		Set: map[string]struct{}{
			"Keith": {},
			"Gary":  {},
			"Jack":  {},
		},
	}
	populatedConf := view.View().(*config)
	assert.EqualValues(t, expectedConfig, *populatedConf)
}

type validatingConfig struct {
	// Path will contain the path to the config file and will be set by
	// environment variable
	Path string              `dials:"CONFIGPATHFIM"`
	Val1 int                 `dials:"Val1"`
	Val2 string              `dials:"Val2"`
	Set  map[string]struct{} `dials:"Set"`
}

// ConfigPath reflects where the path to config file is stored. Path field
// will be populated from environment variable
func (c *validatingConfig) ConfigPath() (string, bool) {
	return c.Path, true
}

func (c *validatingConfig) Verify() error {
	if c.Val1 > 200 {
		return fmt.Errorf("val1 %d > 200", c.Val1)
	}
	return nil
}

func TestYAMLConfigEnvFlagWithValidatingConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpFile, tmpErr := ioutil.TempFile("", "")
	require.NoError(t, tmpErr)
	tmpFile.Write([]byte("Val1: 789"))
	require.NoError(t, tmpFile.Sync())
	require.NoError(t, tmpFile.Close())
	path := tmpFile.Name()
	defer os.Remove(path)

	c := &validatingConfig{Path: path}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c)
	assert.NotNil(t, view)
	require.EqualError(t, dialsErr, "failed to stack/verify config with file layered: val1 789 > 200")
}

func TestYAMLConfigEnvFlagWithValidatingConfigInitiallyValid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, tmpErr := ioutil.TempDir("", "")
	require.NoError(t, tmpErr)
	defer os.RemoveAll(tmpDir)
	path := filepath.Join(tmpDir, "fim1.yaml")
	require.NoError(t, ioutil.WriteFile(path, []byte("Val1: 189"), 0660))
	defer os.Remove(path)

	errCh := make(chan error)
	errHandler := func(ctx context.Context, err error, oldConfig, newConfig interface{}) {
		errCh <- err
	}
	c := &validatingConfig{Path: path}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c, WithOnWatchedError(errHandler), WithWatchingConfigFile(true))
	require.NoError(t, dialsErr)
	assert.NotNil(t, view)

	expectedConfig := validatingConfig{
		Path: path,
		Val1: 189,
		Val2: "",
		Set:  map[string]struct{}{},
	}
	populatedConf := view.View().(*validatingConfig)
	assert.EqualValues(t, expectedConfig, *populatedConf)

	tmpPath2 := filepath.Join(tmpDir, "_tmp_fim1.yaml")
	require.NoError(t, ioutil.WriteFile(tmpPath2, []byte("Val1: 201"), os.FileMode(0660)))

	require.NoError(t, os.Rename(tmpPath2, path))

	// Write a new version of the config
	select {
	case newcfg := <-view.Events():
		t.Errorf("unexpected new version; should have failed verification: %+v", newcfg)
	case err := <-errCh:
		require.EqualError(t, err, "val1 201 > 200")
	}
}
