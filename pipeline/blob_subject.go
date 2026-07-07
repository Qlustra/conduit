package pipeline

import (
	"sort"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// BlobSubject is a pointer-backed byte subject that can be shared across tasks.
type BlobSubject struct {
	mu   sync.RWMutex
	item Item[Blob]
}

// BlobSubjectFromBlob returns a shared subject initialized from blob.
func BlobSubjectFromBlob(blob Blob) *BlobSubject {
	subject := &BlobSubject{}
	subject.put(itemFromBlob(blob))
	return subject
}

// BlobSubjectForFile returns a shared subject initialized from file metadata.
func BlobSubjectForFile(file layout.File) *BlobSubject {
	subject := &BlobSubject{}
	subject.put(itemFromFile(file))
	return subject
}

// Snapshot returns the subject's current item state.
func (s *BlobSubject) Snapshot() Item[Blob] {
	if s == nil {
		return Item[Blob]{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return normalizeBlobItem(s.item)
}

// Clear removes the subject's in-memory bytes while preserving metadata.
func (s *BlobSubject) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.item.Data = nil
	s.item.Value.Data = nil
}

func (s *BlobSubject) put(item Item[Blob]) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.item = normalizeBlobItem(item)
}

type blobSubjectSource struct{ subject *BlobSubject }

func (s blobSubjectSource) snapshotItems() []Item[Blob] {
	if s.subject == nil {
		return nil
	}
	return []Item[Blob]{s.subject.Snapshot()}
}

type blobSubjectsSource struct{ subjects []*BlobSubject }

func (s blobSubjectsSource) snapshotItems() []Item[Blob] {
	items := make([]Item[Blob], 0, len(s.subjects))
	for _, subject := range s.subjects {
		if subject == nil {
			continue
		}
		items = append(items, subject.Snapshot())
	}
	return items
}

// BlobSubjectSet stores keyed shared byte subjects.
type BlobSubjectSet struct {
	mu       sync.RWMutex
	subjects map[string]*BlobSubject
}

// BlobSubjects returns an empty keyed subject set.
func BlobSubjects() *BlobSubjectSet {
	return &BlobSubjectSet{subjects: make(map[string]*BlobSubject)}
}

// At returns the subject for key, creating it when needed.
func (s *BlobSubjectSet) At(key string) *BlobSubject {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if subject, ok := s.subjects[key]; ok {
		return subject
	}
	subject := BlobSubjectFromBlob(Blob{Key: key, Name: key, Path: key})
	s.subjects[key] = subject
	return subject
}

// Get returns the subject for key when present.
func (s *BlobSubjectSet) Get(key string) (*BlobSubject, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	subject, ok := s.subjects[key]
	return subject, ok
}

// Keys returns the set's subject keys in sorted order.
func (s *BlobSubjectSet) Keys() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.subjects))
	for key := range s.subjects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Subjects returns the set's subjects ordered by sorted key.
func (s *BlobSubjectSet) Subjects() []*BlobSubject {
	keys := s.Keys()
	if len(keys) == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	subjects := make([]*BlobSubject, 0, len(keys))
	for _, key := range keys {
		subjects = append(subjects, s.subjects[key])
	}
	return subjects
}

type blobSubjectSetSource struct{ subjects *BlobSubjectSet }

func (s blobSubjectSetSource) snapshotItems() []Item[Blob] {
	if s.subjects == nil {
		return nil
	}
	keys := s.subjects.Keys()
	items := make([]Item[Blob], 0, len(keys))
	for _, key := range keys {
		subject, ok := s.subjects.Get(key)
		if !ok || subject == nil {
			continue
		}
		items = append(items, subject.Snapshot())
	}
	return items
}

func normalizeBlobItem(item Item[Blob]) Item[Blob] {
	data := item.Data
	if data == nil && item.Value.Data != nil {
		data = item.Value.Data
	}
	data = cloneBytes(data)

	key := item.Key
	name := item.Name
	path := item.Path
	if key == "" {
		key = item.Value.Key
	}
	if name == "" {
		name = item.Value.Name
	}
	if path == "" {
		path = item.Value.Path
	}
	if hasFile(item.File) {
		fallbackName := item.File.Base()
		if name == "" {
			name = fallbackName
		}
		if path == "" {
			path = itemPathFromFile(item.File, fallbackName)
		}
	}
	key, name, path = normalizeBlobMetadata(Blob{Key: key, Name: name, Path: path})

	blob := item.Value
	blob.Key = key
	blob.Name = name
	blob.Path = path
	blob.Data = cloneBytes(data)

	item.Key = key
	item.Name = name
	item.Path = path
	item.Data = data
	item.Value = blob
	return item
}
