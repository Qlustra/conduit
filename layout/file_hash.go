package layout

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// HashReader reads src into h and returns the digest bytes.
func HashReader(src io.Reader, h hash.Hash) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("hash source must not be nil")
	}
	if h == nil {
		return nil, fmt.Errorf("hash must not be nil")
	}

	h.Reset()
	if _, err := io.Copy(h, src); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// HashBytes returns the digest bytes for data.
//
// h must not be nil.
func HashBytes(data []byte, h hash.Hash) []byte {
	h.Reset()
	_, _ = h.Write(data)
	return h.Sum(nil)
}

// HashString returns the digest bytes for data.
//
// h must not be nil.
func HashString(data string, h hash.Hash) []byte {
	return HashBytes([]byte(data), h)
}

// Hash reads the file into h and returns the digest bytes.
func (f File) Hash(ctx Context, h hash.Hash) ([]byte, error) {
	handle, err := f.OpenRead(ctx, OpenExisting)
	if err != nil {
		return nil, err
	}

	sum, hashErr := HashReader(handle, h)
	closeErr := handle.Close()
	if hashErr != nil {
		return nil, hashErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return sum, nil
}

// HashHex reads the file into h and returns the digest as lowercase hex.
func (f File) HashHex(ctx Context, h hash.Hash) (string, error) {
	sum, err := f.Hash(ctx, h)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum), nil
}

// HashHexBytes returns the digest for data as lowercase hex.
//
// h must not be nil.
func HashHexBytes(data []byte, h hash.Hash) string {
	return hex.EncodeToString(HashBytes(data, h))
}

// HashHexString returns the digest for data as lowercase hex.
//
// h must not be nil.
func HashHexString(data string, h hash.Hash) string {
	return HashHexBytes([]byte(data), h)
}

// HashHexReader reads src into h and returns the digest as lowercase hex.
func HashHexReader(src io.Reader, h hash.Hash) (string, error) {
	sum, err := HashReader(src, h)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum), nil
}
