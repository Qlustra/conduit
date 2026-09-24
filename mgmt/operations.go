package mgmt

import (
	"context"
	"errors"
	"fmt"
	"github.com/qlustra/conduit/layout"
	"os"
	"reflect"
)

func (s *scope[L]) Inspect(ctx context.Context, p InspectionPolicy) (Inventory, error) {
	var out Inventory
	b := s.binding
	if err := require(b, OpInspect); err != nil {
		return out, err
	}
	if p.Traversal > Discovered || p.Diagnostics > CollectAll {
		return out, fmt.Errorf("invalid inspection policy")
	}
	record := func(n node, e error) error {
		out.Diagnostics.add("inspect", n.path, e)
		if p.Diagnostics == FailFast {
			return e
		}
		return nil
	}
	if p.Traversal == Discovered {
		if err := discover(ctx, b, record); err != nil {
			return out, err
		}
	}
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		if n.kind == "group" {
			continue
		}
		item := Item{Path: n.path, Kind: n.kind, Role: n.meta.role}
		if e := guard(b.context.root, n.path, physicalKind(n)); e != nil {
			if e = record(n, e); e != nil {
				return out, e
			}
			out.Entries = append(out.Entries, item)
			continue
		}
		_, e := os.Lstat(n.path)
		item.Present = e == nil
		if e != nil && !os.IsNotExist(e) {
			if e = record(n, e); e != nil {
				return out, e
			}
		}
		if scanner, ok := n.value.(layout.Scannable); ok {
			_, e = scanner.Scan()
			if e != nil {
				if e = record(n, e); e != nil {
					return out, e
				}
			}
		}
		out.Entries = append(out.Entries, item)
	}
	return out, out.Diagnostics.Err()
}

func (s *scope[L]) Load(ctx context.Context, p LoadPolicy) (Outcome, error) {
	var out Outcome
	b := s.binding
	if err := require(b, OpLoad); err != nil {
		return out, err
	}
	if p.Traversal > Discovered || p.Dirty > ReplaceDirty || p.Diagnostics > CollectAll {
		return out, fmt.Errorf("invalid load policy")
	}
	var errs []error
	record := func(n node, e error) error {
		out.add("load", n.path, "failed", false, e)
		errs = append(errs, e)
		if p.Diagnostics == FailFast {
			return e
		}
		return nil
	}
	if p.Traversal != Cached {
		if err := discover(ctx, b, record); err != nil {
			return out, err
		}
	}
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	// Reject dirty conflicts before loading any participant.
	if p.Dirty == RejectDirty {
		for _, n := range ns {
			if state, ok := n.value.(layout.ContentState); ok && state.MemoryState() == layout.MemoryDirty {
				e := fmt.Errorf("%w: %s", ErrDirty, n.path)
				if p.Diagnostics == FailFast {
					out.add("load", n.path, "failed", false, e)
					return out, e
				}
				errs = append(errs, e)
				out.add("load", n.path, "failed", false, e)
			}
		}
		if len(errs) > 0 {
			return out, errors.Join(errs...)
		}
	}
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		if _, stateful := n.value.(layout.ContentState); stateful && !n.mutable {
			if e := record(n, fmt.Errorf("%w: value-backed stateful slot", ErrUnsupported)); e != nil {
				return out, e
			}
			continue
		}
		loader, ok := n.value.(layout.Loadable)
		if !ok {
			continue
		}
		if !n.mutable {
			if e := record(n, fmt.Errorf("%w: value-backed cached participant", ErrUnsupported)); e != nil {
				return out, e
			}
			continue
		}
		if e := guard(b.context.root, n.path, physicalKind(n)); e != nil {
			if e = record(n, e); e != nil {
				return out, e
			}
			continue
		}
		exists, e := loader.Load()
		if e != nil {
			if e = record(n, e); e != nil {
				return out, e
			}
			continue
		}
		status := "loaded"
		if !exists {
			status = "missing"
		}
		out.add("load", n.path, status, false, nil)
	}
	return out, errors.Join(errs...)
}

