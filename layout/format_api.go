package layout

// FormatAPI is the complete exported API of Format for content type T, independent of its codec.
type FormatAPI[T any] interface {
	FileAPI
	Clear()
	DecodeValue(data []byte) (T, error)
	Delete(ctx Context) error
	Discover() (DiskState, error)
	DiskState() DiskState
	Encode() ([]byte, error)
	EncodeValue(value T) ([]byte, error)
	Get() (T, bool)
	HasBeenLoaded() bool
	HasContent() bool
	HasKnownDiskState() bool
	IsDirty() bool
	Load() (bool, error)
	LoadOrInit(defaultValue T) error
	MemoryState() MemoryState
	MustGet() T
	PrepareWrite() (PreparedWrite, error)
	PreserveState() func()
	Read() (T, error)
	ReadIfExists() (T, bool, error)
	Save(ctx Context) error
	Scan() (DiskState, error)
	Set(value T)
	SetDefault(value T) bool
	SnapshotState() (func(), error)
	Sync(ctx Context) (ResultCode, error)
	Unload()
	WasObservedOnDisk() bool
	Write(value T, ctx Context) error
}
