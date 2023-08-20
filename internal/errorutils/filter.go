package errorutils

func Filtered(err error, filters ...func(error) (keep bool)) error {
	if err == nil {
		return nil
	}
	errs := Unwrap(err)
	filtered := Filter(errs, AndFilter(filters...))
	if len(filtered) == 0 {
		return nil
	}
	return Join(filtered...)
}

func Filter(errs []error, filter func(error) (keep bool)) []error {
	var result []error
	for _, err := range errs {
		if filter(err) {
			result = append(result, err)
		}
	}
	return result
}

func AndFilter(filters ...func(error) (keep bool)) func(error) (keep bool) {
	return func(err error) bool {
		for _, filter := range filters {
			if !filter(err) {
				return false
			}
		}
		return true
	}
}
