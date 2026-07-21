package decodo

// Ptr returns a pointer to the given value.
// Useful for setting optional fields in parameter structs.
func Ptr[T any](v T) *T {
	return &v
}
