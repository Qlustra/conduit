// Package mgmt provides explicit management operations over Conduit layouts.
// Controllers retain supplied layout state. Operations are synchronous; callers
// must serialize mutations sharing layout values. No multi-file transaction or
// protection against hostile filesystem races is implied.
package mgmt

import (
	"context"
	"errors"
	"fmt"
	"github.com/qlustra/conduit/layout"
)

type Operation string

const (
	OpAt        Operation = "at"
	OpList      Operation = "list"
	OpCreate    Operation = "create"
	OpDetect    Operation = "detect"
	OpInspect   Operation = "inspect"
	OpLoad      Operation = "load"
	OpValidate  Operation = "validate"
	OpScaffold  Operation = "scaffold"
	OpSave      Operation = "save"
	OpBootstrap Operation = "bootstrap"
	OpRefresh   Operation = "refresh"
)

type Capabilities []Operation

func (c Capabilities) Has(op Operation) bool {
	for _, x := range c {
		if x == op {
			return true
		}
	}
	return false
}

// Effect records completed, skipped, and failed work, including partial writes.
type Effect struct {
	Phase, Path, Status string
	// Applied identifies effects known to have occurred.
	Applied bool
	// MayHaveApplied records uncertainty after a failed mutation attempt. For
	// example, a failed direct write may already have truncated its destination.
	MayHaveApplied bool
	Err            error
}
type Outcome struct{ Effects []Effect }

func (o Outcome) Applied() bool {
	for _, e := range o.Effects {
		if e.Applied {
			return true
		}
	}
	return false
}

// MayHaveApplied reports known or possible filesystem effects. A false Applied
// result alone does not prove that a failed operation left storage unchanged.
func (o Outcome) MayHaveApplied() bool {
	for _, e := range o.Effects {
		if e.Applied || e.MayHaveApplied {
			return true
		}
	}
	return false
}

func (o *Outcome) add(phase, path, status string, applied bool, err error) {
	o.Effects = append(o.Effects, Effect{Phase: phase, Path: path, Status: status, Applied: applied, Err: err})
}

type Diagnostic struct {
	Phase, Path, Message string
	Err                  error
}
type Diagnostics struct{ Entries []Diagnostic }

func (d Diagnostics) HasErrors() bool { return len(d.Entries) > 0 }
func (d Diagnostics) Err() error {
	var errs []error
	for _, e := range d.Entries {
		errs = append(errs, fmt.Errorf("%s %s: %s: %w", e.Phase, e.Path, e.Message, e.Err))
	}
	return errors.Join(errs...)
}
func (d *Diagnostics) add(phase, path string, err error) {
	d.Entries = append(d.Entries, Diagnostic{phase, path, err.Error(), err})
}

type Item struct {
	Key, Path, Kind, Role string
	Present               bool
}
type Inventory struct {
	Entries     []Item
	Diagnostics Diagnostics
}

type Source uint8

const (
	DefaultSource Source = iota
	Memory
	Disk
)

type DiagnosticMode uint8

const (
	FailFast DiagnosticMode = iota
	CollectAll
)

type Traversal uint8

const (
	DefaultTraversal Traversal = iota
	Cached
	Discovered
)

type DirtyPolicy uint8

const (
	RejectDirty DirtyPolicy = iota
	ReplaceDirty
)

type EffectsPolicy uint8

const (
	Persist EffectsPolicy = iota
	MemoryOnly
)

type MissingPolicy uint8

const (
	DefaultMissing MissingPolicy = iota
	FailIfMissing
	CreateMissing
)

type UnsupportedPolicy uint8

const (
	ErrorUnsupported UnsupportedPolicy = iota
	SkipUnsupported
)

type DestinationPolicy uint8

const (
	AbsentOnly DestinationPolicy = iota
	AbsentOrEmpty
)

type Eligibility uint8

const (
	DirtyOnly Eligibility = iota
	Rewrite
)

