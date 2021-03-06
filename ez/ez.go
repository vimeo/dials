package ez

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/env"
	"github.com/vimeo/dials/file"
	"github.com/vimeo/dials/flag"
	"github.com/vimeo/dials/json"
	"github.com/vimeo/dials/pflag"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/toml"
	"github.com/vimeo/dials/transform"
	"github.com/vimeo/dials/yaml"
)

// Option sets an option on dialsOptions
type Option func(*dialsOptions)

type dialsOptions struct {
	watch          bool
	onWatchedError dials.WatchedErrorHandler

	flagConfig     *flag.NameConfig
	flagSubstitute dials.Source

	autoSetToSlice bool

	fileKeyDecoder caseconversion.DecodeCasingFunc
	fileKeyEncoder caseconversion.EncodeCasingFunc
}

func getDefaultOption() *dialsOptions {
	return &dialsOptions{
		watch:          false,
		onWatchedError: nil,

		flagConfig:     flag.DefaultFlagNameConfig(),
		flagSubstitute: nil,

		autoSetToSlice: true,

		fileKeyDecoder: nil,
		fileKeyEncoder: nil,
	}
}

// WithPflagSet allows you to use pflag source instead of the default flag source
func WithPflagSet(set *pflag.Set) Option {
	return func(d *dialsOptions) { d.flagSubstitute = set }
}

// WithFlagConfig sets the flag NameConfig to the specified one
func WithFlagConfig(flagConfig *flag.NameConfig) Option {
	return func(d *dialsOptions) { d.flagConfig = flagConfig }
}

// WithWatchingConfigFile allows to watch the config file by using the watching
// file source.  This defaults to false.
func WithWatchingConfigFile(enabled bool) Option {
	return func(d *dialsOptions) { d.watch = enabled }
}

// WithAutoSetToSlice allows you to set whether sets (map[string]struct{})
// should be automatically converted to slices ([]string) so they can be
// naturally parsed by JSON, YAML, or TOML parsers.  This defaults to true.
func WithAutoSetToSlice(enabled bool) Option {
	return func(d *dialsOptions) { d.autoSetToSlice = enabled }
}

// WithOnWatchedError registers a callback to record any errors encountered
// while stacking or verifying a new version of a configuration (if
// file-watching is enabled)
func WithOnWatchedError(cb dials.WatchedErrorHandler) Option {
	return func(d *dialsOptions) { d.onWatchedError = cb }
}

// WithFileKeyNaming provides a method for manipulating the casing of the keys
// in the configuration file.  The decoder argument indicates how to interpret
// the `dials` tag's formatting.  If the `dials` tag is unspecified, the struct
// field's name will be used.  The encoder argument should indicate the format
// that dials should expect to find in the file.  For instance if you leave the
// `dials` tag unspecified and want a field named `SecretValues` in your
// configuration to map to a value in your config named "secret-values" you
// should call: WithFileKeyNaming(caseconversion.DecodeGoCamelCase,
// caseconversion.EncodeKebabCase). Note that this does not affect the flags or
// environment naming.  To manipulate flag naming, see `WithFlagConfig`.
func WithFileKeyNaming(decoder caseconversion.DecodeCasingFunc, encoder caseconversion.EncodeCasingFunc) Option {
	return func(d *dialsOptions) {
		d.fileKeyDecoder = decoder
		d.fileKeyEncoder = encoder
	}
}

// DecoderFactory should return the appropriate decoder based on the config file
// path that is passed as the string argument to DecoderFactory
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

	flagSrc := option.flagSubstitute
	if flagSrc == nil {
		// flag source isn't substituted so use the flag source
		fset, flagErr := flag.NewCmdLineSet(option.flagConfig, cfg)
		if flagErr != nil {
			return nil, fmt.Errorf("failed to register commandline flags: %s", flagErr)
		}
		flagSrc = fset
	}

	// If file-watching is not enabled, we should shutdown the monitor
	// goroutine when exiting this function.
	// Usually `dials.Config` is smart enough not to start a monitor when
	// there are no `Watcher` implementations in the source-list, but the
	// `Blank` source uses `Watcher` for its core functionality, so we need
	// to shutdown the blank source to actually clean up resources.
	if !option.watch {
		defer blank.Done(ctx)
	}
	// OnWatchedError is never called from this goroutine, so it can be
	// unbuffered without deadlocking.
	//
	// However, it is buffered to avoid a race where the non-blocking send
	// in the callback happens before the select statement at the bottom of
	// this function starts. If this weren't buffered, the send would fall
	// through to the delegated error-handler, and the select statement
	// would block until a new config version (or error) was created
	// (assuming this is a watching-mode) or possibly forever.
	blankErrCh := make(chan error, 1)
	p := dials.Params{
		OnWatchedError: func(ctx context.Context, err error, oldConfig, newConfig interface{}) {
			select {
			case blankErrCh <- err:
			default:
				if option.onWatchedError != nil {
					option.onWatchedError(ctx, err, oldConfig, newConfig)
				}
			}
		},
		// Skip the initial verification to allow files to provide values that
		// will be considered during verification.  If a file source isn't
		// provided we'll appropriately call Verify before returning.
		SkipInitialVerification: true,
	}

	d, err := p.Config(ctx, cfg, &blank, &env.Source{}, flagSrc)
	if err != nil {
		return nil, err
	}

	basecfg := d.View().(ConfigWithConfigPath)
	cfgPath, filepathSet := basecfg.ConfigPath()
	if !filepathSet {
		// Since we disabled initial verification earlier verify the config explicitly.
		// Without a config file, the sources never get re-stacked, so the `Verify()`
		// method is never run by `dials.Config`.
		if vf, ok := basecfg.(dials.VerifiedConfig); ok {
			if vfErr := vf.Verify(); vfErr != nil {
				return nil, fmt.Errorf("Initial configuration verification failed: %w", vfErr)
			}
		}

		// The callback indicated that we shouldn't read any config
		// file after all.
		return d, nil
	}

	decoder := df(cfgPath)
	if decoder == nil {
		return nil, fmt.Errorf("decoderFactory provided a nil decoder")
	}

	manglers := make([]transform.Mangler, 0, 2)

	if option.fileKeyEncoder != nil && option.fileKeyDecoder != nil {
		manglers = append(
			manglers,
			tagformat.NewTagReformattingMangler(
				common.DialsTagName, option.fileKeyDecoder, option.fileKeyEncoder,
			),
		)
	}

	if option.autoSetToSlice {
		manglers = append(manglers, &transform.SetSliceMangler{})
	}

	// add the manglers if any options called for them
	if len(manglers) > 0 {
		decoder = sourcewrap.NewTransformingDecoder(
			decoder,
			manglers...,
		)
	}

	fileSrc, fileErr := fileSource(cfgPath, decoder, option.watch)
	if fileErr != nil {
		return nil, fileErr
	}

	blankErr := blank.SetSource(ctx, fileSrc)
	if blankErr != nil {
		return d, fmt.Errorf("failed to read config file: %w", blankErr)
	}

	// wait for the composition of the config struct with the config file values
	select {
	case <-d.Events():
	case err := <-blankErrCh:
		return d, fmt.Errorf("failed to stack/verify config with file layered: %w", err)
	}
	// If there was no error, make sure the blankErrCh buffer is full so
	// subsequent calls always call the registered callback.
	select {
	case blankErrCh <- nil:
	default:
	}
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

// FileExtensionDecoderConfigEnvFlag takes advantage of the
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
