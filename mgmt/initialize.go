package mgmt

import (
	"context"
	"fmt"
	"github.com/qlustra/conduit/layout"
	"os"
)

// Bootstrap completes supported missing content. Existing disk content is loaded
// and restored after customization, then used by the context callback. Callbacks
// may change memory but must not perform filesystem writes or rebind participants.
// Preparation failures may leave new/defaulted values in memory; disk application
// is not a transaction. Existing content bytes are never rewritten by Bootstrap.
func Bootstrap[L, I any](ctx context.Context, s Scope[L], input I, config Initialization[L, I], p BootstrapPolicy) (Outcome, error) {
	var out Outcome
	if s == nil {
		return out, fmt.Errorf("scope must not be nil")
	}
	b := s.Binding()
	if err := require(b, OpBootstrap); err != nil {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if p.Dirty > ReplaceDirty || p.Unsupported > SkipUnsupported {
		return out, fmt.Errorf("invalid bootstrap policy")
	}
	if p.NewRootOnly {
		if _, e := os.Lstat(b.path); !os.IsNotExist(e) {
			if e == nil {
				e = os.ErrExist
			}
			return out, e
		}
	}
	return initialize(ctx, s, input, config, p, nil)
}

func initialize[L, I any](ctx context.Context, s Scope[L], input I, config Initialization[L, I], p BootstrapPolicy, admit func(*Outcome) error) (Outcome, error) {
	var out Outcome
	b := s.Binding()
	var existing []func()
	known := map[nodeIdentity]bool{}
	present := map[nodeIdentity]bool{}
	restoreExisting := func() {
		for _, f := range existing {
			f()
		}
		existing = nil
	}
	defer restoreExisting()
	if err := discover(ctx, b, func(n node, e error) error { return e }); err != nil {
		return out, err
	}
	// Preflight dirty state for all known participants before authoritative loads.
	ns, err := nodesOf(b)
	if err != nil {
		return out, err
	}
	for _, n := range ns {
		if state, ok := n.value.(layout.ContentState); ok && state.MemoryState() == layout.MemoryDirty && p.Dirty == RejectDirty {
			// Prepared values on genuinely missing files are legitimate initialization input.
			if _, e := os.Lstat(n.path); e == nil {
				return out, fmt.Errorf("%w: %s", ErrDirty, n.path)
			} else if !os.IsNotExist(e) {
				return out, e
			}
		}
	}
	observe := func() error {
		ns, e := nodesOf(b)
		if e != nil {
			return e
		}
		for _, n := range ns {
			if e = ctx.Err(); e != nil {
				return e
			}
			id := identity(n.value)
			if known[id] {
				continue
			}
			known[id] = true
			if n.kind != "group" {
				if e = guard(b.context.root, n.path, physicalKind(n)); e != nil {
					return e
				}
			}
			_, e = os.Lstat(n.path)
			exists := e == nil
			if e != nil && !os.IsNotExist(e) {
				return e
			}
			if exists {
				present[id] = true
			}
			if loader, ok := n.value.(layout.Loadable); ok && exists {
				if !n.mutable {
					return fmt.Errorf("%w: value-backed content %s", ErrUnsupported, n.path)
				}
				snap, ok := n.value.(snapshotter)
				if !ok {
					return fmt.Errorf("%w: preservation requires snapshot for %s", ErrUnsupported, n.path)
				}
				if _, e = loader.Load(); e != nil {
					return fmt.Errorf("load %s: %w", n.path, e)
				}
				restore, e := snap.SnapshotState()
				if e != nil {
					return e
				}
				existing = append(existing, restore)
				out.add("observe", n.path, "preserved", false, nil)
			}
		}
		return nil
	}
	if err = observe(); err != nil {
		return out, err
	}
	if err = defaultNodes(b); err != nil {
		return out, err
	}
	if config.Initialize != nil {
		if err = config.Initialize(ctx, s.Layout(), input); err != nil {
			return out, err
		}
	}
	// Include explicitly introduced children, restoring any existing disk authority.
	if err = observe(); err != nil {
		return out, err
	}
	restoreExisting()
	if err = checkBinding(b); err != nil {
		return out, err
	}
	if config.Context != nil {
		if err = config.Context(ctx, s.Layout()); err != nil {
			return out, err
		}
	}
	if err = checkBinding(b); err != nil {
		return out, err
	}
	ns, err = nodesOf(b)
	if err != nil {
		return out, err
	}
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		if present[identity(n.value)] {
			continue
		}
		_, renderable := n.value.(layout.Renderable)
		_, templatable := n.value.(layout.Templatable)
		if renderable || templatable {
			if err = renderNode(n); err != nil {
				return out, err
			}
		}
		if n.kind == "file" {
			state, ok := n.value.(layout.ContentState)
			if (!ok && !n.meta.empty) || (ok && !state.HasContent()) {
				if p.Unsupported == ErrorUnsupported {
					return out, fmt.Errorf("%w: no initialization content for %s", ErrUnsupported, n.path)
				}
				out.add("prepare", n.path, "unsupported", false, nil)
			}
		}
	}
	if config.Validate != nil {
		if err = config.Validate(ctx, s.Layout()); err != nil {
			return out, err
		}
	}
	internal := &scope[L]{binding: b, tree: s.Layout()}
	internal = internal.internal()
	if _, err = internal.validate(ctx, ValidationPolicy{}, true); err != nil {
		return out, err
	}
	writes, prepared, err := prepareWrites(b, SavePolicy{Eligibility: Rewrite, Missing: CreateMissing}, func(n node) bool { return !present[identity(n.value)] })
	out.Effects = append(out.Effects, prepared.Effects...)
	if err != nil {
		return out, err
	}
	if len(writes) > 0 && b.context.io.WritePolicy != layout.WriteDirect {
		return out, fmt.Errorf("%w: exclusive initialization requires WriteDirect", ErrUnsupported)
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if admit != nil {
		if err = admit(&out); err != nil {
			return out, err
		}
	}
	structural, err := internal.Scaffold(ctx, ScaffoldPolicy{})
	out.Effects = append(out.Effects, structural.Effects...)
	if err != nil {
		return out, err
	}
	out, err = applyWrites(ctx, b, writes, out, true)
	if err != nil {
		return out, err
	}
	if config.AfterApply != nil {
		if err = config.AfterApply(ctx, out); err != nil {
			return out, &PostApplyError{err}
		}
	}
	return out, nil
}
