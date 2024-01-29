package parse

import "testing"

func TestSignedIntegralSliceInts(t *testing.T) {
	for _, tbl := range []struct {
		name   string
		in     string
		expOut []int
		expErr bool
	}{
		{
			name:   "good_1_int",
			in:     "1234",
			expOut: []int{1234},
			expErr: false,
		}, {
			name:   "good_1_int_trailing_whitespace",
			in:     "1234    ",
			expOut: []int{1234},
			expErr: false,
		}, {
			name:   "good_1_int_leading_whitespace",
			in:     "   1234",
			expOut: []int{1234},
			expErr: false,
		}, {
			name:   "good_2_int",
			in:     "1234,3456",
			expOut: []int{1234, 3456},
			expErr: false,
		}, {
			name:   "good_2_int_interstitial_whitespace",
			in:     "1234, 3456",
			expOut: []int{1234, 3456},
			expErr: false,
		}, {
			name:   "good_3_int",
			in:     "1234,3456,789",
			expOut: []int{1234, 3456, 789},
			expErr: false,
		}, {
			name:   "good_2_int_negative",
			in:     "-1234,-3456",
			expOut: []int{-1234, -3456},
			expErr: false,
		}, {
			name:   "good_3_int_negative",
			in:     "-1234,-3456,-789",
			expOut: []int{-1234, -3456, -789},
			expErr: false,
		}, {
			name:   "good_3_int_digit_separator",
			in:     "1_234,3_456,789",
			expOut: []int{1_234, 3_456, 789},
			expErr: false,
		}, {
			name:   "good_3_ints_hex",
			in:     "0x1234e,0x3456e,0x789e",
			expOut: []int{0x1234e, 0x3456e, 0x789e},
			expErr: false,
		}, {
			name:   "good_3_ints_octal_legacy",
			in:     "0777,03456,07004",
			expOut: []int{0777, 03456, 07004},
			expErr: false,
		}, {
			name:   "good_3_ints_octal_new",
			in:     "0o777,0o3456,0o7004",
			expOut: []int{0777, 03456, 07004},
			expErr: false,
		}, {
			name:   "good_3_ints_binary",
			in:     "0b1111,0b111001,0b1000111",
			expOut: []int{0b1111, 0b111001, 0b1000111},
			expErr: false,
		}, {
			name:   "bad_2_ints_not_int",
			in:     "1234,3456,fizzlebit",
			expOut: nil,
			expErr: true,
		}, {
			name:   "not_int",
			in:     "fizzlebit!!!!",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_2_ints_trailing_garbage",
			in:     "1234,3456$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_trailing_garbage",
			in:     "3456$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_leading_garbage",
			in:     "$%^&3456",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_overflow",
			in:     "123_434_599_999_000_999_000",
			expOut: nil,
			expErr: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			out, outErr := SignedIntegralSlice[int](tbl.in)
			if outErr != nil {
				if !tbl.expErr {
					t.Errorf("unexpected error for input %q: %s", tbl.in, outErr)
				}
				t.Logf("error: %s", outErr)
				return
			}
			if len(out) != len(tbl.expOut) {
				t.Errorf("mismatched lengths: got %d; want %d", len(out), len(tbl.expOut))
			}
			for i, v := range out {
				if tbl.expOut[i] != v {
					t.Errorf("unexpected value at output index %d: got %d; want %d", i, v, tbl.expOut[i])
				}
			}
		})
	}
}
func TestSignedIntegralSliceInt8s(t *testing.T) {
	for _, tbl := range []struct {
		name   string
		in     string
		expOut []int8
		expErr bool
	}{
		{
			name:   "good_1_int",
			in:     "123",
			expOut: []int8{123},
			expErr: false,
		}, {
			name:   "good_2_int",
			in:     "123,34",
			expOut: []int8{123, 34},
			expErr: false,
		}, {
			name:   "good_2_int_negative",
			in:     "-123,-34",
			expOut: []int8{-123, -34},
			expErr: false,
		}, {
			name:   "good_3_int",
			in:     "123,34,78",
			expOut: []int8{123, 34, 78},
			expErr: false,
		}, {
			name:   "good_3_int_digit_separator",
			in:     "1_2,3_4,78",
			expOut: []int8{12, 3_4, 78},
			expErr: false,
		}, {
			name:   "good_3_ints_hex",
			in:     "0x12,0x7f,0x78",
			expOut: []int8{0x12, 0x7f, 0x78},
			expErr: false,
		}, {
			name:   "bad_2_ints_not_int",
			in:     "12,34,fizzlebit",
			expOut: nil,
			expErr: true,
		}, {
			name:   "not_int",
			in:     "fizzlebit!!!!",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_2_ints_trailing_garbage",
			in:     "123,34$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_trailing_garbage",
			in:     "45$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_leading_garbage",
			in:     "$%^&3456",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_overflow",
			in:     "123_434",
			expOut: nil,
			expErr: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			out, outErr := SignedIntegralSlice[int8](tbl.in)
			if outErr != nil {
				if !tbl.expErr {
					t.Errorf("unexpected error for input %q: %s", tbl.in, outErr)
				}
				t.Logf("error: %s", outErr)
				return
			}
			if len(out) != len(tbl.expOut) {
				t.Errorf("mismatched lengths: got %d; want %d", len(out), len(tbl.expOut))
			}
			for i, v := range out {
				if tbl.expOut[i] != v {
					t.Errorf("unexpected value at output index %d: got %d; want %d", i, v, tbl.expOut[i])
				}
			}
		})
	}
}

