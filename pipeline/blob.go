package pipeline

// Blob is an opaque byte item with pipeline metadata.
type Blob struct {
	// Key is the logical identity used by duplicate handling and results.
	Key string
	// Name is the display or basename-style name for the blob.
	Name string
	// Path is the relative path used by directory sinks when preserving structure.
	Path string
	// Data is the in-memory byte payload. When empty, file-backed items may read
	// from File instead.
	Data []byte
}
