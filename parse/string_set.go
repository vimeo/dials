package parse

import (
	"fmt"
)

func ParseStringSet(s string) (map[string]struct{}, error) {
	ss := map[string]struct{}{}

	splitErr := splitStringsSlice(s, func(val string) error {
		if _, present := ss[val]; present {
			return fmt.Errorf("%q already present in set", val)
		}
		ss[val] = struct{}{}
		return nil
	})
	if splitErr != nil {
		return nil, splitErr
	}
	return ss, nil
}
