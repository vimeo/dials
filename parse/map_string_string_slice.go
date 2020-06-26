package parse

// StringStringSliceMap takes in a string representation of a map with
// strings as keys and string slices as values, and converts it to a
// map[string][]string.
func StringStringSliceMap(s string) (map[string][]string, error) {
	ss := map[string][]string{}
	splitErr := splitMap(s,
		func(k, v string) error {
			ss[k] = append(ss[k], v)
			return nil
		})
	if splitErr != nil {
		return nil, splitErr
	}
	return ss, nil
}
