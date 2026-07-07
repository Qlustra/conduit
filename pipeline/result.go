package pipeline

import "github.com/qlustra/conduit/layout"

// Result records a pipeline run.
type Result struct {
	// Tasks contains task results in execution order.
	Tasks []TaskResult
}

// TaskResult records one task run.
type TaskResult struct {
	// Name is the task name.
	Name string
	// Status is the task outcome.
	Status Status
	// Err is the task error when Status is StatusFailed.
	Err error

	// EnsureDeep records typed EnsureDeep sink outcomes.
	EnsureDeep DeepOperationResult
	// DefaultDeep records typed DefaultDeep sink outcomes.
	DefaultDeep DeepOperationResult
	// RenderDeep records typed RenderDeep sink outcomes.
	RenderDeep DeepOperationResult
	// SyncDeep records typed SyncDeep sink outcomes.
	SyncDeep DeepOperationResult
	// ValidateDeep records typed ValidateDeep sink outcomes.
	ValidateDeep DeepOperationResult

	// WriteBack records byte WriteBack sink outcomes.
	WriteBack ByteWriteResult
	// To records byte To sink outcomes.
	To ByteWriteResult
	// ToDir records byte ToDir sink outcomes.
	ToDir ByteWriteResult
	// ToFiles records byte ToFiles sink outcomes.
	ToFiles ByteWriteResult
}

func (r *TaskResult) recordDeepOperation(kind slotSinkOperation, entry SlotOperationResult) {
	switch kind {
	case slotSinkEnsureDeep:
		r.EnsureDeep.Items = append(r.EnsureDeep.Items, entry)
	case slotSinkDefaultDeep:
		r.DefaultDeep.Items = append(r.DefaultDeep.Items, entry)
	case slotSinkRenderDeep:
		r.RenderDeep.Items = append(r.RenderDeep.Items, entry)
	case slotSinkSyncDeep:
		r.SyncDeep.Items = append(r.SyncDeep.Items, entry)
	case slotSinkValidateDeep:
		r.ValidateDeep.Items = append(r.ValidateDeep.Items, entry)
	}
}

func (r *TaskResult) recordByteWrite(kind byteSinkOperation, entry ByteWriteItemResult) {
	switch kind {
	case byteSinkWriteBack:
		r.WriteBack.Items = append(r.WriteBack.Items, entry)
	case byteSinkToFile:
		r.To.Items = append(r.To.Items, entry)
	case byteSinkToDir:
		r.ToDir.Items = append(r.ToDir.Items, entry)
	case byteSinkToFiles:
		r.ToFiles.Items = append(r.ToFiles.Items, entry)
	}
}

// DeepOperationResult records per-item deep layout operation outcomes.
type DeepOperationResult struct {
	// Items contains per-item deep operation results.
	Items []SlotOperationResult
}

// SlotOperationResult records one typed item deep operation outcome.
type SlotOperationResult struct {
	// Key is the item key.
	Key string
	// Name is the item name.
	Name string
	// Path is the item path metadata.
	Path string
	// File is the backing file path, when present.
	File string
	// Dir is the backing directory path, when present.
	Dir string
	// Result is the layout operation result code.
	Result layout.ResultCode
	// Entries contains path-level layout report entries captured during traversal.
	Entries []layout.Entry
	// Err records the item-level operation error, if any.
	Err error
}

// ByteWriteResult records per-item byte sink outcomes.
type ByteWriteResult struct {
	// Items contains per-item byte write results.
	Items []ByteWriteItemResult
}

// ByteWriteItemResult records one byte write outcome.
type ByteWriteItemResult struct {
	// Key is the item key.
	Key string
	// Name is the item name.
	Name string
	// Path is the item path metadata.
	Path string
	// File is the written destination file path.
	File string
	// Bytes is the number of bytes written.
	Bytes int
	// Err records the write error, if any.
	Err error
}

// Status records task outcome.
type Status uint8

const (
	// StatusRan indicates a task completed successfully.
	StatusRan Status = iota + 1

	// StatusFailed indicates a task returned an error.
	StatusFailed
)

func failTask(result TaskResult, err error) (TaskResult, error) {
	result.Status = StatusFailed
	result.Err = err
	return result, err
}
