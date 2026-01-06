package transform

import (
	"reflect"
	"testing"
	"time"
)

type checkFieldTypeAndSetValsFieldDesc struct {
	fieldName string
	expType   reflect.Type
	setVal    any
}

func checkFieldTypeAndSetVals(t testing.TB, val reflect.Value, fDescs []checkFieldTypeAndSetValsFieldDesc) {
	t.Helper()

	for _, fDesc := range fDescs {
		f := val.FieldByName(fDesc.fieldName)
		if f.Type() != fDesc.expType {
			t.Errorf("unexpected type for field %q: %s; expected %s", fDesc.fieldName, f.Type(), fDesc.expType)
			continue
		}
		setV := reflect.ValueOf(fDesc.setVal)
		f.Set(setV)
	}
}

func TestSingleTypeSubstitutionMangler_Int64_Int8(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Foo          int64
		Bar          int64
		Boop         [3]int64
		Baz          *int64
		BarChan      <-chan int64
		BarChanUnbuf <-chan int64
		BazChan      <-chan *int64
		BazChanUnbuf <-chan *int64
		Fizzle       string
		Fooble       *string
	}

	m, constrErr := NewSingleTypeSubstitutionMangler[int64, int8]()
	if constrErr != nil {
		t.Fatalf("failed to construct substitution mangler: %s", constrErr)
	}

	itype := reflect.TypeFor[testStruct]()

	tfmr := NewTransformer(itype, m)
	val, trErr := tfmr.Translate()
	if trErr != nil {
		t.Fatalf("failed to translate type: %s", trErr)
	}
	int8T := reflect.TypeFor[int8]()
	strT := reflect.TypeFor[string]()

	bazChanVal := int8(31)
	checkFieldTypeAndSetVals(t, val, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "Foo", expType: int8T, setVal: int8(8)},
		{fieldName: "Bar", expType: int8T, setVal: int8(33)},
		{fieldName: "Boop", expType: reflect.ArrayOf(3, int8T), setVal: [3]int8{33, 123, 9}},
		{fieldName: "Baz", expType: reflect.PointerTo(int8T), setVal: func() *int8 { v := int8(38); return &v }()},
		{fieldName: "BarChan", expType: reflect.ChanOf(reflect.RecvDir, int8T), setVal: func() chan int8 { v := make(chan int8, 3); v <- 22; return v }()},
		{fieldName: "BarChanUnbuf", expType: reflect.ChanOf(reflect.RecvDir, int8T), setVal: make(chan int8)},
		{fieldName: "BazChan", expType: reflect.ChanOf(reflect.RecvDir, reflect.PointerTo(int8T)), setVal: func() chan *int8 { v := make(chan *int8, 3); v <- &bazChanVal; return v }()},
		{fieldName: "BazChanUnbuf", expType: reflect.ChanOf(reflect.RecvDir, reflect.PointerTo(int8T)), setVal: make(chan *int8)},
		{fieldName: "Fizzle", expType: strT, setVal: "foobar"},
		{fieldName: "Fooble", expType: reflect.PointerTo(strT), setVal: func() *string { v := "feebleboop"; return &v }()},
	})

	revVal, revTrErr := tfmr.ReverseTranslate(val)
	if revTrErr != nil {
		t.Fatalf("failed to reverse translate type: %s", revTrErr)
	}

	expBazVal := int64(38)
	expFooble := "feebleboop"

	rv := revVal.Interface().(testStruct)
	if expOut := (testStruct{
		Foo:          8,
		Bar:          33,
		Baz:          &expBazVal,
		Boop:         [3]int64{33, 123, 9},
		BarChan:      rv.BarChan,
		BarChanUnbuf: rv.BarChanUnbuf,
		BazChan:      rv.BazChan,
		BazChanUnbuf: rv.BazChanUnbuf,
		Fizzle:       "foobar",
		Fooble:       &expFooble,
	}); !reflect.DeepEqual(revVal.Interface(), expOut) {
		t.Errorf("unexpected output:\n got %+v\nwant %+v", revVal.Interface(), expOut)
	}
	if len(rv.BarChan) != 1 {
		t.Errorf("BarChan has unexpected number of values: %d; expected 1", len(rv.BarChan))
	}
	if cap(rv.BarChan) != 3 {
		t.Errorf("BarChan has unexpected capacity: %d; expected 3", cap(rv.BarChan))
	}
	select {
	case v := <-rv.BarChan:
		if v != 22 {
			t.Errorf("unexpected value for value in BarChan: %d; expected 22", v)
		}
	default:
		t.Errorf("BarChan empty")
	}
	if len(rv.BazChan) != 1 {
		t.Errorf("BazChan has unexpected number of values: %d; expected 1", len(rv.BazChan))
	}
	if cap(rv.BazChan) != 3 {
		t.Errorf("BazChan has unexpected capacity: %d; expected 3", cap(rv.BazChan))
	}
	select {
	case v := <-rv.BazChan:
		if v == nil {
			t.Errorf("unexpected nil value for value in BazChan; expected pointer to 31")
		} else if *v != 31 {
			t.Errorf("unexpected value for value in BazChan: %d; expected pointer to 31", *v)
		}
	default:
		t.Errorf("BazChan empty")
	}
}

