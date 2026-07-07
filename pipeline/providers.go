package pipeline

import "github.com/qlustra/conduit/layout"

// TaskFromFile returns a single-subject byte task backed by file.
func TaskFromFile(name string, file layout.File) *ByteSingleTask {
	return newByteSingleTask(name, itemFromFile(file))
}

// TaskFromFiles returns a multi-subject byte task backed by files.
func TaskFromFiles(name string, files ...layout.File) *ByteMultiTask {
	items := make([]Item[Blob], 0, len(files))
	for _, file := range files {
		items = append(items, itemFromFile(file))
	}
	return newByteMultiTask(name, items)
}

// TaskFromBlob returns a single-subject byte task backed by blob data.
func TaskFromBlob(name string, blob Blob) *ByteSingleTask {
	return newByteSingleTask(name, itemFromBlob(blob))
}

// TaskFromBlobs returns a multi-subject byte task backed by blob data.
func TaskFromBlobs(name string, blobs ...Blob) *ByteMultiTask {
	items := make([]Item[Blob], 0, len(blobs))
	for _, blob := range blobs {
		items = append(items, itemFromBlob(blob))
	}
	return newByteMultiTask(name, items)
}

// TaskFromBlobSubject returns a single-subject byte task backed by subject.
func TaskFromBlobSubject(name string, subject *BlobSubject) *ByteSingleTask {
	return &ByteSingleTask{task: &byteTask{name: name, kind: subjectSingle, source: blobSubjectSource{subject: subject}}}
}

// TaskFromBlobSubjects returns a multi-subject byte task backed by subjects.
func TaskFromBlobSubjects(name string, subjects ...*BlobSubject) *ByteMultiTask {
	return &ByteMultiTask{task: &byteTask{name: name, kind: subjectMulti, source: blobSubjectsSource{subjects: subjects}}}
}

// TaskFromBlobSubjectSet returns a multi-subject byte task backed by subjects.
func TaskFromBlobSubjectSet(name string, subjects *BlobSubjectSet) *ByteMultiTask {
	return &ByteMultiTask{task: &byteTask{name: name, kind: subjectMulti, source: blobSubjectSetSource{subjects: subjects}}}
}

// TaskFromSlot returns a single-subject typed task for slot itself.
func TaskFromSlot[T any](name string, slot *layout.Slot[T]) *TypedSingleTask[*layout.Slot[T]] {
	return newTypedSingleTask(name, itemFromSlot(slot))
}

// TaskFromSlots returns a multi-subject typed task for slots themselves.
func TaskFromSlots[T any](name string, slots ...*layout.Slot[T]) *TypedMultiTask[*layout.Slot[T]] {
	items := make([]Item[*layout.Slot[T]], 0, len(slots))
	for _, slot := range slots {
		items = append(items, itemFromSlot(slot))
	}
	return newTypedMultiTask(name, items)
}

// TaskFromSlotEntries snapshots cached slot entries into a typed multi task.
func TaskFromSlotEntries[T any](name string, slots ...*layout.Slot[T]) *TypedMultiTask[T] {
	var items []Item[T]
	for _, slot := range slots {
		for _, entry := range slot.Entries() {
			items = append(items, itemFromSlotEntry(slot, entry))
		}
	}
	return newTypedMultiTask(name, items)
}

// TaskFromFileSlot returns a single-subject typed task for file slot itself.
func TaskFromFileSlot[T any](name string, slot *layout.FileSlot[T]) *TypedSingleTask[*layout.FileSlot[T]] {
	return newTypedSingleTask(name, itemFromFileSlot(slot))
}

// TaskFromFileSlots returns a multi-subject typed task for file slots themselves.
func TaskFromFileSlots[T any](name string, slots ...*layout.FileSlot[T]) *TypedMultiTask[*layout.FileSlot[T]] {
	items := make([]Item[*layout.FileSlot[T]], 0, len(slots))
	for _, slot := range slots {
		items = append(items, itemFromFileSlot(slot))
	}
	return newTypedMultiTask(name, items)
}

// TaskFromFileSlotEntries snapshots cached file-slot entries into a typed multi task.
func TaskFromFileSlotEntries[T any](name string, slots ...*layout.FileSlot[T]) *TypedMultiTask[T] {
	var items []Item[T]
	for _, slot := range slots {
		for _, entry := range slot.Entries() {
			items = append(items, itemFromFileSlotEntry(slot, entry))
		}
	}
	return newTypedMultiTask(name, items)
}
