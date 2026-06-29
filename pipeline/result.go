package pipeline

import "github.com/qlustra/conduit/layout"

// Result records a pipeline run.
type Result struct {
	Tasks []TaskResult
}

// TaskResult records one task run.
type TaskResult struct {
	Name   string
	Status Status
	Err    error

	EnsureDeep   DeepOperationResult
	DefaultDeep  DeepOperationResult
	RenderDeep   DeepOperationResult
	SyncDeep     DeepOperationResult
	ValidateDeep DeepOperationResult

	WriteBack ByteWriteResult
	To        ByteWriteResult
	ToDir     ByteWriteResult
	ToFiles   ByteWriteResult

	Handover HandoverResult
}

// DeepOperationResult records per-item deep layout operation outcomes.
type DeepOperationResult struct {
	Items []DeepItemResult
}

// DeepItemResult records one typed item deep operation outcome.
type DeepItemResult struct {
	Key     string
	Name    string
	Path    string
	File    string
	Dir     string
	Result  layout.ResultCode
	Entries []layout.Entry
	Err     error
}

// ByteWriteResult records per-item byte sink outcomes.
type ByteWriteResult struct {
	Items []ByteWriteItemResult
}

// ByteWriteItemResult records one byte write outcome.
type ByteWriteItemResult struct {
	Key   string
	Name  string
	Path  string
	File  string
	Bytes int
	Err   error
}

// HandoverKind records which typed handover task produced a result.
type HandoverKind uint8

const (
	HandoverUnknown HandoverKind = iota
	HandoverBridge
	HandoverExpand
	HandoverCompile
)

// HandoverResult records typed inter-task handover outcomes.
type HandoverResult struct {
	Kind  HandoverKind
	Items []HandoverItemResult
}

// HandoverItemResult records one target item populated by a handover task.
type HandoverItemResult struct {
	OriginKey  string
	OriginName string
	OriginPath string
	OriginFile string
	OriginDir  string
	OriginKeys []string

	TargetKey  string
	TargetName string
	TargetPath string
	TargetFile string
	TargetDir  string

	Err error
}

// Status records task outcome.
type Status uint8

const (
	StatusRan Status = iota + 1
	StatusFailed
)

func failTask(result TaskResult, err error) (TaskResult, error) {
	result.Status = StatusFailed
	result.Err = err
	return result, err
}
