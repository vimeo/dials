# Dials

[![Actions Status](https://github.com/vimeo/dials/workflows/Go/badge.svg)](https://github.com/vimeo/dials/actions)
[![GoDoc](https://godoc.org/github.com/vimeo/dials?status.svg)](https://godoc.org/github.com/vimeo/dials)
[![Go Report Card](https://goreportcard.com/badge/github.com/vimeo/dials)](https://goreportcard.com/report/github.com/vimeo/dials)


Dials is an extensible configuration package for Go.

## Installation

```
go get github.com/vimeo/dials
```

## Prerequisites

Dials requires Go 1.13 or later.

## What is Dials

Dials is a configuration package for Go applications. It supports several different configuration sources including:
 * JSON, YAML, and TOML config files
 * environment variables
 * command line flags (for both Go's [flag](https://golang.org/pkg/flag) package and [pflag](https://pkg.go.dev/github.com/spf13/pflag) package)
 * watched config files and re-reading when there are changes to the watched files
 * default values

## Why choose Dials
Dials is a configuration solution that supports several configuration sources so you only have to focus on the business logic.
Define the configuration struct and select the configuration sources and Dials will do the rest. Moreover, setting defaults doesn't require additional function calls.
Just populate the config struct with the default values and pass the struct to Dials. 
Dials also allows the flexibility to choose the precedence order to determine which sources can overwrite the configuration values. Additionally, Dials has special handling of structs that implement [`encoding.TextUnmarshaler`](https://golang.org/pkg/encoding/#TextUnmarshaler) so structs (like [`IP`](https://pkg.go.dev/net?tab=doc#IP) and [`time`](https://pkg.go.dev/time?tab=doc#Time)) can be properly parsed.

## Using Dials

### Reading from config files, environment variables, and command line flags
Dials requires very minimal configuration to populate values from several sources through the `ez` package. Define the config struct and provide a method `ConfigPath() (string, bool)` to indicate the path to the config file. The package defines several functions that help populate the config struct by reading from config files,
environment variables, and command line flags

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
	// the dials tag can be used as alias so when the name in the config file
	// changes, the code doesn't have to change.
	Val2 int `dials:"val_2"`
	// Dials follows the Go convention for flags and will look for the dials
	// tag or field name in lower-kebab-case. In this case, it would look for
	// val-3 flag. To specify a different flag name, use the `dialsflag` tag.
	// Now, Dials will lookup "some-val" flag instead. Since only
	// `dialsflag` tag is specified, Val3 will only be populated from command
	// line flags
	Val3 bool `dialsflag:"some-val"`
	// Path holds the value of the path to the config file. Dials follows the
	// Go convention and will look for the dials tag or field name in all caps. 
	// In this case, it would lookup the PATH environment variable. To specify
	// a different env variable, use the `dials_env` tag. Now Dials will lookup
	// "configpath" env value to populate the Path field
	Path string `dials_env:"configpath"`
}

// ConfigPath returns the path to the config file. This is particularly helpful
// when the path is populated from environment variables or command line flags.
// Dials will first read from environment variables and command line flags
// and then read the config file specified by the populated field.
func (c *Config) ConfigPath() (string, bool) {
	// can alternatively return empty string and false if the state of the
	// struct doesn't specify a config file to read
	return c.Path, true
}

func main() {
	c := &Config{}

	// The following function will populate the config struct by reading the
	// config files, environment variables, and command line flags (order matches
	// the function name) with increasing precedence. In other words, the flag
	// source (last) would overwrite the YAML source (first) were they both to
	// attempt to set the same struct field. The boolean argument passed to the
	// function indicates whether the file will be watched and updates to the
	// file should update the config struct.
	view, dialsErr := ez.YAMLConfigEnvFlag(context.Background(), c, false)
	if dialsErr != nil {
		// error handling
	}

	// Get an interface corresponding to the filled-out config struct, and
	// assert it to the correct type. Here's the struct populated from config file,
	// environment variables, and command line flags.
	Config := view.View().(*Config)
	fmt.Printf("Config: %+v\n", Config)
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

Note that even though val_2 has a value of 2 in the yaml file, the config value 
output for Val2 is 5 because environment variables take precedence.

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
	"github.com/vimeo/dials/env"
	"github.com/vimeo/dials/file"
	"github.com/vimeo/dials/flag"
	"github.com/vimeo/dials/yaml"
)

type Config struct {
	Val1 string `dials:"Val1" yaml:"b"`
	// The `dials_env` tag is used for environment values. The Val2 value will
	// only be populated from environment variables. If you want several different
	// sources to be able to set this value, use the `dials` tag instead
	Val2 int `dials_env:"VAL_2"`
	// the `dialsflag` tag is used for command line flag values. The Val3 value
	// will only be populated from command line flags
	Val3 bool `dialsflag:"val-3"`
}

func main() {
	config := &Config{
		// Val1 has a default value of "hello" and will be overwritten by the
		// sources if there is a corresponding field for Val1
		Val1: "hello",
	}

	// To read from other source files such as JSON, and TOML, use
	// "&json.Decoder{}" or "&toml.Decoder{}"
	fileSrc, fileErr := file.NewSource("path/to/config", &yaml.Decoder{})
	if fileErr != nil {
		// error handling
	}

	// Define a `dials.Source` for command line flags. Consider using the dials pflag library
	// if the application uses the spf13/pflag package
	flagSet, flagErr := flag.NewCmdLineSet(flag.DefaultFlagNameConfig(), config)
	if flagErr != nil {
		// error handling
	}

	// use the environment source to get values from environment variables
	envSrc := &env.Source{}

	// the Config struct will be populated in the order in which the sources are
	// passed in the Config function with increasing precedence. So the fileSrc value
	// will overwrite the flagSet value if they both were to set for the same field
	view, err := dials.Config(context.Background(), config, envSrc, flagSet, fileSrc)
	if err != nil {
		// error handling
	}

	// Config holds the populated config struct
	Config := view.View().(*Config)
	fmt.Printf("Config: %+v\n", Config)
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
 
the output will be `Config: &{Val1:valueb Val2:5 Val3:true}`. Note that even when val-3 is defined in the yaml file and the file source takes precedence, 
only the value from command line flag populates the config due to the special `dialsflag` tag. If you prefer the value for `Val3` be overwritten by other sources, then
use the `dials` tag instead of the `dialsflag` tag



### Watching file source
If you wish to watch the config file and make updates to your configuration, use the watching source. This functionality is available in the `ez` package by passing the `true` boolean to the functions. The `WatchingSource` can be used when you want to further customize the configuration as well.

``` go
	// NewWatchSource also has watch options that can be passed to have the
	// ability to use a ticker for polling, set a logger, and more
	watchingFileSource, fsErr := file.NewWatchingSource(
		"path/to/config", &yaml.Decoder{})

	if fsErr != nil {
		// error handling
		return
	}

	// additional sources can be passed along with the watching file source and the
	// precedence order will still be dictated by the order in which the sources are
	// defined in the Config function.
	view, err := dials.Config(context.Background(), config, watchingFileSource)
	if err != nil {
		// error handling
	}

	Config := view.View().(*Config)
```

### Source
Source interface is implemented by different configuration sources that populate the configuration struct. Dials currently supports environment variables, command line flags, and config file sources. When `dials.Config` function is going through the different sources to extract the values, it calls the `Value` method on each of these sources. This allows for the logic of the Source to be encapsulated while giving the application access to the values populated by each Source.


### Decoder
Decoders are modular allowing users to mix and match Decoders and Sources. Dials currently supports Decoders that decode different data formats (JSON, YAML, and TOML) and insert the values into the appropriate fields in the config struct. Decoders can be expanded from that use case and users can write their own Decoders to perform the tasks they like (more info in the section below). Decoder is called when the supported Source calls the `Decode` method to unmarshal the data into the config struct and returns the populated struct. There are two sources that the Decoders can be used with: files (including watched files) and `static.StringSource`.

### Write your own Source and Decoder
If you wish to define your own source, implement the `Source` interface and pass the source to the `dials.Config` function. If you want the Source to interact with a Decoder, call `Decode` in the `Value` method of the Source. If you want to define your own Decoder to interact with your Source, implement the `Decoder` interface. Since Decoders are modular, keep the logic of Decoder encapsulated and separate from the Source. For example, you can have an `HTTP` Source and custom `Get` and `Post` Decoders that repeatedly make `GET` and `POST` requests. The user can use the HTTP Source with either one of the Get or Post decoders and potentially also use the `POST` Decoder with other Sources.

### Putting it all together

The `dials.Config` function first makes a deep copy of the configuration struct and makes each field a pointer (even the fields in nested structs) with special handling for structs that implement [`encoding.TextUnmarshaler`](https://golang.org/pkg/encoding/#TextUnmarshaler). Then it calls the `Value` method on each Source and stores the returned value. The final step is to to compose the final config struct by overlaying the values from all the different Sources and accounting for the precedence order. Since the fields are pointers, we can directly assign pointers while overlaying. Overlay even has safety checks for deduplicating maps sharing a backing pointer and for structs with self-referential pointers. So when you write your own Source, you just have to pass the Source in to the `dials.Config` function and Dials will take care of deep copying and pointerifying the struct and composing the final struct with overlay.


## Contributors
Dials is a production of Vimeo's Core Services team
* [@sergiosalvatore](https://github.com/sergiosalvatore)
* [@dfinkel](https://github.com/dfinkel)
* [@sachinagada](https://github.com/sachinagada)

