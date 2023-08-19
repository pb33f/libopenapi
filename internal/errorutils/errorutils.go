package errorutils

type MultiError struct {
	errs []error
}

func (e *MultiError) Error() string {
	var b []byte
	for i, err := range e.errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (e *MultiError) Unwrap() []error {
	return e.errs
}

func Join(errs ...error) error {
	var result MultiError

	size := 0
	for _, err := range errs {
		if err != nil {
			size++
		}
	}
	if size == 0 {
		return nil
	}

	result.errs = make([]error, 0, size)
	for _, err := range errs {
		if err != nil {
			result.errs = append(result.errs, err)
		}
	}
	return &result
}

// Unwrap recursively unwraps errors and flattens them into a slice of errors.
func Unwrap(err error) []error {
	if err == nil {
		return nil
	}
	var result []error
	if unwrap, ok := err.(interface{ Unwrap() []error }); ok {
		// ignore wrapping error - no hierarchy
		for _, e := range unwrap.Unwrap() {
			result = append(result, Unwrap(e)...)
		}
		return result
	} else if unwrap, ok := err.(interface{ Unwrap() error }); ok {
		// add parent error to result
		result = append(result, err)
		result = append(result, Unwrap(unwrap.Unwrap())...)
		return result
	}
	// no unwrapping needed, as it's not wrapped
	result = append(result, err)
	return result
}