type InspectionPolicy struct {
	Traversal   Traversal
	Diagnostics DiagnosticMode
}
type LoadPolicy struct {
	Traversal   Traversal
	Dirty       DirtyPolicy
	Diagnostics DiagnosticMode
}
type ValidationPolicy struct {
	Source      Source
	Diagnostics DiagnosticMode
	Traversal   Traversal
}
type ScaffoldPolicy struct{ Files bool }
type SavePolicy struct {
	Eligibility Eligibility
	Missing     MissingPolicy
}
type BootstrapPolicy struct {
	Dirty       DirtyPolicy
	Unsupported UnsupportedPolicy
	NewRootOnly bool
}
type RefreshPolicy struct {
	Source  Source
	Effects EffectsPolicy
	Dirty   DirtyPolicy
	Missing MissingPolicy
}
type ReadPolicy struct{ Source Source }
type UpdatePolicy struct {
	CheckConflict bool
	Source        Source
	Effects       EffectsPolicy
	Dirty         DirtyPolicy
}
type BindPolicy struct{ RequireExisting bool }
type ListPolicy struct {
	Validate    bool
	Diagnostics DiagnosticMode
}
type CreatePolicy struct {
	Destination DestinationPolicy
	Parent      MissingPolicy
	Unsupported UnsupportedPolicy
}

// ScopeOptions restricts operations when Ops is non-nil. Nil selects applicable
// operations. Restriction does not authorize an intrinsically unsupported recipe.
type ScopeOptions struct{ Ops []Operation }
type SpaceOptions struct {
	Context layout.Context
	Scope   ScopeOptions
}
type ProjectOptions struct {
	Space   SpaceOptions
	Signals []string
}
type CollectionOptions struct{ Scope ScopeOptions }
type DocumentOptions[T any] struct {
	Validate func(context.Context, T) error
}

type Bound interface {
	Binding() Binding
	Capabilities() Capabilities
}
type Scope[L any] interface {
	Bound
	Layout() *L
	Inspect(context.Context, InspectionPolicy) (Inventory, error)
	Load(context.Context, LoadPolicy) (Outcome, error)
	Validate(context.Context, ValidationPolicy) (Diagnostics, error)
	Scaffold(context.Context, ScaffoldPolicy) (Outcome, error)
	Save(context.Context, SavePolicy) (Outcome, error)
	Refresh(context.Context, RefreshPolicy) (Outcome, error)
}
type Space[L any] interface {
	Bound
	Root() layout.Dir
	Layout() *L
	Scope() Scope[L]
}
type Project[L any] interface {
	Space[L]
	Detect(context.Context, DetectionPolicy) (Detection, error)
}
type Collection[L any] interface {
	Bound
	List(context.Context, ListPolicy) (Inventory, error)
	At(string, BindPolicy) (Scope[L], error)
}
type Document[T any] interface {
	Bound
	Read(context.Context, ReadPolicy) (T, error)
	Load(context.Context, LoadPolicy) (Outcome, error)
	Save(context.Context, SavePolicy) (Outcome, error)
	Update(context.Context, func(*T) error, UpdatePolicy) (Outcome, error)
	Validate(context.Context, ValidationPolicy) (Diagnostics, error)
}

type Initialization[L, I any] struct {
	Initialize func(context.Context, *L, I) error
	Context    func(context.Context, *L) error
	Validate   func(context.Context, *L) error
	AfterApply func(context.Context, Outcome) error
}
type Creation[L, I any] Initialization[L, I]
type Created[L any] struct {
	Value   Scope[L]
	Outcome Outcome
}
type PostApplyError struct{ Err error }

func (e *PostApplyError) Error() string {
	return "operation applied; after-apply callback: " + e.Err.Error()
}
func (e *PostApplyError) Unwrap() error { return e.Err }

var ErrUnsupported = errors.New("unsupported management capability")
var ErrDirty = errors.New("operation would replace dirty content")
var ErrConflict = errors.New("filesystem changed during operation")

func ioDefaults(c layout.Context) layout.Context {
	if c.DirMode == 0 {
		c.DirMode = 0755
	}
	if c.FileMode == 0 {
		c.FileMode = 0644
	}
	if c.ExecMode == 0 {
		c.ExecMode = 0755
	}
	return c
}
