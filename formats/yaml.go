package formats

import (
	"github.com/qlustra/conduit/layout"
	"gopkg.in/yaml.v3"
)

// YAMLCodec marshals and unmarshals typed values as YAML.
type YAMLCodec[T any] struct{}

func (c YAMLCodec[T]) Marshal(v T) ([]byte, error) { return yaml.Marshal(v) }
func (c YAMLCodec[T]) Unmarshal(data []byte) (T, error) {
	var value T
	err := yaml.Unmarshal(data, &value)
	return value, err
}

// YAMLFile is a Format that stores typed content as YAML.
type YAMLFile[T any] struct {
	layout.Format[T, YAMLCodec[T]]
}