func (s *scope[L]) Validate(ctx context.Context, p ValidationPolicy) (Diagnostics, error) {
	return s.validate(ctx, p, false)
}
func (s *scope[L]) validate(ctx context.Context, p ValidationPolicy, preparing bool) (Diagnostics, error) {
	var out Diagnostics
	b := s.binding
	if err := require(b, OpValidate); err != nil {
		return out, err
	}
	if p.Source > Disk || p.Diagnostics > CollectAll || p.Traversal > Discovered {
		return out, fmt.Errorf("invalid validation policy")
	}
	record := func(n node, e error) error {
		out.add("validate", n.path, e)
		if p.Diagnostics == FailFast {
			return e
		}
		return nil
	}
	if p.Traversal == Discovered || (p.Source == Disk && p.Traversal == DefaultTraversal) {
		if e := discover(ctx, b, record); e != nil {
			return out, e
		}
	}
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	failed := map[string]bool{}
	var restores []func()
	defer func() {
		for i := len(restores) - 1; i >= 0; i-- {
			restores[i]()
		}
	}()
	if p.Source == Disk {
		for _, n := range ns {
			if err = ctx.Err(); err != nil {
				return out, err
			}
			if _, stateful := n.value.(layout.ContentState); stateful && !n.mutable {
				failed[n.path] = true
				if e := record(n, fmt.Errorf("%w: value-backed stateful slot", ErrUnsupported)); e != nil {
					return out, e
				}
				continue
			}
			if loader, ok := n.value.(layout.Loadable); ok {
				snap, ok := n.value.(interface{ PreserveState() func() })
				if !ok {
					failed[n.path] = true
					if e := record(n, fmt.Errorf("%w: disk validation requires state snapshot", ErrUnsupported)); e != nil {
						return out, e
					}
					continue
				}
				restores = append(restores, snap.PreserveState())
				var e error
				if e = guard(b.context.root, n.path, physicalKind(n)); e != nil {
					failed[n.path] = true
					if e = record(n, e); e != nil {
						return out, e
					}
					continue
				}
				if _, e = loader.Load(); e != nil {
					failed[n.path] = true
					if e = record(n, e); e != nil {
						return out, e
					}
				}
			}
		}
	}
	// Admit observable structure and required content before invoking domain
	// rules. A container rule may rely on every required child being available.
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		if failed[n.path] {
			continue
		}
		var e error
		if n.kind != "group" {
			e = guard(b.context.root, n.path, physicalKind(n))
		}
		if e == nil && n.meta.required {
			if p.Source == Disk {
				_, e = os.Lstat(n.path)
			} else if state, ok := n.value.(layout.ContentState); ok && !state.HasContent() {
				e = fmt.Errorf("required content is not loaded")
			} else if _, stateful := n.value.(layout.ContentState); !stateful && !(preparing && (n.kind == "dir" || n.kind == "collection" || n.meta.empty)) {
				_, e = os.Lstat(n.path)
			}
		}
		if e != nil {
			failed[n.path] = true
			if e = record(n, e); e != nil {
				return out, e
			}
		}
	}
	// Match Conduit's explicit validation boundaries. Custom ValidateDeep or
	// Validate owns its subtree's domain checks; management still admits every
	// declared node above. Slots remain traversable for independent diagnostics.
	err = walk(reflect.ValueOf(b.input), b.path, ".", b.meta, map[nodeIdentity]bool{}, func(n node) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if n.kind == "collection" {
			return nil
		}
		for path := range failed {
			if within(n.path, path) {
				return nil
			}
		}
		opts := layout.ValidateOptions{PathSafetyPolicy: b.context.io.PathSafetyPolicy}
		var e error
		if validator, ok := n.value.(layout.DeepValidator); ok {
			_, e = validator.ValidateDeep(opts)
		} else if validator, ok := n.value.(layout.Validator); ok {
			e = validator.Validate(opts)
		} else {
			return nil
		}
		if e != nil {
			if e = record(n, e); e != nil {
				return e
			}
		}
		return skipChildren
	})
	if err != nil {
		return out, err
	}

	return out, out.Err()
}

