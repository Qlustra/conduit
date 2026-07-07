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
	// ToTarget records byte writes to one attached target file.
	ToTarget ByteWriteResult
	// ToTargets records byte writes to attached target files.
	ToTargets ByteWriteResult
	// To records byte To sink outcomes.
	To ByteWriteResult
	// ToDir records byte ToDir sink outcomes.
	ToDir ByteWriteResult
	// ToFiles records byte ToFiles sink outcomes.
	ToFiles ByteWriteResult

	// Handover records typed Bridge, Compile, or Expand outcomes.
	Handover HandoverResult
}

// DeepOperationResult records per-item deep layout operation outcomes.
type DeepOperationResult struct {
	// Items contains per-item deep operation results.
	Items []DeepItemResult
}

// DeepItemResult records one typed item deep operation outcome.
type DeepItemResult struct {
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

// HandoverKind records which typed handover task produced a result.
type HandoverKind uint8

const (
	// HandoverUnknown indicates no typed handover kind was recorded.
	HandoverUnknown HandoverKind = iota

	// HandoverBridge indicates a Bridge task produced the result.
	HandoverBridge

	// HandoverExpand indicates an Expand task produced the result.
	HandoverExpand

	// HandoverCompile indicates a Compile task produced the result.
	HandoverCompile
)

// HandoverResult records typed inter-task handover outcomes.
type HandoverResult struct {
	// Kind identifies the handover task kind.
	Kind HandoverKind
	// Items contains per-target handover results.
	Items []HandoverItemResult
}

// HandoverItemResult records one target item populated by a handover task.
type HandoverItemResult struct {
	// OriginKey is the origin item key for one-to-one and one-to-many handovers.
	OriginKey string
	// OriginName is the origin item name.
	OriginName string
	// OriginPath is the origin item path metadata.
	OriginPath string
	// OriginFile is the origin backing file path, when present.
	OriginFile string
	// OriginDir is the origin backing directory path, when present.
	OriginDir string
	// OriginKeys are the origin item keys for many-to-one handovers.
	OriginKeys []string

	// TargetKey is the target item key.
	TargetKey string
	// TargetName is the target item name.
	TargetName string
	// TargetPath is the target item path metadata.
	TargetPath string
	// TargetFile is the target backing file path, when present.
	TargetFile string
	// TargetDir is the target backing directory path, when present.
	TargetDir string

	// Err records the handover error for this item, if any.
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