func TestUnsignedIntegralSlice(t *testing.T) {
	for _, tbl := range []struct {
		name   string
		in     string
		expOut []uint
		expErr bool
	}{
		{
			name:   "good_1_int",
			in:     "1234",
			expOut: []uint{1234},
			expErr: false,
		}, {
			name:   "good_1_int_leading_whitespace",
			in:     "\t1234",
			expOut: []uint{1234},
			expErr: false,
		}, {
			name:   "good_1_int_trailing_whitespace",
			in:     "1234 \t",
			expOut: []uint{1234},
			expErr: false,
		}, {
			name:   "good_2_int",
			in:     "1234,3456",
			expOut: []uint{1234, 3456},
			expErr: false,
		}, {
			name:   "good_3_int",
			in:     "1234,3456,789",
			expOut: []uint{1234, 3456, 789},
			expErr: false,
		}, {
			name:   "bad_2_int_negative",
			in:     "-1234,-3456",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_3_int_negative",
			in:     "-1234,-3456,-789",
			expOut: nil,
			expErr: true,
		}, {
			name:   "good_3_int_digit_separator",
			in:     "1_234,3_456,789",
			expOut: []uint{1_234, 3_456, 789},
			expErr: false,
		}, {
			name:   "good_3_ints_hex",
			in:     "0x1234e,0x3456e,0x789e",
			expOut: []uint{0x1234e, 0x3456e, 0x789e},
			expErr: false,
		}, {
			name:   "good_3_ints_octal_legacy",
			in:     "0777,03456,07004",
			expOut: []uint{0777, 03456, 07004},
			expErr: false,
		}, {
			name:   "good_3_ints_octal_new",
			in:     "0o777,0o3456,0o7004",
			expOut: []uint{0777, 03456, 07004},
			expErr: false,
		}, {
			name:   "good_3_ints_binary",
			in:     "0b1111,0b111001,0b1000111",
			expOut: []uint{0b1111, 0b111001, 0b1000111},
			expErr: false,
		}, {
			name:   "bad_2_ints_not_int",
			in:     "1234,3456,fizzlebit",
			expOut: nil,
			expErr: true,
		}, {
			name:   "not_int",
			in:     "fizzlebit!!!!",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_2_ints_trailing_garbage",
			in:     "1234,3456$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_trailing_garbage",
			in:     "3456$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_leading_garbage",
			in:     "$%^&3456",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_overflow",
			in:     "123_434_599_999_000_999_000",
			expOut: nil,
			expErr: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			out, outErr := UnsignedIntegralSlice[uint](tbl.in)
			if outErr != nil {
				if !tbl.expErr {
					t.Errorf("unexpected error for input %q: %s", tbl.in, outErr)
				}
				t.Logf("error: %s", outErr)
				return
			}
			if len(out) != len(tbl.expOut) {
				t.Errorf("mismatched lengths: got %d; want %d", len(out), len(tbl.expOut))
			}
			for i, v := range out {
				if tbl.expOut[i] != v {
					t.Errorf("unexpected value at output index %d: got %d; want %d", i, v, tbl.expOut[i])
				}
			}
		})
	}
}

func TestUnsignedIntegralSliceUint8(t *testing.T) {
	for _, tbl := range []struct {
		name   string
		in     string
		expOut []uint8
		expErr bool
	}{
		{
			name:   "good_1_int",
			in:     "123",
			expOut: []uint8{123},
			expErr: false,
		}, {
			name:   "good_2_int",
			in:     "123,34",
			expOut: []uint8{123, 34},
			expErr: false,
		}, {
			name:   "bad_2_int_negative",
			in:     "-123,-34",
			expOut: nil,
			expErr: true,
		}, {
			name:   "good_3_int",
			in:     "123,34,78",
			expOut: []uint8{123, 34, 78},
			expErr: false,
		}, {
			name:   "good_3_int_digit_separator",
			in:     "1_2,3_4,78",
			expOut: []uint8{12, 3_4, 78},
			expErr: false,
		}, {
			name:   "good_3_ints_hex",
			in:     "0x12,0xff,0x78",
			expOut: []uint8{0x12, 0xff, 0x78},
			expErr: false,
		}, {
			name:   "bad_2_ints_not_int",
			in:     "12,34,fizzlebit",
			expOut: nil,
			expErr: true,
		}, {
			name:   "not_int",
			in:     "fizzlebit!!!!",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_2_ints_trailing_garbage",
			in:     "123,34$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_trailing_garbage",
			in:     "45$%^&",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_leading_garbage",
			in:     "$%^&3456",
			expOut: nil,
			expErr: true,
		}, {
			name:   "bad_1_int_overflow",
			in:     "123_434",
			expOut: nil,
			expErr: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			out, outErr := UnsignedIntegralSlice[uint8](tbl.in)
			if outErr != nil {
				if !tbl.expErr {
					t.Errorf("unexpected error for input %q: %s", tbl.in, outErr)
				}
				t.Logf("error: %s", outErr)
				return
			}
			if len(out) != len(tbl.expOut) {
				t.Errorf("mismatched lengths: got %d; want %d", len(out), len(tbl.expOut))
			}
			for i, v := range out {
				if tbl.expOut[i] != v {
					t.Errorf("unexpected value at output index %d: got %d; want %d", i, v, tbl.expOut[i])
				}
			}
		})
	}
}