func TestSingleTypeSubstitutionMangler_timeDuration_int64(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Foo         time.Duration
		Bar         time.Duration
		Baz         *time.Duration
		BarChanNone chan time.Duration
		BazChanNone chan *time.Duration
		Fizzle      string
		Fooble      *string
	}

	m, constrErr := NewSingleTypeSubstitutionMangler[time.Duration, int64]()
	if constrErr != nil {
		t.Fatalf("failed to construct substitution mangler: %s", constrErr)
	}

	itype := reflect.TypeFor[testStruct]()

	tfmr := NewTransformer(itype, m)
	val, trErr := tfmr.Translate()
	if trErr != nil {
		t.Fatalf("failed to translate type: %s", trErr)
	}
	int64T := reflect.TypeFor[int64]()
	strT := reflect.TypeFor[string]()

	checkFieldTypeAndSetVals(t, val, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "Foo", expType: int64T, setVal: int64(8_000_023)},
		{fieldName: "Bar", expType: int64T, setVal: int64(33_000_081)},
		{fieldName: "Baz", expType: reflect.PointerTo(int64T), setVal: func() *int64 { v := int64(393455); return &v }()},
		{fieldName: "BarChanNone", expType: reflect.ChanOf(reflect.BothDir, int64T), setVal: (chan int64)(nil)},
		{fieldName: "BazChanNone", expType: reflect.ChanOf(reflect.BothDir, reflect.PointerTo(int64T)), setVal: (chan *int64)(nil)},
		{fieldName: "Fizzle", expType: strT, setVal: "foobar"},
		{fieldName: "Fooble", expType: reflect.PointerTo(strT), setVal: func() *string { v := "feebleboop"; return &v }()},
	})

	revVal, revTrErr := tfmr.ReverseTranslate(val)
	if revTrErr != nil {
		t.Fatalf("failed to reverse translate type: %s", revTrErr)
	}

	expBazVal := 393455 * time.Nanosecond
	expFooble := "feebleboop"

	if expOut := (testStruct{
		Foo:         8_000_023 * time.Nanosecond,
		Bar:         33_000_081 * time.Nanosecond,
		Baz:         &expBazVal,
		BarChanNone: nil,
		BazChanNone: nil,
		Fizzle:      "foobar",
		Fooble:      &expFooble,
	}); !reflect.DeepEqual(revVal.Interface(), expOut) {
		t.Errorf("unexpected output: got %+v; want %+v", revVal.Interface(), expOut)
	}
}

func TestSingleTypeSubstitutionMangler_Int64_Int16_map_slice_fields(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Foo           map[string]int64
		Bar           []int64
		Baz           *int64
		Bamboozle     map[int64]int64
		BamboozleNone map[int64]int64
		BarNone       []int64
		BazNone       *int64
		Fizzle        string
		Fooble        *string
		Fromble       *string
	}

	m, constrErr := NewSingleTypeSubstitutionMangler[int64, int16]()
	if constrErr != nil {
		t.Fatalf("failed to construct substitution mangler: %s", constrErr)
	}

	itype := reflect.TypeFor[testStruct]()

	tfmr := NewTransformer(itype, m)
	val, trErr := tfmr.Translate()
	if trErr != nil {
		t.Fatalf("failed to translate type: %s", trErr)
	}
	int16T := reflect.TypeFor[int16]()
	strT := reflect.TypeFor[string]()

	checkFieldTypeAndSetVals(t, val, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "Foo", expType: reflect.MapOf(strT, int16T), setVal: map[string]int16{"abc": 8}},
		{fieldName: "Bar", expType: reflect.SliceOf(int16T), setVal: []int16{33, 1337}},
		{fieldName: "BarNone", expType: reflect.SliceOf(int16T), setVal: []int16(nil)},
		{fieldName: "Bamboozle", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16{1997: 9}},
		{fieldName: "BamboozleNone", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16(nil)},
		{fieldName: "Baz", expType: reflect.PointerTo(int16T), setVal: func() *int16 { v := int16(38); return &v }()},
		{fieldName: "BazNone", expType: reflect.PointerTo(int16T), setVal: (*int16)(nil)},
		{fieldName: "Fizzle", expType: strT, setVal: "foobar"},
		{fieldName: "Fooble", expType: reflect.PointerTo(strT), setVal: func() *string { v := "feebleboop"; return &v }()},
		{fieldName: "Fromble", expType: reflect.PointerTo(strT), setVal: (*string)(nil)},
	})

	revVal, revTrErr := tfmr.ReverseTranslate(val)
	if revTrErr != nil {
		t.Fatalf("failed to reverse translate type: %s", revTrErr)
	}

	expBazVal := int64(38)
	expFooble := "feebleboop"

	if expOut := (testStruct{
		Foo:           map[string]int64{"abc": 8},
		Bar:           []int64{33, 1337},
		Baz:           &expBazVal,
		Bamboozle:     map[int64]int64{1997: 9},
		BarNone:       nil,
		BazNone:       nil,
		BamboozleNone: nil,
		Fizzle:        "foobar",
		Fooble:        &expFooble,
		Fromble:       nil,
	}); !reflect.DeepEqual(revVal.Interface(), expOut) {
		t.Errorf("unexpected output: got %+v; want %+v", revVal.Interface(), expOut)
	}
}

