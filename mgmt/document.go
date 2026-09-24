package mgmt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/qlustra/conduit/layout"
)

const (
	OpRead   Operation = "read"
	OpUpdate Operation = "update"
)

// DocumentNode is the typed participant contract used by Document. Pointers to
// the built-in JSON, YAML, TOML, and other Format wrappers satisfy this contract.
// Codec operations must not mutate their input or the participant's state.
type DocumentNode[T any] interface {
	layout.Pather
	layout.ContentState
	layout.Loadable
	layout.WritePreparer
	layout.StateSnapshotter
	Get() (T, bool)
	Set(T)
	Read() (T, error)
	EncodeValue(T) ([]byte, error)
	DecodeValue([]byte) (T, error)
}

type document[T any] struct {
	binding Binding
	node    DocumentNode[T]
	options DocumentOptions[T]
}

// BindDocument creates a typed operational view over the supplied participant,
// preserving its path, cached content, and tracked state. Node must be a pointer.
// Domain validation is applied by Load, Save, Update, and Validate; Read only
// obtains a value, allowing callers to inspect or repair invalid domain content.
func BindDocument[T any](owner Bound, node DocumentNode[T], options DocumentOptions[T]) (Document[T], error) {
	binding, err := childBinding(owner, node, ScopeOptions{})
	if err != nil {
		return nil, err
	}
	return &document[T]{binding: binding, node: node, options: options}, nil
}

func (d *document[T]) Binding() Binding { return d.binding }
func (d *document[T]) Capabilities() Capabilities {
	return Capabilities{OpRead, OpLoad, OpSave, OpUpdate, OpValidate}
}

func (d *document[T]) core() *scope[DocumentNode[T]] {
	return &scope[DocumentNode[T]]{binding: d.binding, tree: &d.node}
}