func (s *scope[L]) Scaffold(ctx context.Context, p ScaffoldPolicy) (Outcome, error) {
	var out Outcome
	b := s.binding
	if err := require(b, OpScaffold); err != nil {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	for _, n := range ns {
		if n.kind != "group" {
			if err = guard(b.context.root, n.path, physicalKind(n)); err != nil {
				out.add("scaffold", n.path, "failed", false, err)
				return out, err
			}
		}
	}
	ioPolicy := b.context.io
	ioPolicy.EnsurePolicy = layout.EnsureScaffold
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		_, stateful := n.value.(layout.ContentState)
		eligible := n.kind == "dir" || n.kind == "collection" || (n.kind == "file" && !stateful && (p.Files || n.meta.empty))
		if !eligible {
			continue
		}
		beforeInfo, before := os.Lstat(n.path)
		if n.kind == "collection" {
			err = layout.NewDir(n.path).Ensure(ioPolicy)
		} else if ensure, ok := n.value.(interface{ Ensure(layout.Context) error }); ok {
			err = ensure.Ensure(ioPolicy)
		} else {
			err = fmt.Errorf("%w: scaffold %s", ErrUnsupported, n.path)
		}
		afterInfo, after := os.Lstat(n.path)
		created := os.IsNotExist(before) && after == nil
		changedMode := before == nil && after == nil && beforeInfo.Mode() != afterInfo.Mode()
		applied := created || changedMode
		status := "preserved"
		if created {
			status = "created"
		} else if changedMode {
			status = "permissions_changed"
		}
		if err != nil {
			status = "failed"
		}
		out.add("scaffold", n.path, status, applied, err)
		if err != nil {
			out.Effects[len(out.Effects)-1].MayHaveApplied = true
		}
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

type pendingWrite struct {
	node    node
	write   layout.PreparedWrite
	missing bool
}

func prepareWrites(b Binding, p SavePolicy, selected func(node) bool) ([]pendingWrite, Outcome, error) {
	var writes []pendingWrite
	paths := map[string]bool{}
	var out Outcome
	ns, err := nodesOf(b)
	if err != nil {
		return nil, out, err
	}
	for _, n := range ns {
		if selected != nil && !selected(n) {
			continue
		}
		state, ok := n.value.(layout.ContentState)
		if !ok {
			continue
		}
		if !state.HasContent() {
			out.add("prepare", n.path, "no_content", false, nil)
			continue
		}
		if p.Eligibility == DirtyOnly && state.MemoryState() != layout.MemoryDirty {
			out.add("prepare", n.path, "unchanged", false, nil)
			continue
		}
		if !n.mutable {
			return nil, out, fmt.Errorf("%w: value-backed stateful slot %s", ErrUnsupported, n.path)
		}
		if err = guard(b.context.root, n.path, physicalKind(n)); err != nil {
			return nil, out, err
		}
		_, e := os.Lstat(n.path)
		missing := os.IsNotExist(e)
		if e != nil && !missing {
			return nil, out, e
		}
		if missing && p.Missing != CreateMissing {
			return nil, out, fmt.Errorf("save target missing: %s", n.path)
		}
		preparer, ok := n.value.(layout.WritePreparer)
		if !ok {
			return nil, out, fmt.Errorf("%w: cannot prepare %s", ErrUnsupported, n.path)
		}
		w, e := preparer.PrepareWrite()
		if e != nil {
			return nil, out, e
		}
		if paths[n.path] {
			return nil, out, fmt.Errorf("conflicting writes to %s", n.path)
		}
		paths[n.path] = true
		writes = append(writes, pendingWrite{n, w, missing})
	}
	return writes, out, nil
}
func applyWrites(ctx context.Context, b Binding, writes []pendingWrite, out Outcome, exclusive bool) (Outcome, error) {
	// Reject unsupported publication policies before any member is written.
	for _, p := range writes {
		if (exclusive || p.missing) && b.context.io.WritePolicy != layout.WriteDirect {
			err := fmt.Errorf("%w: exclusive creation requires WriteDirect: %s", ErrUnsupported, p.node.path)
			out.add("prepare", p.node.path, "failed", false, err)
			return out, err
		}
	}
	for _, p := range writes {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if err := guard(b.context.root, p.node.path, "file"); err != nil {
			out.add("write", p.node.path, "failed", false, err)
			return out, err
		}
		if !p.missing {
			if _, err := os.Lstat(p.node.path); err != nil {
				out.add("write", p.node.path, "failed", false, err)
				return out, err
			}
		}
		var err error
		if exclusive || p.missing {
			err = p.write.WriteExclusive(b.context.io)
		} else {
			err = p.write.Write(b.context.io)
		}
		applied := err == nil
		status := "written"
		if err != nil {
			status = "failed"
		}
		out.add("write", p.node.path, status, applied, err)
		if err != nil && !errors.Is(err, layout.ErrPreparedWriteStale) {
			out.Effects[len(out.Effects)-1].MayHaveApplied = true
		}
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
func (s *scope[L]) Save(ctx context.Context, p SavePolicy) (Outcome, error) {
	var out Outcome
	if err := require(s.binding, OpSave); err != nil {
		return out, err
	}
	if p.Eligibility > Rewrite || p.Missing > CreateMissing {
		return out, fmt.Errorf("invalid save policy")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if diagnostics, err := s.internal().Validate(ctx, ValidationPolicy{}); err != nil {
		for _, d := range diagnostics.Entries {
			out.add(d.Phase, d.Path, "failed", false, d.Err)
		}
		return out, err
	}
	writes, out, err := prepareWrites(s.binding, p, nil)
	if err != nil {
		return out, err
	}
	return applyWrites(ctx, s.binding, writes, out, false)
}
func (s *scope[L]) Refresh(ctx context.Context, p RefreshPolicy) (Outcome, error) {
	var out Outcome
	b := s.binding
	if err := require(b, OpRefresh); err != nil {
		return out, err
	}
	if p.Source > Disk || p.Effects > MemoryOnly || p.Dirty > ReplaceDirty || p.Missing > CreateMissing {
		return out, fmt.Errorf("invalid refresh policy")
	}
	if p.Source == Disk {
		var err error
		out, err = s.internal().Load(ctx, LoadPolicy{Dirty: p.Dirty, Traversal: Discovered})
		if err != nil {
			return out, err
		}
	}
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		if n.meta.role != "derived" {
			continue
		}
		if err = renderNode(n); err != nil {
			return out, err
		}
		out.add("render", n.path, "prepared", false, nil)
	}
	if _, err = s.internal().Validate(ctx, ValidationPolicy{}); err != nil {
		return out, err
	}
	if p.Effects == MemoryOnly {
		return out, nil
	}
	missing := p.Missing
	if missing == DefaultMissing {
		missing = CreateMissing
	}
	writes, prepared, err := prepareWrites(b, SavePolicy{Missing: missing}, func(n node) bool { return n.meta.role == "derived" })
	out.Effects = append(out.Effects, prepared.Effects...)
	if err != nil {
		return out, err
	}
	return applyWrites(ctx, b, writes, out, false)
}
func renderNode(n node) error {
	if r, ok := n.value.(layout.Renderable); ok {
		text, e := r.Render()
		if e != nil {
			return e
		}
		r.SetRendered(text)
		return nil
	}
	if r, ok := n.value.(layout.Templatable); ok {
		text, e := r.RenderTemplate(r.Template())
		if e != nil {
			return e
		}
		r.SetRendered(text)
		return nil
	}
	return fmt.Errorf("%w: no derivation contract for %s", ErrUnsupported, n.path)
}

func defaultNodes(b Binding) error {
	return layout.DefaultDeep(b.input)
}

func (s *scope[L]) internal() *scope[L] {
	b := s.binding
	b.options = ScopeOptions{}
	return &scope[L]{b, s.tree}
}