func TestSingleTypeSubstitutionMangler_Int64_Int16_nested_fields(t *testing.T) {
	t.Parallel()
	type innerStruct struct {
		Foo           map[string]int64
		Bar           []int64
		Baz           *int64
		Bamboozle     map[int64]int64
		BamboozleNone map[int64]int64
	}
	type testStruct struct {
		P       *innerStruct
		I       innerStruct
		PNone   *innerStruct
		BarNone []int64
		BazNone *int64
		Fizzle  string
		Fooble  *string
		Fromble *string
	}

	m, constrErr := NewSingleTypeSubstitutionMangler[int64, int16]()
	if constrErr != nil {
		t.Fatalf("failed to construct substitution mangler: %s", constrErr)
	}

	itype := reflect.TypeFor[testStruct]()

	tfmr := NewTransformer(itype, m)
	val, trErr := tfmr.Translate()
	if trErr != nil {
		t.Fatalf("failed to translate type: %s", trErr)
	}
	int16T := reflect.TypeFor[int16]()
	strT := reflect.TypeFor[string]()

	iVal := val.FieldByName("I")
	checkFieldTypeAndSetVals(t, iVal, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "Foo", expType: reflect.MapOf(strT, int16T), setVal: map[string]int16{"abc": 8}},
		{fieldName: "Bar", expType: reflect.SliceOf(int16T), setVal: []int16{33, 1337}},
		{fieldName: "Baz", expType: reflect.PointerTo(int16T), setVal: func() *int16 { v := int16(38); return &v }()},
		{fieldName: "Bamboozle", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16{1997: 9}},
		{fieldName: "BamboozleNone", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16(nil)},
	})
	pField := val.FieldByName("P")
	pPtr := reflect.New(pField.Type().Elem())
	pVal := pPtr.Elem()
	pField.Set(pPtr)
	checkFieldTypeAndSetVals(t, pVal, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "Foo", expType: reflect.MapOf(strT, int16T), setVal: map[string]int16{"abc": 31_123}},
		{fieldName: "Bar", expType: reflect.SliceOf(int16T), setVal: []int16{37, 1339}},
		{fieldName: "Baz", expType: reflect.PointerTo(int16T), setVal: func() *int16 { v := int16(36); return &v }()},
		{fieldName: "Bamboozle", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16{1987: 10}},
		{fieldName: "BamboozleNone", expType: reflect.MapOf(int16T, int16T), setVal: map[int16]int16(nil)},
	})

	checkFieldTypeAndSetVals(t, val, []checkFieldTypeAndSetValsFieldDesc{
		{fieldName: "BarNone", expType: reflect.SliceOf(int16T), setVal: []int16(nil)},
		{fieldName: "BazNone", expType: reflect.PointerTo(int16T), setVal: (*int16)(nil)},
		{fieldName: "Fizzle", expType: strT, setVal: "foobar"},
		{fieldName: "Fooble", expType: reflect.PointerTo(strT), setVal: func() *string { v := "feebleboop"; return &v }()},
		{fieldName: "Fromble", expType: reflect.PointerTo(strT), setVal: (*string)(nil)},
		{fieldName: "PNone", expType: reflect.PointerTo(iVal.Type()), setVal: reflect.Zero(reflect.PointerTo(iVal.Type())).Interface()},
	})

	revVal, revTrErr := tfmr.ReverseTranslate(val)
	if revTrErr != nil {
		t.Fatalf("failed to reverse translate type: %s", revTrErr)
	}

	expBazVal := int64(38)
	expPtrBazVal := int64(36)
	expFooble := "feebleboop"

	rv := revVal.Interface().(testStruct)

	if expOut := (testStruct{
		I: innerStruct{
			Foo:           map[string]int64{"abc": 8},
			Bar:           []int64{33, 1337},
			Baz:           &expBazVal,
			Bamboozle:     map[int64]int64{1997: 9},
			BamboozleNone: nil,
		},
		P: &innerStruct{
			Foo:           map[string]int64{"abc": 31_123},
			Bar:           []int64{37, 1339},
			Baz:           &expPtrBazVal,
			Bamboozle:     map[int64]int64{1987: 10},
			BamboozleNone: nil,
		},
		PNone:   nil,
		BarNone: nil,
		BazNone: nil,
		Fizzle:  "foobar",
		Fooble:  &expFooble,
		Fromble: nil,
	}); !reflect.DeepEqual(rv, expOut) {
		t.Errorf("unexpected output:\n got %+v\nwant %+v", revVal.Interface(), expOut)
		t.Logf("I:\n got %+v\nwant %+v", rv.I, expOut.I)
		t.Logf("P:\n got %+v\nwant %+v", *rv.P, *expOut.P)
		t.Logf("P.Baz:\n got %v\nwant %v", *rv.P.Baz, *expOut.P.Baz)
	}
}
