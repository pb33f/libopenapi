package errorutils

// Filter returns a filtered multi error.
// filter functions must return false for their specific errors
// but true for every other unknown error in order to keep them untouched and potentially
// removed by another filter.
func Filtered(err error, filters ...func(error) (keep bool)) error {
	if err == nil {
		return nil
	}
	errs := ShallowUnwrap(err)
	filtered := Filter(errs, and(filters...))
	if len(filtered) == 0 {
		return nil
	}
	return Join(filtered...)
}

func Filter(errs []error, filter func(error) (keep bool)) []error {
	var result []error
	var keep bool
	for _, err := range errs {
		if err == nil {
			continue
		}
		keep = filter(err)
		if keep {
			result = append(result, err)
		}
	}
	return result
}

func and(filters ...func(error) (keep bool)) func(error) (keep bool) {
	return func(err error) bool {
		var keep bool
		for _, filter := range filters {
			keep = filter(err)
			if !keep {
				return false
			}
		}
		// all true -> true
		return true
	}
}
