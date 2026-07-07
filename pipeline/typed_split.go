package pipeline

// TypedSplitter emits same-typed items from a typed split callback.
type TypedSplitter[T any] interface {
	// Emit appends item to the split output.
	Emit(item Item[T])

	// EmitValue appends value using key for item metadata.
	EmitValue(key string, value T)
}

type typedSplitter[T any] struct{ items []Item[T] }

func (s *typedSplitter[T]) Emit(item Item[T]) { s.items = append(s.items, item) }
func (s *typedSplitter[T]) EmitValue(key string, value T) {
	s.Emit(Item[T]{Key: key, Name: key, Path: key, Value: value})
}
