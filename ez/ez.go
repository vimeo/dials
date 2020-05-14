package ez

import (
	"context"
	"fmt"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/env"
	"github.com/vimeo/dials/file"
	"github.com/vimeo/dials/flag"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/yaml"
)

// ConfigWithConfigPath is an interface config struct that supplies a
// ConfigPath() method to indicate which file to read as the config file once
// populated.
type ConfigWithConfigPath interface {
	ConfigPath() (string, bool)
}

func fileSource(cfgPath string, watch bool) (dials.Source, error) {
	if watch {
		fileSrc, fileErr := file.NewWatchingSource(cfgPath, &yaml.Decoder{})
		if fileErr != nil {
			return nil, fmt.Errorf("invalid configuration path %q: %s", cfgPath, fileErr)
		}
		return fileSrc, nil
	}
	fsrc, fileErr := file.NewSource(cfgPath, &yaml.Decoder{})
	if fileErr != nil {
		return nil, fmt.Errorf("invalid configuration path %q: %s", cfgPath, fileErr)
	}
	return fsrc, nil
}

// YAMLConfigFlagEnv takes advantage of the ConfigWithConfigPath cfg.
func YAMLConfigFlagEnv(ctx context.Context, cfg ConfigWithConfigPath, watch bool) (*dials.Dials, error) {
	blank := sourcewrap.Blank{}

	fset, flagErr := flag.NewCmdLineSet(flag.DashesNameConfig(), cfg)
	if flagErr != nil {
		return nil, fmt.Errorf("failed to register commandline flags: %s", flagErr)
	}

	d, err := dials.Config(ctx, cfg, &blank, &env.Source{}, fset)
	if err != nil {
		return nil, err
	}

	basecfg := d.View().(ConfigWithConfigPath)
	cfgPath, filepathSet := basecfg.ConfigPath()
	if !filepathSet {
		// The callback indicated that we shouldn't read any config
		// file after all.
		return d, nil
	}

	fileSrc, fileErr := fileSource(cfgPath, watch)
	if fileErr != nil {
		return nil, fileErr
	}

	blankErr := blank.SetSource(ctx, fileSrc)
	if blankErr != nil {
		return d, fmt.Errorf("failed to read config file: %s", blankErr)
	}

	return d, nil
}
