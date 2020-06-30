# Dials

[![Actions Status](https://github.com/vimeo/dials/workflows/Go/badge.svg)](https://github.com/vimeo/dials/actions)
[![GoDoc](https://godoc.org/github.com/vimeo/dials?status.svg)](https://godoc.org/github.com/vimeo/dials)
[![Go Report Card](https://goreportcard.com/badge/github.com/vimeo/dials)](https://goreportcard.com/report/github.com/vimeo/dials)


Dials is a no-BS configuration package for Go projects.

## Installation

```
go get github.com/vimeo/dials
```

## Prerequisites

Dials requires Go 1.13 or later.

## What is Dials

Dials is a configuration package for Go applications. It supports several different formats including:
 * reading from JSON, YAML, and TOML config files
 * getting values from environement variables 
 * parsing command line flags (for both Go's flag package and pflag pacakge)
 * watching config files and re-reading when there are changes to the watched files
 * setting default values

## Why choose Dials
Dials is a configuration solution that supports several formats so you only have to focus on the business logic. 
Define the configuration struct and select the formats and Dials will do the rest. Moreover, setting defaults doesn't require additional function calls. 
Just populate the config struct with the default values and pass the struct to Dials. 
Dials also allows the flexibility to choose the precedence order to determine which formats can override the configuration values. 

## Using Dials

### Reading from config files, environment variables, and command line flags
Dials requires very minimal configuration to populate values from several formats through the `ez` package. Define the config struct and provide a method `ConfigPath() (string, bool)` to indicate which file to read as the config file. The package defines several functions that help populate the config struct by reading from config files,
environment variables, and command line flags

``` go
type Config struct {
	// When a struct tag more specifically corresponding to a source is
	// present, it takes precedence over the `dials` tag. Note that just
	// because there is a `yaml` struct tag present doesn't mean that other
	// sources can't fill this field.
    Val1 string `dials:"Val1" yaml:"b"`
    // the dials tag can be used as alias so when the config file name changes, 
    // the code doesn't have to change
    Val2 int    `dials:"val_2"` 
    // the `dialsflag` tag is used for command line flag values. The Val3 value
    //  will only be populated from command line flags
	Val3 bool `dialsflag:"val-3"`
}

// ConfigPath method indicates which file to read as the config file
func (c *Config) ConfigPath() (string, bool) {
	// can alternatively return empty string and false
	return "path/to/config", true
}

func main() {
	y := &Config{}

    // The following function will populate the config struct by reading the 
    // config files, environment variables, and command line flags in that order
    // with increasing precedence. In other words, the flag source (last) would
	// overwrite the YAML source (first) were they both to attempt to set the 
    // same struct field. The boolean argument passed to the function indicates 
    // whether the file will be watched and updates to the file should update
    // the config struct
	view, dialsErr := ez.YAMLConfigEnvFlag(context.Background(), y, false)
	if dialsErr != nil {
		// error handling
	}

    // Get an interface corresponding to the filled-out config struct, and
    // case it to the correct type. Here's your struct populated from config file, 
    // environment variables, and command line flags in that order.
    Config := view.View().(*Config)
    fmt.Printf("Config :%+v", Config)
}
```

For reading from JSON or TOML config files along with environment variables and command line flags,
use the `ez.JSONConfigEnvFlag` or `ez.TOMLConfigEnvFlag` functions.

If the above code is run with the following YAML file (make sure to change the path in the code):

``` yaml
b: valueb
val_2: 2
val-3: false
```

and the following command 

```
export val_2=5
go run main.go --val-3
```

the output will be 
`Config: &{Val1:valueb Val2:5 Val3:true}`

Note that even though val_2 has a value of 2 in the yaml file, the config value 
output for Val2 is 5 because environment variables take precedence

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
	// When a struct tag more specifically corresponding to a source is
	// present, it takes precedence over the `dials` tag. Note that just
	// because there is a `yaml` struct tag present doesn't mean that other
	// sources can't fill this field.
	Val1 string `dials:"Val1" yaml:"b"`
    // The `dials_env` tag is used for environment values. The Val2 value will
    //  only be populated from environment variables. If you want several different
    // sources to be able to set this value, use the `dials` tag instead
	Val2 int `dials_env:"VAL_2"`
    // the `dialsflag` tag is used for command line flag values. The Val3 value
    //  will only be populated from command line flags
	Val3 bool `dialsflag:"val-3"`
}

func main() {
	config := &Config{
		// Val1 has a default value of "hello" and will be over written by the
		// config file if there is a corresponding field for Val1
		Val1: "hello",
	}

	// To read from other source files such as JSON, and TOML, use
	// "&json.Decoder{}" or "&toml.Decoder{}"
	fileSrc, fileErr := file.NewSource("path/to/config", &yaml.Decoder{})
	if fileErr != nil {
		// error handling
	}

    // Define a `dials.Source` for command line flags. Consider using the dials pflag library
    // if the application uses the spf13/pflags package
	flagSet, flagErr := flag.NewCmdLineSet(flag.DefaultFlagNameConfig(), config)
	if flagErr != nil {
		// error handling
	}

	// use the environment source to get values from environment variables
	envSrc := &env.Source{}

	// the Config struct will be populated in the order in which the sources are
	// passed in the Config function with increasing precedence. So the fileSrc value
	// will override the flagSet value if they both were to set for the same field
	view, err := dials.Config(context.Background(), config, envSrc, flagSet, fileSrc)
	if err != nil {
		// error handling
	}

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
only the value from command line flag populates the config due to the special `dialsflag` tag

### Watching file source
If you wish to watch the config file and make updates to your configuration, use the watching source.

``` go
    // NewWatchSource has watch options that can be passed in as well to have the
	// ability to use a ticker for polling, set a logger, and more
	watchingFileSource, fsErr := file.NewWatchingSource(
		"path/to/config", &yaml.Decoder{})

	if fsErr != nil {
		// error handling
		return
	}

	// additional sources can be passed along with the watching file source and the
	// precedence order will still be dictated by the order in which the sources are
	// defined in the Config function
	view, err := dials.Config(context.Background(), config, watchingFileSource)
	if err != nil {
		// error handling
	}

	Config := view.View().(*Config)
```

## Contributors
Pentagon is a production of Vimeo's Core Services team
* [@sergiosalvatore](https://github.com/sergiosalvatore)
* [@dfinkel](https://github.com/dfinkel)
* [@sachinagada](https://github.com/sachinagada)

