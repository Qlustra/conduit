package pipeline

import (
	"github.com/qlustra/conduit/layout"
)

// TaskFromFile returns a single-item byte task that snapshots one file when it runs.
func TaskFromFile(name string, file layout.File) *SingleContentTask {
	task := &SingleContentTask{stepper: newByteStepperTask(name, singleTask)}
	task.stepper.setSource(fileByteSource{files: []layout.File{file}})
	return task
}

// TaskFromBlob returns a single-item byte task initialized from one in-memory blob.
func TaskFromBlob(name string, blob Blob) *SingleContentTask {
	task := &SingleContentTask{stepper: newByteStepperTask(name, singleTask)}
	task.stepper.setSource(blobByteSource{blobs: []Blob{blob}})
	return task
}

// TaskFromFiles returns a multi-item byte task that snapshots the supplied files when it runs.
func TaskFromFiles(name string, files ...layout.File) *ContentCollectionTask {
	task := &ContentCollectionTask{stepper: newByteStepperTask(name, multiTask)}
	task.stepper.setSource(fileByteSource{files: files})
	return task
}

// TaskFromDir returns a multi-item byte task that snapshots the directory's direct regular files when it runs.
func TaskFromDir(name string, dir layout.Dir) *ContentCollectionTask {
	task := &ContentCollectionTask{stepper: newByteStepperTask(name, multiTask)}
	task.stepper.setSource(dirFilesByteSource{dir: dir})
	return task
}

// TaskFromBlobs returns a multi-item byte task initialized from the supplied in-memory blobs.
func TaskFromBlobs(name string, blobs ...Blob) *ContentCollectionTask {
	task := &ContentCollectionTask{stepper: newByteStepperTask(name, multiTask)}
	task.stepper.setSource(blobByteSource{blobs: blobs})
	return task
}

// TaskFromSlotEntries returns a typed task that snapshots cached entries from slot when it runs.
func TaskFromSlotEntries[T any](name string, slot *layout.Slot[T]) *MultiSlotTask[T] {
	task := &MultiSlotTask[T]{stepper: newSlotStepper[T](name)}
	task.stepper.setSource(createSlotSourceFromSlot[T](slot))
	return task
}

// TaskFromFileSlotEntries returns a typed task that snapshots cached entries from a file slot when it runs.
func TaskFromFileSlotEntries[T any](name string, slot *layout.FileSlot[T]) *MultiSlotTask[T] {
	task := &MultiSlotTask[T]{stepper: newSlotStepper[T](name)}
	task.stepper.setSource(createSlotSourceFromFileSlot[T](slot))
	return task
}
