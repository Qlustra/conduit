package formats

import (
	"encoding/json"

	"github.com/qlustra/conduit/layout"
)

type JSONCodec[T any] struct{}

func (c JSONCodec[T]) Marshal(v T) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}

func (c JSONCodec[T]) Unmarshal(data []byte) (T, error) {
	var value T
	err := json.Unmarshal(data, &value)
	return value, err
}

type JSONFile[T any] struct {
	layout.Format[T, JSONCodec[T]]
}
