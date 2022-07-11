package sourcewrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/static"
	"github.com/vimeo/dials/transform"
)

type trivialJSONDecoder struct{}

func (tjd *trivialJSONDecoder) Decode(r io.Reader, dt *dials.Type) (reflect.Value, error) {
	jsonBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("error reading JSON: %s", err)
	}

	val := reflect.New(dt.Type()).Elem()

	instance := val.Addr().Interface()
	err = json.Unmarshal(jsonBytes, instance)
	if err != nil {
		return reflect.Value{}, err
	}

	return val, nil
}

func TestTransformingDecoder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type conf struct {
		Set map[string]struct{}
	}

	c := conf{}

	ss := static.StringSource{
		Data: `{"Set": ["a", "b", "c"]}`,
		Decoder: NewTransformingDecoder(
			&trivialJSONDecoder{},
			&transform.SetSliceMangler{},
		),
	}

	d, err := dials.Config(ctx, &c, &ss)
	require.NoError(t, err)

	theConf := d.View()
	assert.Len(t, theConf.Set, 3)
	assert.Contains(t, theConf.Set, "a")
	assert.Contains(t, theConf.Set, "b")
	assert.Contains(t, theConf.Set, "c")
}
