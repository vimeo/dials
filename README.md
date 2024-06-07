# Dials

[![Actions Status](https://github.com/vimeo/dials/workflows/Go/badge.svg)](https://github.com/vimeo/dials/actions)
[![GoDoc](https://godoc.org/github.com/vimeo/dials?status.svg)](https://godoc.org/github.com/vimeo/dials)
[![Go Report Card](https://goreportcard.com/badge/github.com/vimeo/dials)](https://goreportcard.com/report/github.com/vimeo/dials)


Dials is an extensible configuration package for Go.

## Installation

```
go get github.com/vimeo/dials@latest
```

## Prerequisites

Dials requires Go 1.18 or later.

## What is Dials?

Dials is a configuration package for Go applications. It supports several different configuration sources including:
 * [Cue](https://cuelang.org), JSON, YAML, and TOML config files
 * environment variables
 * command line flags (for both Go's [flag](https://golang.org/pkg/flag) package and [pflag](https://pkg.go.dev/github.com/spf13/pflag) package)
 * watched config files and re-reading when there are changes to the watched files
 * default values

## Why choose Dials?
Dials is a configuration solution that supports several configuration sources so you only have to focus on the business logic.
Define the configuration struct and select the configuration sources and Dials will do the rest. Dials is designed to be extensible so if the built-in sources don't meet your needs, you can write your own and still get all the other benefits. Moreover, setting defaults doesn't require additional function calls.
Just populate the config struct with the default values and pass the struct to Dials.
Dials also allows the flexibility to choose the precedence order to determine which sources can overwrite the configuration values. Additionally, Dials has special handling of structs that implement [`encoding.TextUnmarshaler`](https://golang.org/pkg/encoding/#TextUnmarshaler) so structs (like [`IP`](https://pkg.go.dev/net?tab=doc#IP) and [`time`](https://pkg.go.dev/time?tab=doc#Time)) can be properly parsed.

## Using Dials

### Reading from config files, environment variables, and command line flags
Dials requires very minimal configuration to populate values from several sources through the `ez` package. Define the config struct and provide a method `ConfigPath() (string, bool)` to indicate the path to the config file. The package defines several functions that help populate the config struct by reading from config files,
environment variables, and command line flags.

``` go
package main

import (
	"context"
	"fmt"

	"github.com/vimeo/dials/ez"
)

// Config is the configuration struct needed for the application
type Config struct {
	// When a struct tag more specifically corresponding to a source is
	// present, it takes precedence over the `dials` tag. Note that just
	// because there is a `yaml` struct tag present doesn't mean that other
	// sources can't fill this field.
	Val1 string `dials:"Val1" yaml:"b"`
	// the dials tag can be used as an alias so when the name in the config file
	// changes, the code doesn't have to change.
	Val2 int `dials:"val_2"`
	// Dials will register a flag with the name matching the dials tag.
	// Without any struct tags, dials will decode the go camel case field name
	// and encode it using lower-kebab-case and use the encoded name as the flag
	// name (ex: val-3). To specify a different flag name, use the `dialsflag`
	// tag. Now, Dials will register a flag with "some-val" name instead.
	// The `dialsdesc` tag is used to provide help message for the flag.
	Val3 bool `dialsflag:"some-val" dialsdesc:"enable auth"`
	// Path holds the value of the path to the config file. Dials follows the
	// *nix convention for environment variables and will look for the dials tag
	// or field name in all caps when struct tags aren't specified. Without any
	// struct tags, it would lookup the PATH environment variable. To specify a
	// different env variable, use the `dialsenv` tag. Now Dials will lookup
	// "configpath" env value to populate the Path field.
	Path string `dialsenv:"configpath"`
}

// ConfigPath returns the path to the config file that Dials should read. This 
// is particularly helpful when it's desirable to specify the file's path via
// environment variables or command line flags. Dials will first populate the 
// configuration struct from environment variables and command line flags
// and then read the config file that the ConfigPath() method returns
func (c *Config) ConfigPath() (string, bool) {
	// can alternatively return empty string and false if the state of the
	// struct doesn't specify a config file to read. We would recommend to use
	// dials.Config directly (shown in the next example) instead of the ez
	// package if you just want to use environment variables and flags without a
	// file source
	return c.Path, true
}

func main() {
	defCfg := Config{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The following function will populate the config struct by reading the
	// config files, environment variables, and command line flags (order matches
	// the function name) with increasing precedence. In other words, the flag
	// source (last) would overwrite the YAML source (first) were they both to
	// attempt to set the same struct field. There are several options that can be
	// passed in as well to indicate whether the file will be watched and updates
	// to the file should update the config struct and if the flags name component
	// separation should use different encoding than lower-kebab-case
	d, dialsErr := ez.YAMLConfigEnvFlag(ctx, &defCfg, ez.Params[Config]{WatchConfigFile: true})
	if dialsErr != nil {
		// error handling
	}

	// View returns a pointer to the fully stacked configuration file
	// The stacked configuration is populated from the config file, environment
	// variables and commandline flags.
	cfg := d.View()
	fmt.Printf("Config: %+v\n", cfg)
}
```

For reading from JSON or TOML config files along with environment variables and command line flags,
use the `ez.JSONConfigEnvFlag` or `ez.TOMLConfigEnvFlag` functions.

If the above code is run with the following YAML file:

``` yaml
b: valueb
val_2: 2
val-3: false
```

and the following command (make sure to change the configpath value to point to your path)

```
export configpath=path/to/config/file
export VAL_2=5
go run main.go --some-val
```

the output will be
`Config: &{Val1:valueb Val2:5 Val3:true}`

Note that even though `val_2` has a value of 2 in the yaml file, the config value
output for `Val2` is 5 because environment variables take precedence.

### Configure your configuration settings
If the predefined functions in the ez package don't meet your needs, you can specify the
sources you would like in the order of precedence you prefer. Not much setup is needed
to configure this. Choose the predefined sources and add the appropriate `dials` tags to the config struct.

``` go
package main

import (
	"context"
	"fmt"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/env"
	"github.com/vimeo/dials/sources/file"
	"github.com/vimeo/dials/sources/flag"
	"github.com/vimeo/dials/decoders/yaml"
)

type Config struct {
	Val1 string `dials:"Val1" yaml:"b"`
	// The `dialsenv` tag is used to override the name used by the environment
	// source. If you want to use a single, consistent name across several
	// sources, set the `dials` tag instead
	Val2 int `dialsenv:"VAL_2"`
	// the `dialsflag` tag is used for command line flag values and the dialsdesc
	// tag provides the flag help text
	Val3 bool `dialsflag:"val-3" dialsdesc:"maximum number of idle connections to DB"`
}

func main() {
	defaultConfig := &Config{
		// Val1 has a default value of "hello" and will be overwritten by the
		// sources if there is a corresponding field for Val1
		Val1: "hello",
	}

	// Define a file source if you want to read from a config file. To read
	// from other source files such as Cue, JSON, and TOML, use "&cue.Decoder{}",
    // "&json.Decoder{}" or "&toml.Decoder{}"
	fileSrc, fileErr := file.NewSource("path/to/config", &yaml.Decoder{})
	if fileErr != nil {
		// error handling
	}

	// Define a `dials.Source` for command line flags. Consider using the dials
	// pflag library if the application uses the github.com/spf13/pflag package
	flagSet, flagErr := flag.NewCmdLineSet(flag.DefaultFlagNameConfig(), defaultConfig)
	if flagErr != nil {
		// error handling
	}

	// use the environment source to get values from environment variables
	envSrc := &env.Source{}

	// the Config struct will be populated in the order in which the sources are
	// passed in the Config function with increasing precedence. So the fileSrc
	// value will overwrite the flagSet value if they both were to set the
	// same field
	d, err := dials.Config(context.Background(), defaultConfig, envSrc, flagSet, fileSrc)
	if err != nil {
		// error handling
	}

	// View returns a pointer to the fully stacked configuration.
	// The stacked configuration is populated from the config file, environment
	// variables and commandline flags. Can alternatively use
	cfg := d.View()
	fmt.Printf("Config: %+v\n", cfg)
}
```

If the above code is run with the following YAML file (make sure to change the path in the code):

``` yaml
b: valueb
val-3: false
```

and the following commands

``` 
export VAL_2=5
go run main.go --val-3
```

the output will be `Config: &{Val1:valueb Val2:5 Val3:true}`.

Note that even when val-3 is defined in the yaml file and the file source takes precedence,
only the value from command line flag populates the config due to the special `dialsflag` tag. The `val-3` name is only used by the flag source. The file source will still use the field name. You can update the yaml file to `val3: false` to have the file source overwrite the field. Alternatively, we recommend using the `dials` tag to have consistent naming across all sources.


### Watching file source
If you wish to watch the config file and make updates to your configuration, use the watching source. This functionality is available in the `ez` package by using the `WithWatchingConfigFile(true)` option (the default is false). The `WatchingSource` can be used when you want to further customize the configuration as well. Please note that the Watcher interface is likely to change in the near future.

``` go
	// NewWatchSource also has watch options that can be passed to use a ticker
	// for polling, set a logger, and more
	watchingFileSource, fsErr := file.NewWatchingSource(
		"path/to/config", &yaml.Decoder{})

	if fsErr != nil {
		// error handling
		return
	}

	// Use a non-background context with the WatchingSource to prevent goroutine leaks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	// additional sources can be passed along with the watching file source and the
	// precedence order will still be dictated by the order in which the sources are
	// defined in the Config function.
	d, err := dials.Config(ctx, defCfg, watchingFileSource)
	if err != nil {
		// error handling
	}

	conf, serial := d.ViewVersion()

	// You can get notified whenever the config changes by registering a callback.
	// If a new version has become available since the serial argument was
	// returned by ViewVersion(), it will be called immediately to bring
	// the callback up to date.
	// You can call the returned unregister function to unregister at a later point.
	unreg := d.RegisterCallback(ctx, serial, func(ctx context.Context, oldCfg, newCfg *Config) {
		// log, increment metrics, re-index a map, etc.
	})
	// If the passed context expires before registration succeeds, a nil
	// unregister callback will be returned.
	if unreg != nil {
		defer unreg()
	}
```

### Flags
When setting commandline flags using either the pflag or flag sources, additional flag-types become available for simple slices and maps.

#### Slices

Slices of integer-types get parsed as comma-separated values using Go's parsing rules (with whitespace stripped off each component)
e.g. `--a=1,2,3` parses as `[]int{1,2,3}`

Slices of strings get parsed as comma-separated values if the individual values are alphanumeric, and must be quoted in conformance with Go's [`strconv.Unquote`](https://pkg.go.dev/strconv#Unquote) for more complicated values
e.g. `--a=abc` parses as `[]string{"abc"}`, `--a=a,b,c` parses as `[]string{"a", "b", "c"}`, while `--a="bbbb,ffff"` has additional quoting (ignoring any shell), so it becomes `[]string{"bbbb,ffff"}`

Slice-typed flags may be specified multiple times, and the values will be concatenated.
As a result, a commandline with `"--a=b", "--a=c"` may be parsed as `[]string{b,c}`.

#### Maps
Maps are parsed like Slices, with the addition of `:` separators between keys and values. ([`strconv.Unquote`](https://pkg.go.dev/strconv#Unquote)-compatible quoting is mandatory for more complicated strings as well)

e.g. `--a=b:c` parses as `map[string]string{"b": "c"}`

### Source
The Source interface is implemented by different configuration sources that populate the configuration struct. Dials currently supports environment variables, command line flags, and config file sources. When the `dials.Config` method is going through the different `Source`s to extract the values, it calls the `Value` method on each of these sources. This allows for the logic of the Source to be encapsulated while giving the application access to the values populated by each Source. Please note that the Value method on the Source interface and the Watcher interface are likely to change in the near future.


### Decoder
Decoders are modular, allowing users to mix and match Decoders and Sources. Dials currently supports Decoders that decode different data formats (JSON, YAML, and TOML) and insert the values into the appropriate fields in the config struct. Decoders can be expanded from that use case and users can write their own Decoders to perform the tasks they like (more info in the section below).

Decoder is called when the supported Source calls the `Decode` method to unmarshal the data into the config struct and returns the populated struct. There are two sources that the Decoders can be used with: files (including watched files) and `static.StringSource`. Please note that the Decoder interface is likely to change in the near future.

### Write your own Source and Decoder
If you wish to define your own source, implement the `Source` interface and pass the source to the `dials.Config` function. If you want the Source to interact with a Decoder, call `Decode` in the `Value` method of the Source.

Since Decoders are modular, keep the logic of Decoder encapsulated and separate from the Source. `Source` and `Decoder` implementations should be orthogonal and `Decoder`s should not be `Source` specific. For example, you can have an `HTTP` or `File` Source that can interact with the `JSON` decoder to unmarshal the data to a struct.

### Putting it all together

1. The `dials.Config` function makes a deep copy of the configuration struct and makes each field a pointer (even the fields in nested structs) with special handling for structs that implement [`encoding.TextUnmarshaler`](https://golang.org/pkg/encoding/#TextUnmarshaler).
2. Call the `Value` method on each Source and stores the returned value.
3. The final step is to to compose the final config struct by overlaying the values from all the different Sources and accounting for the precedence order. Since the fields are pointers, we can directly assign pointers while overlaying. Overlay even has safety checks for deduplicating maps sharing a backing pointer and for structs with self-referential pointers.

So when you write your own Source, you just have to pass the Source in to the `dials.Config` function and Dials will take care of deep copying and pointerifying the struct and composing the final struct with overlay.


## Contributors
Dials is a production of Vimeo's Core Services team
* [@sergiosalvatore](https://github.com/sergiosalvatore)
* [@dfinkel](https://github.com/dfinkel)
* [@sachinagada](https://github.com/sachinagada)

