package pipeline

type inputSource[I any] interface {
	snapshotItems() ([]Item[I], error)
}

type singleSource[I any] interface {
}

type mapSource[I any] interface {
	at(key string) (Item[I], error)
	put(key string, value I)
}
