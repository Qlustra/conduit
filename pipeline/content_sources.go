package pipeline

import "github.com/qlustra/conduit/layout"

type fileByteSource struct{ files []layout.File }

func (s fileByteSource) snapshotItems() ([]Item[Blob], error) {
	items := make([]Item[Blob], 0, len(s.files))
	for _, file := range s.files {
		items = append(items, itemFromFile(file))
	}
	return items, nil
}

type dirFilesByteSource struct{ dir layout.Dir }

func (s dirFilesByteSource) snapshotItems() ([]Item[Blob], error) {
	files, err := s.dir.FileList()
	if err != nil {
		return []Item[Blob]{}, err
	}
	items := make([]Item[Blob], 0, len(files))
	for _, file := range files {
		items = append(items, itemFromFile(file))
	}
	return items, nil
}

type blobByteSource struct{ blobs []Blob }

func (s blobByteSource) snapshotItems() ([]Item[Blob], error) {
	items := make([]Item[Blob], 0, len(s.blobs))
	for _, blob := range s.blobs {
		items = append(items, itemFromBlob(blob))
	}
	return items, nil
}
