package ez

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials/tagformat/caseconversion"
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
	view, dialsErr := YAMLConfigEnvFlag(ctx, c, Params[config]{})
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
	populatedConf := view.View()
	assert.EqualValues(t, expectedConfig, *populatedConf)
}

type beatlesConfig struct {
	YAMLPath       string
	BeatlesMembers map[string]string
}

// ConfigPath reflects where the path to config file is stored. Path field
// will be populated from environment variable
func (bc *beatlesConfig) ConfigPath() (string, bool) {
	return bc.YAMLPath, true
}

func TestYAMLConfigEnvFlagWithFileKeyNaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := &beatlesConfig{YAMLPath: "../testhelper/testconfig.yaml"}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c, Params[beatlesConfig]{
		DialsTagNameDecoder:  caseconversion.DecodeGoCamelCase,
		FileFieldNameEncoder: caseconversion.EncodeKebabCase,
	})

	require.NoError(t, dialsErr)

	expectedConfig := beatlesConfig{
		YAMLPath: "../testhelper/testconfig.yaml",
		BeatlesMembers: map[string]string{
			"John":   "guitar",
			"Paul":   "bass",
			"George": "guitar",
			"Ringo":  "drums",
		},
	}
	populatedConf := view.View()
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
	// check for the zero value, so we can distinguish between unset and set to
	// something bad.
	if c.Val1 == 0 {
		return fmt.Errorf("val1 unset")
	}

	if c.Val1 > 200 {
		return fmt.Errorf("val1 %d > 200", c.Val1)
	}
	return nil
}

func TestYAMLConfigEnvFlagWithValidatingConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpFile, tmpErr := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, tmpErr)
	tmpFile.Write([]byte("Val1: 789"))
	require.NoError(t, tmpFile.Sync())
	require.NoError(t, tmpFile.Close())
	path := tmpFile.Name()

	c := &validatingConfig{Path: path}
	d, dialsErr := YAMLConfigEnvFlag(ctx, c, Params[validatingConfig]{})
	assert.NotNil(t, d)
	require.EqualError(t, dialsErr, "failed to integrate file source: failed to propagate change: stacking failed: val1 789 > 200")
}

func TestYAMLConfigEnvFlagWithValidatingConfigInitiallyValid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "fim1.yaml")
	require.NoError(t, os.WriteFile(path, []byte("Val1: 189"), os.FileMode(0660)))

	errCh := make(chan error)
	errHandler := func(ctx context.Context, err error, oldConfig, newConfig *validatingConfig) {
		errCh <- err
	}
	c := &validatingConfig{Path: path}
	view, dialsErr := YAMLConfigEnvFlag(ctx, c, Params[validatingConfig]{OnWatchedError: errHandler, WatchConfigFile: true})
	require.NoError(t, dialsErr)
	assert.NotNil(t, view)

	expectedConfig := validatingConfig{
		Path: path,
		Val1: 189,
		Val2: "",
		Set:  map[string]struct{}(nil),
	}
	populatedConf := view.View()
	assert.EqualValues(t, expectedConfig, *populatedConf)

	tmpPath2 := filepath.Join(tmpDir, "_tmp_fim1.yaml")
	require.NoError(t, os.WriteFile(tmpPath2, []byte("Val1: 201"), os.FileMode(0660)))

	require.NoError(t, os.Rename(tmpPath2, path))

	// Write a new version of the config
	select {
	case newcfg := <-view.Events():
		t.Errorf("unexpected new version; should have failed verification: %+v", newcfg)
	case err := <-errCh:
		require.EqualError(t, err, "val1 201 > 200")
	}
}

func TestJSONConfigEnvFlagWithNewConfigCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "fim1.json")

	origCfgFileContents := struct {
		Val1 int
		Val2 string
		Set  []string
	}{
		Val1: rand.Intn(100) + 1, // make sure to keep below 200 and above 0
		Val2: "",
		Set:  []string{"foo", "bar", "baz"},
	}
	origJS, jsMarshalErr := json.Marshal(&origCfgFileContents)
	require.NoError(t, jsMarshalErr)

	require.NoError(t, os.WriteFile(path, origJS, 0660))

	newCfg := make(chan *validatingConfig, 1)
	newConfigCB := func(ctx context.Context, oldConfig, newConfig *validatingConfig) {
		newCfg <- newConfig
	}
	c := &validatingConfig{Path: path}
	view, dialsErr := JSONConfigEnvFlag(ctx, c, Params[validatingConfig]{OnNewConfig: newConfigCB, WatchConfigFile: true})
	require.NoError(t, dialsErr)
	assert.NotNil(t, view)

	expectedConfig := validatingConfig{
		Path: path,
		Val1: origCfgFileContents.Val1,
		Val2: "",
		Set:  map[string]struct{}{"foo": {}, "bar": {}, "baz": {}},
	}
	populatedConf := view.View()
	assert.EqualValues(t, expectedConfig, *populatedConf)

	select {
	case cfg, ok := <-newCfg:
		if !ok {
			panic("newCfg channel closed somehow")
		}
		t.Errorf("unexpected config before update: %+v", cfg)
	default:
	}

	select {
	case cfg, ok := <-view.Events():
		if !ok {
			panic("events channel closed")
		}
		t.Errorf("unexpected events config before update: %+v", cfg)
	default:
	}

	// Write a new version of the config
	updatedCfgContents := origCfgFileContents
	updatedCfgContents.Val1 += 87
	updateJS, updatejsMarshalErr := json.Marshal(&updatedCfgContents)
	require.NoError(t, updatejsMarshalErr)

	tmpPath2 := filepath.Join(tmpDir, "_tmp_fim1.json")
	require.NoError(t, os.WriteFile(tmpPath2, updateJS, os.FileMode(0660)))

	require.NoError(t, os.Rename(tmpPath2, path))

	expectedFinalConfig := expectedConfig
	expectedFinalConfig.Val1 = updatedCfgContents.Val1

	finalCfg := <-newCfg
	assert.EqualValues(t, expectedFinalConfig, *finalCfg)

	finalViewEventCfg := <-view.Events()
	assert.EqualValues(t, expectedFinalConfig, *finalViewEventCfg)
	assert.EqualValues(t, expectedFinalConfig, *view.View())
}
