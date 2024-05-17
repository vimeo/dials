package jsontypes

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func ptrVal[V any](v V) *V {
	return &v
}

func TestParsingDuration(t *testing.T) {
	t.Parallel()
	type decStruct struct {
		Dur    ParsingDuration
		DurPtr *ParsingDuration
	}
	for _, tbl := range []struct {
		name      string
		inJSON    string
		expStruct any
		expErr    bool
	}{
		{
			name:      "integer_value_no_ptr",
			inJSON:    `{"dur": 1234}`,
			expStruct: decStruct{Dur: 1234},
			expErr:    false,
		},
		{
			name:      "string_value_no_ptr",
			inJSON:    `{"dur": "3s"}`,
			expStruct: decStruct{Dur: ParsingDuration(3 * time.Second)},
			expErr:    false,
		},
		{
			name:      "string_value_ptr",
			inJSON:    `{"dur": "3s", "durptr": "9m"}`,
			expStruct: decStruct{Dur: ParsingDuration(3 * time.Second), DurPtr: ptrVal(ParsingDuration(9 * time.Minute))},
			expErr:    false,
		},
		{
			name:      "integer_value_ptr",
			inJSON:    `{"dur": "3s", "durptr": 2048}`,
			expStruct: decStruct{Dur: ParsingDuration(3 * time.Second), DurPtr: ptrVal(ParsingDuration(2048))},
			expErr:    false,
		},
		{
			name:   "error_array_val",
			inJSON: `{"dur": [], "durptr": 2048}`,
			expErr: true,
		},
		{
			name:   "error_object_val",
			inJSON: `{"dur": {}, "durptr": 2048}`,
			expErr: true,
		},
		{
			name:   "error_unparsable_str_val",
			inJSON: `{"dur": "sssssssssss", "durptr": 2048}`,
			expErr: true,
		},
		{
			name:   "error_float_fractional_val",
			inJSON: `{"dur": 0.333333, "durptr": 2048}`,
			expErr: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			v := decStruct{}
			decErr := json.Unmarshal([]byte(tbl.inJSON), &v)
			if decErr != nil {
				if !tbl.expErr {
					t.Errorf("unexpected error unmarshaling: %s", decErr)
				} else {
					t.Logf("expected error: %s", decErr)
				}
				return
			}
			if !reflect.DeepEqual(v, tbl.expStruct) {
				t.Errorf("unexpected value\n got: %+v\nwant:%+v", tbl.expStruct, v)
			}
		})

	}
}
