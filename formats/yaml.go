package formats

import (
	"github.com/qlustra/conduit/layout"
	"gopkg.in/yaml.v3"
)

type YAMLCodec[T any] struct{}

func (c YAMLCodec[T]) Marshal(v T) ([]byte, error) { return yaml.Marshal(v) }
func (c YAMLCodec[T]) Unmarshal(data []byte) (T, error) {
	var value T
	err := yaml.Unmarshal(data, &value)
	return value, err
}

type YAMLFile[T any] struct {
	layout.Format[T, YAMLCodec[T]]
}
