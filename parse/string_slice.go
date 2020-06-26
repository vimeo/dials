package parse

// StringSlice splits a comma separated string and constructs a slice
func StringSlice(s string) ([]string, error) {
	ss := []string{}

	splitErr := splitStringsSlice(s, func(val string) error {
		ss = append(ss, val)
		return nil
	})

	if splitErr != nil {
		return nil, splitErr
	}

	return ss, nil
}
