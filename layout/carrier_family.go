package layout

import (
	"reflect"

	"github.com/qlustra/sail/subject"
)

// ContentCarrier identifies built-in stateful content carriers and their
// embeddings. ContentFamily supplies the complete native API of each carrier.
type ContentCarrier interface {
	Pather
	ContentState
	contentContract() reflect.Type
}

// ContentFamily supplies the carrier's complete API while preserving its typed
// content or render context. Its zero value is ready for declaration checks.
type ContentFamily struct{}

func (ContentFamily) ContractOf(carrier ContentCarrier) reflect.Type {
	return carrier.contentContract()
}

// CollectionCarrier identifies built-in member sources and their embeddings.
// CollectionFamily supplies the complete API with the original member type.
type CollectionCarrier interface {
	Pather
	subject.Enterable
	ChildKind() string
	collectionContract() reflect.Type
}

// CollectionFamily supplies the carrier's complete API while preserving its
// member type. Its zero value is ready for declaration checks.
type CollectionFamily struct{}

func (CollectionFamily) ContractOf(carrier CollectionCarrier) reflect.Type {
	return carrier.collectionContract()
}

// CarrierContract returns the complete native API supplied by a built-in
// carrier or its embedding. Metadata-only checks preserve the contract even
// when an outer wrapper shadows an operation with an incompatible signature.
// Content and collection contracts take precedence over filesystem handles.
func CarrierContract(carrier any) (reflect.Type, bool) {
	switch carrier := carrier.(type) {
	case interface{ contentContract() reflect.Type }:
		return carrier.contentContract(), true
	case interface{ collectionContract() reflect.Type }:
		return carrier.collectionContract(), true
	case interface{ filesystemContract() reflect.Type }:
		return carrier.filesystemContract(), true
	default:
		return nil, false
	}
}

func (*File) filesystemContract() reflect.Type {
	return reflect.TypeFor[FileAPI]()
}

func (*Dir) filesystemContract() reflect.Type {
	return reflect.TypeFor[DirAPI]()
}

func (*Exec) filesystemContract() reflect.Type {
	return reflect.TypeFor[ExecAPI]()
}

func (*Format[T, C]) contentContract() reflect.Type {
	var _ FormatAPI[T] = (*Format[T, C])(nil)
	return reflect.TypeFor[FormatAPI[T]]()
}

func (*TextTemplate[C]) contentContract() reflect.Type {
	var _ TextTemplateAPI[C] = (*TextTemplate[C])(nil)
	return reflect.TypeFor[TextTemplateAPI[C]]()
}

func (*Link) contentContract() reflect.Type {
	return reflect.TypeFor[LinkAPI]()
}

func (*FileLink) contentContract() reflect.Type {
	return reflect.TypeFor[FileLinkAPI]()
}

func (*DirLink) contentContract() reflect.Type {
	return reflect.TypeFor[DirLinkAPI]()
}

func (*Slot[T]) collectionContract() reflect.Type {
	var _ SlotAPI[T] = (*Slot[T])(nil)
	return reflect.TypeFor[SlotAPI[T]]()
}

func (*FileSlot[T]) collectionContract() reflect.Type {
	var _ FileSlotAPI[T] = (*FileSlot[T])(nil)
	return reflect.TypeFor[FileSlotAPI[T]]()
}

func (*LinkSlot[T]) collectionContract() reflect.Type {
	var _ LinkSlotAPI[T] = (*LinkSlot[T])(nil)
	return reflect.TypeFor[LinkSlotAPI[T]]()
}

var (
	_ FileAPI        = (*File)(nil)
	_ DirAPI         = (*Dir)(nil)
	_ ExecAPI        = (*Exec)(nil)
	_ LinkAPI        = (*Link)(nil)
	_ FileLinkAPI    = (*FileLink)(nil)
	_ DirLinkAPI     = (*DirLink)(nil)
	_ ContentCarrier = (*Link)(nil)
)
