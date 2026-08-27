package u

// Skip ignore the error.
func Skip(_ error) {}

// SkipV ignore the error and return the value.
func SkipV[T any](val T, _ error) T {
	return val
}

// SkipE ignore the error and return the error.
func SkipE[T any](_ T, e error) error {
	return e
}

// SkipD can be used to skip the error in a defer clause.
func SkipD(f func() error) {
	Skip(f())
}

// SkipD1 can be used to skip the error in a defer clause when the function takes a single argument.
func SkipD1[T any](f func(T) error, arg T) {
	Skip(f(arg))
}
