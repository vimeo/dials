package integrationtests_test

import (
	"context"

	"fmt"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/decoders/json"
	"github.com/vimeo/dials/sources/static"
)

type Protocol int

const (
	HTTP Protocol = iota
	HTTPS
	Gopher
	FTP
)

func (p Protocol) DialsValueMap() map[string]Protocol {
	return map[string]Protocol{
		"http":   HTTP,
		"https":  HTTPS,
		"gopher": Gopher,
		"ftp":    FTP,
	}
}

func (p Protocol) String() string {
	switch p {
	case HTTP:
		return "HyperText Transfer Protocol"
	case HTTPS:
		return "Secure HyperText Transfer Protocol"
	case Gopher:
		return "Gopher"
	case FTP:
		return "File Transfer Protocol"
	}
	return "unknown"
}

type EnumConfig struct {
	SelectedProto dials.Enum[Protocol]
	Backup        dials.Enum[Protocol]
}

func ExampleEnum() {

	source := &static.StringSource{
		Data:    `{"selectedProto": "gopher"}`,
		Decoder: &json.Decoder{},
	}

	c := &EnumConfig{
		Backup: dials.EnumValue(HTTPS),
	}

	d, err := dials.Config(context.Background(), c, source)
	if err != nil {
		panic("something went wrong!")
	}

	v := d.View()
	fmt.Printf("selected: %s, backup: %s", v.SelectedProto.Value, v.Backup.Value)
	// Output: selected: Gopher, backup: Secure HyperText Transfer Protocol
}
