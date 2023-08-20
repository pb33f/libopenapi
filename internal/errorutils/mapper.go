package errorutils

func Mapped(err error, mapper ...func(src error) (dst error, keep bool)) error {
	if err == nil {
		return nil
	}
	errs := Unwrap(err)
	mapped := Map(errs, AndMapper(mapper...))
	if len(mapped) == 0 {
		return nil
	}
	return Join(mapped...)
}

func Map(errs []error, mapper func(src error) (dst error, keep bool)) []error {
	var result []error
	for _, err := range errs {
		dst, keep := mapper(err)
		if keep {
			result = append(result, dst)
		}
	}
	return result
}

func AndMapper(mappers ...func(error) (error, bool)) func(error) (error, bool) {
	return func(srcErr error) (error, bool) {
		var (
			dstErr = srcErr
			keep   bool
		)
		for _, mapper := range mappers {

			dstErr, keep = mapper(dstErr)
			if !keep {
				return nil, false
			}
		}
		// final result to keep
		return dstErr, true
	}
}
