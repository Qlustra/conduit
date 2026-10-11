package conduit_test

import (
	"github.com/qlustra/conduit"
	"testing"
)

func composeTree[T any](t *testing.T, root string) (*conduit.Layout[T], *T) {
	t.Helper()
	structure, err := conduit.Compile[T]()
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	manager, err := structure.Compose(root)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	return manager, manager.Tree()
}

func composeError[T any](root string) error {
	structure, err := conduit.Compile[T]()
	if err != nil {
		return err
	}
	_, err = structure.Compose(root)
	return err
}

type textCodec struct{}

func (textCodec) Marshal(value string) ([]byte, error)  { return []byte(value), nil }
func (textCodec) Unmarshal(data []byte) (string, error) { return string(data), nil }