// Read defaults to disk and never replaces the bound cache. Memory reads return
// an independent codec clone, so maps, slices, and pointers cannot mutate cached
// values through the returned result. Only codec-representable fields are copied.
func (d *document[T]) Read(ctx context.Context, policy ReadPolicy) (T, error) {
	var zero T
	if policy.Source > Disk {
		return zero, fmt.Errorf("invalid document read policy")
	}
	if err := checkBinding(d.binding); err != nil {
		return zero, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if policy.Source == Memory {
		value, ok := d.node.Get()
		if !ok {
			return zero, fmt.Errorf("document content is not loaded: %s", d.binding.path)
		}
		return d.clone(value)
	}
	if err := guard(d.binding.context.root, d.binding.path, "file"); err != nil {
		return zero, err
	}
	value, err := d.node.Read()
	if err != nil {
		return zero, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return value, nil
}

// Load replaces memory only after successful decoding and validation. Unlike a
// whole-scope inventory/load, loading an absent typed document is an error and
// preserves any previous cached value. ReplaceDirty explicitly permits replacing
// authored memory. Successful loading retains the participant's loaded state.
func (d *document[T]) Load(ctx context.Context, policy LoadPolicy) (out Outcome, err error) {
	if policy.Traversal > Discovered || policy.Diagnostics > CollectAll || policy.Dirty > ReplaceDirty {
		return out, fmt.Errorf("invalid document load policy")
	}
	if err = require(d.binding, OpLoad); err != nil {
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if policy.Dirty == RejectDirty && d.node.MemoryState() == layout.MemoryDirty {
		return out, fmt.Errorf("%w: %s", ErrDirty, d.binding.path)
	}
	if err = guard(d.binding.context.root, d.binding.path, "file"); err != nil {
		return out, err
	}
	restore, err := d.preserve()
	if err != nil {
		return out, err
	}
	committed := false
	defer func() {
		if !committed {
			restore()
		}
	}()
	exists, err := d.node.Load()
	if err == nil && !exists {
		err = &os.PathError{Op: "load", Path: d.binding.path, Err: os.ErrNotExist}
	}
	if err != nil {
		out.add("load", d.binding.path, "failed", false, err)
		return out, err
	}
	if _, err = d.Validate(ctx, ValidationPolicy{Source: Memory, Diagnostics: policy.Diagnostics}); err != nil {
		out.add("validate", d.binding.path, "failed", false, err)
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	committed = true
	out.add("load", d.binding.path, "loaded", false, nil)
	return out, nil
}

// Validate defaults to memory. Disk validation reads an independent value and
// temporarily binds a clone for wrapper rules, then restores the original cache
// and state. DocumentOptions.Validate also receives an isolated codec clone.
func (d *document[T]) Validate(ctx context.Context, policy ValidationPolicy) (Diagnostics, error) {
	var out Diagnostics
	if policy.Source > Disk || policy.Diagnostics > CollectAll || policy.Traversal > Discovered {
		return out, fmt.Errorf("invalid document validation policy")
	}
	if err := require(d.binding, OpValidate); err != nil {
		return out, err
	}
	source := policy.Source
	if source == DefaultSource {
		source = Memory
	}
	value, err := d.Read(ctx, ReadPolicy{Source: source})
	if err != nil {
		out.add("validate", d.binding.path, err)
		return out, err
	}
	return d.validateValue(ctx, value, policy.Diagnostics)
}

// Save validates the cached value and prepares its encoded bytes before writing.
// It uses the same eligibility, missing-target, and write policy as Scope.Save.
func (d *document[T]) Save(ctx context.Context, policy SavePolicy) (Outcome, error) {
	var out Outcome
	if policy.Eligibility > Rewrite || policy.Missing > CreateMissing {
		return out, fmt.Errorf("invalid document save policy")
	}
	if err := require(d.binding, OpSave); err != nil {
		return out, err
	}
	if _, err := d.Validate(ctx, ValidationPolicy{Source: Memory}); err != nil {
		out.add("validate", d.binding.path, "failed", false, err)
		return out, err
	}
	writes, out, err := prepareWrites(d.binding, policy, nil)
	if err != nil {
		return out, err
	}
	return applyWrites(ctx, d.binding, writes, out, false)
}

// Update defaults to disk, transforms an isolated value, validates and encodes
// it, and then replaces the bound cache. With MemoryOnly it leaves that value
// dirty without writing. Persistence updates the same node's tracked state;
// write failure leaves the prepared candidate dirty for inspection or retry.
//
// Source Memory intentionally edits the current authored value. Source Disk
// rejects a preexisting dirty value unless ReplaceDirty is explicit. An existing
// file is required for persisted updates; initialization belongs to Bootstrap.
// CheckConflict compares disk bytes before transformation and before persistence.
// It is best effort, not an atomic compare-and-swap or an exclusive write lock.
func (d *document[T]) Update(ctx context.Context, transform func(*T) error, policy UpdatePolicy) (out Outcome, err error) {
	if policy.Source > Disk || policy.Effects > MemoryOnly || policy.Dirty > ReplaceDirty {
		return out, fmt.Errorf("invalid document update policy")
	}
	if transform == nil {
		return out, fmt.Errorf("document transform must not be nil")
	}
	if policy.CheckConflict && policy.Effects == MemoryOnly {
		return out, fmt.Errorf("%w: conflict checking requires persistence", ErrUnsupported)
	}
	if err = require(d.binding, OpSave); err != nil {
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if policy.Source != Memory && policy.Dirty == RejectDirty && d.node.MemoryState() == layout.MemoryDirty {
		return out, fmt.Errorf("%w: %s", ErrDirty, d.binding.path)
	}

	var baseline []byte
	if policy.CheckConflict {
		if err = guard(d.binding.context.root, d.binding.path, "file"); err != nil {
			return out, err
		}
		baseline, err = os.ReadFile(d.binding.path)
		if err != nil {
			return out, err
		}
	}
	var value T
	if policy.CheckConflict && policy.Source != Memory {
		value, err = d.node.DecodeValue(bytes.Clone(baseline))
	} else {
		value, err = d.Read(ctx, ReadPolicy{Source: policy.Source})
	}
	if err != nil {
		out.add("read", d.binding.path, "failed", false, err)
		return out, err
	}
	candidate, err := d.clone(value)
	if err != nil {
		return out, err
	}
	if err = transform(&candidate); err != nil {
		out.add("transform", d.binding.path, "failed", false, err)
		return out, err
	}
	if _, err = d.validateValue(ctx, candidate, FailFast); err != nil {
		out.add("validate", d.binding.path, "failed", false, err)
		return out, err
	}
	// Encode before changing cached content, and detach references retained by
	// the transform. Codec-ignored fields are outside this management contract.
	candidate, err = d.clone(candidate)
	if err != nil {
		out.add("prepare", d.binding.path, "failed", false, err)
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if policy.Effects == Persist {
		if err = guard(d.binding.context.root, d.binding.path, "file"); err != nil {
			return out, err
		}
		if _, err = os.Stat(d.binding.path); err != nil {
			return out, err
		}
	}
	restore, err := d.preserve()
	if err != nil {
		return out, err
	}
	committed := false
	defer func() {
		if !committed {
			restore()
		}
	}()
	d.node.Set(candidate)
	prepared, err := d.node.PrepareWrite()
	if err != nil {
		out.add("prepare", d.binding.path, "failed", false, err)
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if policy.Effects == MemoryOnly {
		committed = true
		out.add("update", d.binding.path, "dirty", false, nil)
		return out, nil
	}
	if err = guard(d.binding.context.root, d.binding.path, "file"); err != nil {
		return out, err
	}
	if policy.CheckConflict {
		current, readErr := os.ReadFile(d.binding.path)
		if readErr != nil || !bytes.Equal(baseline, current) {
			err = fmt.Errorf("%w: %s", ErrConflict, d.binding.path)
			if readErr != nil {
				err = fmt.Errorf("%w: %v", err, readErr)
			}
			out.add("prepare", d.binding.path, "conflict", false, err)
			return out, err
		}
	}
	// After preflight, retain the candidate even if persistence fails. The
	// prepared-write contract marks it synced only after successful publication.
	committed = true
	err = prepared.Write(d.binding.context.io)
	status := "written"
	if err != nil {
		status = "failed"
	}
	out.add("write", d.binding.path, status, err == nil, err)
	if err != nil && !errors.Is(err, layout.ErrPreparedWriteStale) {
		// PreparedWrite does not expose the failure's publication stage. Preserve
		// uncertainty: direct writes can truncate before returning an error.
		out.Effects[len(out.Effects)-1].MayHaveApplied = true
	}
	return out, err
}

func (d *document[T]) clone(value T) (T, error) {
	data, err := d.node.EncodeValue(value)
	if err != nil {
		var zero T
		return zero, err
	}
	return d.node.DecodeValue(bytes.Clone(data))
}

func (d *document[T]) preserve() (func(), error) {
	if node, ok := d.node.(interface{ PreserveState() func() }); ok {
		return node.PreserveState(), nil
	}
	return d.node.SnapshotState()
}

func (d *document[T]) validateValue(ctx context.Context, value T, mode DiagnosticMode) (Diagnostics, error) {
	var out Diagnostics
	if err := ctx.Err(); err != nil {
		return out, err
	}
	candidate, err := d.clone(value)
	if err != nil {
		out.add("validate", d.binding.path, err)
		return out, err
	}
	restore, err := d.preserve()
	if err != nil {
		out.add("validate", d.binding.path, err)
		return out, err
	}
	defer restore()
	d.node.Set(candidate)
	out, err = d.core().Validate(ctx, ValidationPolicy{Source: Memory, Diagnostics: mode, Traversal: Cached})
	if err != nil && mode == FailFast {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if d.options.Validate != nil {
		// The typed rule receives its own copy: accidental changes to nested
		// maps or slices cannot alter either the candidate or the bound cache.
		ruleValue, cloneErr := d.clone(value)
		if cloneErr != nil {
			out.add("validate", d.binding.path, cloneErr)
		} else if ruleErr := d.options.Validate(ctx, ruleValue); ruleErr != nil {
			out.add("validate", d.binding.path, ruleErr)
		}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return out, out.Err()
}
