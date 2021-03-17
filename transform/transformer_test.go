package transform

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMangler struct {
	fieldNameMods map[string][]reflect.StructField
	recurseField  map[string]bool
	origFieldVals map[string]interface{}
	// A couple maps to let tests return errors for certain fields.
	unmangleErrs map[string]error
	mangleErrs   map[string]error
}

// Mangle is called for every field in a struct, and maps that to one or more output fields.
func (f *fakeMangler) Mangle(field reflect.StructField) ([]reflect.StructField, error) {
	if f.mangleErrs != nil {
		if mErr, ok := f.mangleErrs[field.Name]; ok {
			return nil, mErr
		}

	}
	if sfs, ok := f.fieldNameMods[field.Name]; ok {
		return sfs, nil
	}
	return []reflect.StructField{field}, nil

}

// Unmangle is called for every source-field->mangled-field
// mapping-set, with the mangled-field and its populated value set.
func (f *fakeMangler) Unmangle(origField reflect.StructField, mangledFieldVals []FieldValueTuple) (reflect.Value, error) {
	if f.unmangleErrs != nil {
		if mErr, ok := f.unmangleErrs[origField.Name]; ok {
			return reflect.Value{}, mErr
		}

	}
	return reflect.ValueOf(f.origFieldVals[origField.Name]), nil
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
func (f *fakeMangler) ShouldRecurse(field reflect.StructField) bool {
	return f.recurseField[field.Name]
}

type nameVal struct {
	name string
	val  interface{}
}

func strPtr(in string) *string {
	return &in
}

func TestTransformer(t *testing.T) {
	intType := reflect.TypeOf(int(1))
	strType := reflect.TypeOf(string(""))
	for _, itbl := range []struct {
		name                        string
		inStruct                    interface{}
		unmangleVal                 interface{}
		fm                          fakeMangler
		expectedMangledFieldNames   []string
		expectedUnmangledNameValues []nameVal
		expectedErr                 *string
		unexported                  int
	}{
		{
			name:        "unassignable_final_type",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{ F int }{F: 42},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					// return a bool rather than an int
					"I": bool(true),
				},
				// mark unmangledErrs as non-empty so the test
				// knows to expect an error on unmangle.
				unmangleErrs: map[string]error{"foobar": nil},
			},
			expectedMangledFieldNames: []string{"F"},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  34,
			}},
			expectedErr: strPtr("incompatible types for field \"I\"; original field type int; final unmangled type bool"),
		}, {
			name:        "simple_mangle_error",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{ F int }{F: 42},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					// return a bool rather than an int
					"I": bool(true),
				},
				mangleErrs: map[string]error{"I": errors.New("fimbat")},
			},
			expectedMangledFieldNames:   nil,
			expectedUnmangledNameValues: nil,
			expectedErr:                 strPtr("failed to mangle field 0 with mangler 0 (type *transform.fakeMangler): fimbat"),
		}, {
			name:        "simple_unmangle_error",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{ F int }{F: 42},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					"I": int(538),
				},
				unmangleErrs: map[string]error{"I": errors.New("fimbaz")},
			},
			expectedMangledFieldNames:   []string{"F"},
			expectedUnmangledNameValues: nil,
			expectedErr:                 strPtr("failed to unmangle field 0 (\"I\") with mangler 0 (type *transform.fakeMangler): unmangle from mangler 0 (type *transform.fakeMangler) failed: fimbaz"),
		}, {
			name:        "one_int_change_field_name",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{ F int }{F: 42},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					"I": 34,
				},
			},
			expectedMangledFieldNames: []string{"F"},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name:        "one_int_delete_field",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{}{},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					"I": 34,
				},
			},
			expectedMangledFieldNames: []string{},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name: "one_int_delete_field_one_unexported",
			inStruct: struct {
				I int
				z bool
			}{I: 42},
			unexported:  1,
			unmangleVal: struct{}{},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					"I": 34,
				},
			},
			expectedMangledFieldNames: []string{},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name:        "one_int_expand_two",
			inStruct:    struct{ I int }{I: 42},
			unmangleVal: struct{ F, Q int }{F: 42, Q: 88},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{},
				origFieldVals: map[string]interface{}{
					"I": 34,
				},
			},
			expectedMangledFieldNames: []string{"F", "Q"},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_norecurse_flatten",
			inStruct: struct {
				J struct{ I int }
				I int
			}{J: struct{ I int }{I: 128}, I: 42},
			unmangleVal: struct{ J, Q int }{J: 1023, Q: 88},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bizzle",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{"J": false},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": struct{ I int }{I: 255},
				},
			},
			expectedMangledFieldNames: []string{"J", "Q"},
			expectedUnmangledNameValues: []nameVal{{
				name: "J",
				val:  struct{ I int }{I: 255},
			}, {
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name:        "one_int_slice_to_int",
			inStruct:    struct{ I []int }{I: []int{42}},
			unmangleVal: struct{ F int }{F: 42},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      intType,
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// recurse into "I"
				recurseField: map[string]bool{"I": true},
				origFieldVals: map[string]interface{}{
					"I": []int{34},
				},
			},
			expectedMangledFieldNames: []string{"F"},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  []int{34},
			}},
			expectedErr: nil,
		}, {
			name:        "one_int_slice_to_int_slice",
			inStruct:    struct{ I []int }{I: []int{42}},
			unmangleVal: struct{ F []int }{F: []int{42}},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "F",
							PkgPath:   "",
							Type:      reflect.SliceOf(intType),
							Tag:       "fimbat",
							Anonymous: false,
						},
					},
				},
				// recurse into "I"
				recurseField: map[string]bool{"I": true},
				origFieldVals: map[string]interface{}{
					"I": []int{34},
				},
			},
			expectedMangledFieldNames: []string{"F"},
			expectedUnmangledNameValues: []nameVal{{
				name: "I",
				val:  []int{34},
			}},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_recurse_flatten",
			inStruct: struct {
				J struct{ I int }
				I int
			}{J: struct{ I int }{I: 128}, I: 42},
			unmangleVal: struct{ J, Q int }{J: 1023, Q: 88},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bizzle",
							Anonymous: false,
						},
					},
				},
				// Don't recurse any fields
				recurseField: map[string]bool{"J": true},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": struct{ I int }{I: 255},
				},
			},
			expectedMangledFieldNames: []string{"J", "Q"},
			expectedUnmangledNameValues: []nameVal{{
				name: "J",
				val:  struct{ I int }{I: 255},
			}, {
				name: "I",
				val:  34,
			}},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_recurse_noflatten",
			inStruct: struct {
				J struct{ L int }
				I int
			}{
				J: struct{ L int }{
					L: 128,
				},
				I: 42,
			},
			unmangleVal: struct {
				J struct{ L int }
				Q int
			}{
				J: struct{ L int }{L: 235},
				Q: 88,
			},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      reflect.TypeOf(struct{ L int }{}),
							Tag:       "bizzlebazzle",
							Anonymous: false,
						},
					},
					"L": {
						{
							Name:      "L",
							PkgPath:   "",
							Type:      intType,
							Tag:       "'ellothere",
							Anonymous: false,
						},
					},
				},
				recurseField: map[string]bool{
					"J": true,
				},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": struct{ L int }{
						L: 255,
					},
					"L": 3128,
				},
			},
			expectedMangledFieldNames: []string{
				"J",
				"Q",
			},
			expectedUnmangledNameValues: []nameVal{
				{
					name: "J",
					val: struct{ L int }{
						L: 255,
					},
				},
				{
					name: "I",
					val:  34,
				},
			},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_slice_recurse_noflatten",
			inStruct: struct {
				J []struct{ L int }
				I int
			}{
				J: []struct{ L int }{
					{L: 128},
				},
				I: 42,
			},
			unmangleVal: struct {
				J []struct{ L int }
				Q int
			}{
				J: []struct{ L int }{{L: 235}},
				Q: 88,
			},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      reflect.SliceOf(reflect.TypeOf(struct{ L int }{})),
							Tag:       "bizzlebazzle",
							Anonymous: false,
						},
					},
					"L": {
						{
							Name:      "L",
							PkgPath:   "",
							Type:      intType,
							Tag:       "'ellothere",
							Anonymous: false,
						},
					},
				},
				recurseField: map[string]bool{
					"J": true,
				},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": []struct{ L int }{
						{L: 255},
					},
					"L": 3128,
				},
			},
			expectedMangledFieldNames: []string{
				"J",
				"Q",
			},
			expectedUnmangledNameValues: []nameVal{
				{
					name: "J",
					val: []struct{ L int }{
						{L: 255},
					},
				},
				{
					name: "I",
					val:  34,
				},
			},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_array_recurse_noflatten",
			inStruct: struct {
				J [1]struct{ L int }
				I int
			}{
				J: [1]struct{ L int }{
					{L: 128},
				},
				I: 42,
			},
			unmangleVal: struct {
				J [1]struct{ L int }
				Q int
			}{
				J: [1]struct{ L int }{{L: 235}},
				Q: 88,
			},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      reflect.ArrayOf(1, reflect.TypeOf(struct{ L int }{})),
							Tag:       "bizzlebazzle",
							Anonymous: false,
						},
					},
					"L": {
						{
							Name:      "L",
							PkgPath:   "",
							Type:      intType,
							Tag:       "'ellothere",
							Anonymous: false,
						},
					},
				},
				recurseField: map[string]bool{
					"J": true,
				},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": [1]struct{ L int }{
						{L: 255},
					},
					"L": 3128,
				},
			},
			expectedMangledFieldNames: []string{
				"J",
				"Q",
			},
			expectedUnmangledNameValues: []nameVal{
				{
					name: "J",
					val: [1]struct{ L int }{
						{L: 255},
					},
				},
				{
					name: "I",
					val:  34,
				},
			},
			expectedErr: nil,
		}, {
			name: "one_string_one_ptrd_struct_recurse_noflatten",
			inStruct: struct {
				J *struct{ L *string }
				I *string
			}{
				J: &struct{ L *string }{
					L: strPtr("fizzlebat"),
				},
				I: strPtr("foobar"),
			},
			unmangleVal: struct {
				J *struct{ L *string }
				Q *string
			}{
				J: &struct{ L *string }{L: strPtr("bzzzt")},
				Q: strPtr("que?"),
			},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      reflect.PtrTo(strType),
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      reflect.PtrTo(reflect.TypeOf(struct{ L *string }{})),
							Tag:       "bizzlebazzle",
							Anonymous: false,
						},
					},
					"L": {
						{
							Name:      "L",
							PkgPath:   "",
							Type:      reflect.PtrTo(strType),
							Tag:       "'ellothere",
							Anonymous: false,
						},
					},
				},
				recurseField: map[string]bool{
					"J": true,
				},
				origFieldVals: map[string]interface{}{
					"I": strPtr("zilch"),
					"J": &struct{ L *string }{
						L: strPtr("fizzlebizzle"),
					},
					// Note that this value won't make it
					// to the final output due to the
					// overwritten value above.
					"L": strPtr("zop"),
				},
			},
			expectedMangledFieldNames: []string{
				"J",
				"Q",
			},
			expectedUnmangledNameValues: []nameVal{
				{
					name: "J",
					val: &struct{ L *string }{
						L: strPtr("fizzlebizzle"),
					},
				},
				{
					name: "I",
					val:  strPtr("zilch"),
				},
			},
			expectedErr: nil,
		}, {
			name: "one_int_one_struct_recurse_noflatten_unmangle_error",
			inStruct: struct {
				J struct{ L int }
				I int
			}{
				J: struct{ L int }{
					L: 128,
				},
				I: 42,
			},
			unmangleVal: struct {
				J struct{ L int }
				Q int
			}{
				J: struct{ L int }{L: 235},
				Q: 88,
			},
			fm: fakeMangler{
				fieldNameMods: map[string][]reflect.StructField{
					"I": {
						{
							Name:      "Q",
							PkgPath:   "",
							Type:      intType,
							Tag:       "bitterbattle",
							Anonymous: false,
						},
					},
					"J": {
						{
							Name:      "J",
							PkgPath:   "",
							Type:      reflect.TypeOf(struct{ L int }{}),
							Tag:       "bizzlebazzle",
							Anonymous: false,
						},
					},
					"L": {
						{
							Name:      "L",
							PkgPath:   "",
							Type:      intType,
							Tag:       "'ellothere",
							Anonymous: false,
						},
					},
				},
				recurseField: map[string]bool{
					"J": true,
				},
				origFieldVals: map[string]interface{}{
					"I": 34,
					"J": struct{ L int }{
						L: 255,
					},
					"L": 3128,
				},
				unmangleErrs: map[string]error{"L": errors.New("fiddlebaz")},
			},
			expectedMangledFieldNames: []string{
				"J",
				"Q",
			},
			expectedUnmangledNameValues: []nameVal{
				{
					name: "J",
					val: struct{ L int }{
						L: 255,
					},
				},
				{
					name: "I",
					val:  34,
				},
			},
			// TODO(dfinkel): make this error message a bit less redundant
			expectedErr: strPtr("failed to unmangle field 0 (\"J\") with mangler 0 (type *transform.fakeMangler): failed to recursively inverse transform field J: failed to unmangle field 0 (\"L\") with mangler 0 (type *transform.fakeMangler): unmangle from mangler 0 (type *transform.fakeMangler) failed: fiddlebaz"),
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			inType := reflect.TypeOf(tbl.inStruct)
			transformer := NewTransformer(inType, &itbl.fm)

			mangled, mangleErr := transformer.TranslateType()
			if len(tbl.fm.mangleErrs) != 0 && tbl.expectedErr != nil {
				require.Nil(t, mangled)
				assert.EqualError(t, mangleErr, *tbl.expectedErr)

				// verify that the `Mangle` method works as intended as well.
				mangledVal, mangleValErr := transformer.Translate()
				assert.EqualError(t, mangleValErr, "failed to convert type: "+*tbl.expectedErr)
				assert.False(t, mangledVal.IsValid())
				return
			}
			require.NoError(t, mangleErr)
			require.NotNil(t, mangled)

			require.Equal(t, mangled.NumField(), len(tbl.expectedMangledFieldNames))

			for i := 0; i < mangled.NumField(); i++ {
				assert.EqualValuesf(t, tbl.expectedMangledFieldNames[i], mangled.Field(i).Name,
					"index %d", i)
			}

			mangledVal, mangleValErr := transformer.Translate()
			require.NoError(t, mangleValErr)
			assert.Equal(t, mangledVal.Type(), mangled)
			assert.True(t, mangledVal.IsValid())

			unmangleVal := reflect.ValueOf(tbl.unmangleVal)

			unmangled, unmangleErr := transformer.ReverseTranslate(unmangleVal)
			if len(tbl.fm.unmangleErrs) != 0 && tbl.expectedErr != nil {
				require.False(t, unmangled.IsValid())
				assert.EqualError(t, unmangleErr, *tbl.expectedErr)
				return
			}
			require.NoError(t, unmangleErr)
			require.True(t, unmangled.IsValid())

			require.Equal(t, unmangled.NumField()-tbl.unexported, len(tbl.expectedUnmangledNameValues))

			// Note that this loop only works properly if any
			// unexported fields are at the end of the struct.
			for i, expected := range tbl.expectedUnmangledNameValues {
				assert.EqualValuesf(t, expected.name, unmangled.Type().Field(i).Name, "index %d", i)
				assert.EqualValuesf(t, expected.val, unmangled.Field(i).Interface(), "field %q index %d",
					expected.name, i)
			}
		})
	}
}
