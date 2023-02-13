package ez

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/decoders/cue"
	"github.com/vimeo/dials/decoders/json"
	"github.com/vimeo/dials/decoders/toml"
	"github.com/vimeo/dials/decoders/yaml"
	"github.com/vimeo/dials/sources/env"
	"github.com/vimeo/dials/sources/file"
	"github.com/vimeo/dials/sources/flag"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/transform"
)

// Params defines options for the configuration functions within this package
// All fields can be left empty, zero values will be replaced with sane defaults.
// As such, it is expected that struct-literals of this type will be sparsely
// populated in almost all cases.
type Params[T any] struct {
	// OnWatchedError registers a callback to record any errors encountered
	// while stacking or verifying a new version of a configuration (if
	// file-watching is enabled)
	OnWatchedError dials.WatchedErrorHandler[T]
	// OnNewConfig registers a callback to record new config versions
	// reported while watching the config
	OnNewConfig dials.NewConfigHandler[T]

	// WatchConfigFile allows one to watch the config file by using the
	// watching file source.
	WatchConfigFile bool

	// FlagConfig sets the flag NameConfig
	FlagConfig *flag.NameConfig

	// FlagSource one to use a different flag source instead of the
	// default commandline-source from the flag package.
	// This explicitly supports use with the pflag package's source.Set type.
	FlagSource dials.Source

	// DisableAutoSetToSlice allows you to set whether sets (map[string]struct{})
	// should be automatically converted to slices ([]string) so they can be
	// naturally parsed by JSON, YAML, or TOML parsers.  This is named as a
	// negation so AutoSetToSlice is enabled by default.
	DisableAutoSetToSlice bool

	// DialsTagNameDecoder indicates the naming scheme in use for
	// dials tags in this struct (copied from field-names if unspecified).
	// This is only useful if using a FileFieldNameEncoder (below)
	//
	// In many cases, the default of caseconversion.DecodeGoCamelCase
	// should work. This field exists to allow for other naming schemes.
	// (e.g. SCREAMING_SNAKE_CASE).
	//
	// See the caseconversion package for available decoders.
	// Note that this does not affect the flags or environment-variable
	// naming.  To manipulate flag naming, see `WithFlagConfig`.
	DialsTagNameDecoder caseconversion.DecodeCasingFunc

	// FileFieldNameEncoder allows one to manipulate the casing of the keys
	// in the configuration file.  See the DialsTagNameDecoder field for
	// controlling how dials splits field-names into "words".
	// Fields that lack a `dials` tag's formatting.  If the `dials` tag is unspecified, the struct
	// field's name will be used.  The encoder argument should indicate the format
	// that dials should expect to find in the file.
	//
	// For instance if you leave the `dials` tag unspecified and want a
	// field named `SecretValues` in your configuration to map to a value
	// in your config named "secret-values" you can set:
	//   Params {
	//	DialsTagNameDecoder: caseconversion.DecodeGoCamelCase,
	//	FileFieldNameEncoder: caseconversion.EncodeKebabCase,
	//   }
	// Note that this does not affect the flags or environment variable
	// naming.  To manipulate flag naming, see [Params.FlagConfig].
	FileFieldNameEncoder caseconversion.EncodeCasingFunc
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
//
// The contents of cfg for the defaults
// cfg.ConfigPath() is evaluated on the stacked config with the file-contents omitted (using a "blank" source)
func ConfigFileEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, df DecoderFactory, params Params[T]) (*dials.Dials[T], error) {
	blank := sourcewrap.Blank{}

	flagSrc := params.FlagSource
	if flagSrc == nil {
		flagNameCfg := params.FlagConfig
		if flagNameCfg == nil {
			flagNameCfg = flag.DefaultFlagNameConfig()
		}
		// flag source isn't substituted so use the flag source
		fset, flagErr := flag.NewCmdLineSet(flagNameCfg, cfg)
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
	if !params.WatchConfigFile {
		defer blank.Done(ctx)
	}

	watchErrCB := dials.WatchedErrorHandler[T](nil)
	if params.OnWatchedError != nil {
		// if there's an error-callback set, it'll only be used if the config is invalid
		// however, we don't really want to force callers to handle
		// error-callbacks before this function has returned.
		// defer a close on this channel so upon exit, all calls to the
		// watch callback pas through to the wrapped callback.
		passError := make(chan struct{})
		defer close(passError)
		watchErrCB = func(ctx context.Context, err error, oldConfig, newConfig *T) {
			select {
			case <-passError:
				// passError closed, so ConfigFileEnvFlag has exited, and we can now deliver new config errors.
			default:
				return
			}
			if params.OnWatchedError != nil {
				params.OnWatchedError(ctx, err, oldConfig, newConfig)
			}
		}
	}

	dp := dials.Params[T]{
		OnWatchedError: watchErrCB,
		// We'll set the OnNewConfig callback after we've inserted the
		// blank source (or decided that the config is otherwise
		// valid).
		OnNewConfig: nil,
		// Skip the initial verification to allow files to provide values that
		// will be considered during verification.  If a file source isn't
		// provided we'll appropriately call Verify before returning.
		SkipInitialVerification: true,
	}

	d, err := dp.Config(ctx, (*T)(cfg), &blank, &env.Source{}, flagSrc)
	if err != nil {
		return nil, err
	}

	basecfg, tok := d.ViewVersion()
	cfgPath, filepathSet := (TP)(basecfg).ConfigPath()
	if !filepathSet {
		// Since we disabled initial verification earlier verify the config explicitly.
		// Without a config file, the sources never get re-stacked, so the `Verify()`
		// method is never run by `dials.Config`.
		if vf, ok := any(basecfg).(dials.VerifiedConfig); ok {
			if vfErr := vf.Verify(); vfErr != nil {
				return nil, fmt.Errorf("initial configuration verification failed: %w", vfErr)
			}
		}
		if params.OnNewConfig != nil {
			// new config callback set; register it just before returning.
			d.RegisterCallback(ctx, tok, params.OnNewConfig)
		}

		// The callback indicated that we shouldn't read any config
		// file after all.
		return d, nil
	}

	decoder := df(cfgPath)
	if decoder == nil {
		return nil, fmt.Errorf("decoderFactory provided a nil decoder for path: %s", cfgPath)
	}

	manglers := make([]transform.Mangler, 0, 2)

	if params.FileFieldNameEncoder != nil {
		tagDecoder := params.DialsTagNameDecoder
		if tagDecoder == nil {
			tagDecoder = caseconversion.DecodeGoCamelCase
		}
		manglers = append(
			manglers,
			tagformat.NewTagReformattingMangler(
				common.DialsTagName, tagDecoder, params.FileFieldNameEncoder,
			),
		)
	}

	if !params.DisableAutoSetToSlice {
		manglers = append(manglers, &transform.SetSliceMangler{})
	}

	// add the manglers if any options called for them
	if len(manglers) > 0 {
		decoder = sourcewrap.NewTransformingDecoder(
			decoder,
			manglers...,
		)
	}

	fileSrc, fileErr := fileSource(cfgPath, decoder, params.WatchConfigFile)
	if fileErr != nil {
		return nil, fileErr
	}

	// SetSource blocks until the new config is re-stacked. It will fail if
	// either the file source fails, or the config fails validation.
	blankErr := blank.SetSource(ctx, fileSrc)
	if blankErr != nil {
		return d, fmt.Errorf("failed to integrate file source: %w", blankErr)
	}

	// Drain the event from the events channel so users of that interface
	// don't see the intermediate config.
	<-d.Events()

	// If there's a callback to register; register it.
	if params.OnNewConfig != nil {
		// Use a zero-valued token so we ignore any intermediate config values.
		// note: this throws away the unregistration handle.
		d.RegisterCallback(ctx, dials.CfgSerial[T]{}, params.OnNewConfig)
	}

	return d, nil
}

// YAMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to YAML.
func YAMLConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, params Params[T]) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &yaml.Decoder{} }, params)
}

// JSONConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to JSON.
func JSONConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, params Params[T]) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &json.Decoder{} }, params)
}

// CueConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to Cue.
func CueConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, params Params[T]) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &cue.Decoder{} }, params)
}

// TOMLConfigEnvFlag takes advantage of the ConfigWithConfigPath cfg, thinly
// wraping ConfigFileEnvFlag with the decoder statically set to TOML.
func TOMLConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, params Params[T]) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, func(string) dials.Decoder { return &toml.Decoder{} }, params)
}

// DecoderFromExtension is a DecoderFactory that returns an appropriate decoder
// based on the extension of the filename or nil if there is not an appropriate
// mapping.
func DecoderFromExtension(path string) dials.Decoder {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		return &yaml.Decoder{}
	case ".json":
		return &json.Decoder{}
	case ".toml":
		return &toml.Decoder{}
	case ".cue":
		return &cue.Decoder{}
	default:
		return nil
	}
}

// FileExtensionDecoderConfigEnvFlag takes advantage of the
// ConfigWithConfigPath cfg and thinly wraps ConfigFileEnvFlag and and thinly
// wraps ConfigFileEnvFlag choosing the dials.Decoder used when handling the
// file contents based on the file extension (from the limited set of JSON,
// Cue, YAML and TOML).
func FileExtensionDecoderConfigEnvFlag[T any, TP ConfigWithConfigPath[T]](ctx context.Context, cfg TP, params Params[T]) (*dials.Dials[T], error) {
	return ConfigFileEnvFlag(ctx, cfg, DecoderFromExtension, params)
}
