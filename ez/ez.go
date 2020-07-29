package ez

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/env"
	"github.com/vimeo/dials/file"
	"github.com/vimeo/dials/flag"
	"github.com/vimeo/dials/json"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/toml"
	"github.com/vimeo/dials/yaml"
)

// Option sets an option on dialsOptions
type Option func(*dialsOptions)

type dialsOptions struct {
	watch      bool
	flagConfig *flag.NameConfig
}

func getDefaultOption() *dialsOptions {
	return &dialsOptions{
		watch:      false,
		flagConfig: flag.DefaultFlagNameConfig(),
	}
}

// WithFlagConfig sets the flag NameConfig to the specified one
func WithFlagConfig(flagConfig *flag.NameConfig) Option {
	return func(d *dialsOptions) { d.flagConfig = flagConfig }
}

// WithWatchingConfigFile allows to watch the config file by using the watching
// file source
func WithWatchingConfigFile() Option {
	return func(d *dialsOptions) { d.watch = true }
}

// DecoderFactory should return the appropriate decoder based on the config file name
type DecoderFactory func(string) dials.Decoder

// ConfigWithConfigPath is an interface config struct that supplies a
// ConfigPath() method to indicate which file to read as the config file once
// populated.
type ConfigWithConfigPath interface {
	// ConfigPath implementations should return the configuration file to
	// be read as the first return-value, and true, or an empty string and
	// false.
	ConfigPath() (string, bool)
}

func fileSource(cfgPath string, decoder dials.Decoder, watch bool) (dials.Source, error) {
	if watch {
		fileSrc, fileErr := file.NewWatchingSource(cfgPath, decoder)
		if fileErr != nil {
			return nil, fmt.Errorf("invalid configuration path %q: %s", cfgPath, fileErr)
		}
		return fileSrc, nil
	}
	fsrc, fileErr := file.NewSource(cfgPath, decoder)
	if fileErr != nil {
		return nil, fmt.Errorf("invalid configuration path %q: %s", cfgPath, fileErr)
	}
	return fsrc, nil
}

// ConfigFileEnvFlag takes advantage of the ConfigWithConfigPath cfg to indicate
// what file to read and uses the passed decoder.
// Configuration values provided by the returned Dials are the result of
// stacking the sources in the following order:
//   - configuration file
//   - environment variables
//   - flags it registers with the standard library flags package
// The contents of cfg for the defaults
// cfg.ConfigPath() is evaluated on the stacked config with the file-contents omitted (using a "blank" source)
func ConfigFileEnvFlag(ctx context.Context, cfg ConfigWithConfigPath, df DecoderFactory, options ...Option) (*dials.Dials, error) {
	blank := sourcewrap.Blank{}

	option := getDefaultOption()
	for _, o := range options {
		o(option)
	}

	fset, flagErr := flag.NewCmdLineSet(option.flagConfig, cfg)
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
	decoder := df(cfgPath)
	if decoder == nil {
		return nil, fmt.Errorf("decoderFactory provided a nil decoder")
	}

	fileSrc, fileErr := fileSource(cfgPath, decoder, option.watch)
	if fileErr != nil {
		return nil, fileErr
	}

	blankErr := blank.SetSource(ctx, fileSrc)
	if blankErr != nil {
		return d, fmt.Errorf("failed to read config file: %s", blankErr)
	}

	// wait for the composition of the config struct with the config file values
	<-d.Events()
	return d, nil
}

// YAMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to YAML.
func YAMLConfigEnvFlag(ctx context.Context, cfg ConfigWithConfigPath, options ...Option) (*dials.Dials, error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &yaml.Decoder{} }, options...)
}

// JSONConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to JSON.
func JSONConfigEnvFlag(ctx context.Context, cfg ConfigWithConfigPath, options ...Option) (*dials.Dials, error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &json.Decoder{} }, options...)
}

// TOMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to TOML.
func TOMLConfigEnvFlag(ctx context.Context, cfg ConfigWithConfigPath, options ...Option) (*dials.Dials, error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &toml.Decoder{} }, options...)
}

// FileExtensionDecoderConfigEnvFlag TOMLConfigFlagEnv takes advantage of the
// ConfigWithConfigPath cfg and thinly wraps ConfigFileEnvFlag and and thinly
// wraps ConfigFileEnvFlag choosing the dials.Decoder used when handling the
// file contents based on the file extension (from the limited set of JSON,
// YAML and TOML).
func FileExtensionDecoderConfigEnvFlag(ctx context.Context, cfg ConfigWithConfigPath, options ...Option) (*dials.Dials, error) {
	return ConfigFileEnvFlag(ctx, cfg, func(fp string) dials.Decoder {
		ext := filepath.Ext(fp)
		switch strings.ToLower(ext) {
		case ".yaml", ".yml":
			return &yaml.Decoder{}
		case ".json":
			return &json.Decoder{}
		case ".toml":
			return &toml.Decoder{}
		default:
			return nil
		}
	}, options...)
}
