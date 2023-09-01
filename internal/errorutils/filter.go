package errorutils

func Filtered(err error, filters ...func(error) (keep bool)) error {
	if err == nil {
		return nil
	}
	errs := ShallowUnwrap(err)
	filtered := Filter(errs, or(filters...))
	if len(filtered) == 0 {
		return nil
	}
	return Join(filtered...)
}

func Filter(errs []error, filter func(error) (keep bool)) []error {
	var result []error
	var keep bool
	for _, err := range errs {
		keep = filter(err)
		if keep {
			result = append(result, err)
		}
	}
	return result
}

func or(filters ...func(error) (keep bool)) func(error) (keep bool) {
	return func(err error) bool {
		var keep bool
		for _, filter := range filters {
			keep = filter(err)
			if keep {
				return true
			}
		}
		// all false -> false
		return false
	}
}
