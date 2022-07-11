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

// Option sets an option for typedDialsOptions
type Option interface {
	// typeless is a method for updating type-agnositc configuration
	// parameters
	typeless(*dialsOptions)
}

type typedOption[T any] interface {
	// embed Option
	Option

	typed(d *typedDialsOptions[T])
}

type typedOptionWrap[T any] func(*typedDialsOptions[T])

func (o typedOptionWrap[T]) typeless(d *dialsOptions) {
	panic("shouldn't be called; implements typedOption")
}

func (o typedOptionWrap[T]) typed(d *typedDialsOptions[T]) {
	o(d)
}

// make sure our typed wrapper type implements the correct interfaces
var _ typedOption[struct{}] = typedOptionWrap[struct{}](nil)
var _ Option = typedOptionWrap[struct{}](nil)

// typelessOptionWrap sets an option on dialsOptions
type typelessOptionWrap func(*dialsOptions)

func (o typelessOptionWrap) typeless(d *dialsOptions) {
	o(d)
}

// make sure our typeless wrapper type implements the Option interface
var _ Option = typelessOptionWrap(nil)

type dialsOptions struct {
	watch bool

	flagConfig     *flag.NameConfig
	flagSubstitute dials.Source

	autoSetToSlice bool

	fileKeyDecoder caseconversion.DecodeCasingFunc
	fileKeyEncoder caseconversion.EncodeCasingFunc
}

type typedDialsOptions[T any] struct {
	dialsOptions

	onWatchedError dials.WatchedErrorHandler[T]
	onNewConfig    dials.NewConfigHandler[T]
}

func getDefaultOption[T any]() *typedDialsOptions[T] {
	return &typedDialsOptions[T]{
		onWatchedError: nil,
		onNewConfig:    nil,
		dialsOptions: dialsOptions{
			watch: false,

			flagConfig:     flag.DefaultFlagNameConfig(),
			flagSubstitute: nil,

			autoSetToSlice: true,

			fileKeyDecoder: nil,
			fileKeyEncoder: nil,
		},
	}
}

// WithPflagSet allows you to use pflag source instead of the default flag source
func WithPflagSet(set *pflag.Set) Option {
	return typelessOptionWrap(func(d *dialsOptions) { d.flagSubstitute = set })
}

// WithFlagConfig sets the flag NameConfig to the specified one
func WithFlagConfig(flagConfig *flag.NameConfig) Option {
	return typelessOptionWrap(func(d *dialsOptions) { d.flagConfig = flagConfig })
}

// WithWatchingConfigFile allows to watch the config file by using the watching
// file source.  This defaults to false.
func WithWatchingConfigFile(enabled bool) Option {
	return typelessOptionWrap(func(d *dialsOptions) { d.watch = enabled })
}

// WithAutoSetToSlice allows you to set whether sets (map[string]struct{})
// should be automatically converted to slices ([]string) so they can be
// naturally parsed by JSON, YAML, or TOML parsers.  This defaults to true.
func WithAutoSetToSlice(enabled bool) Option {
	return typelessOptionWrap(func(d *dialsOptions) { d.autoSetToSlice = enabled })
}

// WithOnWatchedError registers a callback to record any errors encountered
// while stacking or verifying a new version of a configuration (if
// file-watching is enabled)
func WithOnWatchedError[T any](cb dials.WatchedErrorHandler[T]) Option {
	return typedOptionWrap[T](func(d *typedDialsOptions[T]) { d.onWatchedError = cb })
}

// WithOnNewConfig registers a callback to record new config versions
// reported while watching the config
func WithOnNewConfig[T any](cb dials.NewConfigHandler[T]) Option {
	return typedOptionWrap[T](func(d *typedDialsOptions[T]) { d.onNewConfig = cb })
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
	return typelessOptionWrap(func(d *dialsOptions) {
		d.fileKeyDecoder = decoder
		d.fileKeyEncoder = encoder
	})
}

// DecoderFactory should return the appropriate decoder based on the config file
// path that is passed as the string argument to DecoderFactory
type DecoderFactory func(string) dials.Decoder

// ConfigWithConfigPath is an interface config struct that supplies a
// ConfigPath() method to indicate which file to read as the config file once
// populated.
type ConfigWithConfigPath[T any] interface {
	*T
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
func ConfigFileEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, df DecoderFactory, options ...Option) (*dials.Dials[T], error) {
	blank := sourcewrap.Blank{}

	option := getDefaultOption[T]()
	for _, o := range options {
		switch of := o.(type) {
		case typedOption[T]:
			of.typed(option)
		default:
			of.typeless(&option.dialsOptions)
		}
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
	// We need to see the new config from the blank source.
	evChan := make(chan interface{}, 1)
	p := dials.Params[T]{
		OnWatchedError: func(ctx context.Context, err error, oldConfig, newConfig *T) {
			select {
			case blankErrCh <- err:
			default:
				if option.onWatchedError != nil {
					option.onWatchedError(ctx, err, oldConfig, newConfig)
				}
			}
		},
		OnNewConfig: func(ctx context.Context, oldConfig, newConfig *T) {
			select {
			case evChan <- newConfig:
			default:
				if option.onNewConfig != nil {
					option.onNewConfig(ctx, oldConfig, newConfig)
				}
			}
		},
		// Skip the initial verification to allow files to provide values that
		// will be considered during verification.  If a file source isn't
		// provided we'll appropriately call Verify before returning.
		SkipInitialVerification: true,
	}

	d, err := p.Config(ctx, (*T)(cfg), &blank, &env.Source{}, flagSrc)
	if err != nil {
		return nil, err
	}

	basecfg := d.View()
	cfgPath, filepathSet := (TP)(basecfg).ConfigPath()
	if !filepathSet {
		// Since we disabled initial verification earlier verify the config explicitly.
		// Without a config file, the sources never get re-stacked, so the `Verify()`
		// method is never run by `dials.Config`.
		if vf, ok := any(basecfg).(dials.VerifiedConfig); ok {
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
	case <-evChan:
	case err := <-blankErrCh:
		return d, fmt.Errorf("failed to stack/verify config with file layered: %w", err)
	}

	// Drain the event from the events channel so users of that interface
	// don't see the intermediate config.
	<-d.Events()

	// If there was no error, make sure the blankErrCh buffer is full so
	// subsequent calls always call the registered callback.
	select {
	case blankErrCh <- nil:
	default:
	}
	// Similarly, make sure that evChan is full
	select {
	case evChan <- nil:
	default:
	}
	return d, nil
}

// YAMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to YAML.
func YAMLConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, options ...Option) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &yaml.Decoder{} }, options...)
}

// JSONConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to JSON.
func JSONConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, options ...Option) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &json.Decoder{} }, options...)
}

// TOMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to TOML.
func TOMLConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, options ...Option) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &toml.Decoder{} }, options...)
}

// FileExtensionDecoderConfigEnvFlag takes advantage of the
// ConfigWithConfigPath cfg and thinly wraps ConfigFileEnvFlag and and thinly
// wraps ConfigFileEnvFlag choosing the dials.Decoder used when handling the
// file contents based on the file extension (from the limited set of JSON,
// YAML and TOML).
func FileExtensionDecoderConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, options ...Option) (*dials.Dials[T], error) {
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
