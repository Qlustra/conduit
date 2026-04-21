package formats

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/qlustra/conduit/internal/layout"
)

type TOMLCodec[T any] struct{}

func (c TOMLCodec[T]) Marshal(v T) ([]byte, error) { return toml.Marshal(v) }
func (c TOMLCodec[T]) Unmarshal(data []byte) (T, error) {
	var v T
	err := toml.Unmarshal(data, &v)
	return v, err
}

type TOMLFile[T any] struct {
	layout.Format[T, TOMLCodec[T]]
}
