package utils

func Ptr[T any](v T) *T {
	return &v
}

func EqualPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func MapPtr[T, U any](p *T, f func(T) U) *U {
	if p == nil {
		return nil
	}

	u := f(*p)

	return &u
}
