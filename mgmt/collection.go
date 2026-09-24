package mgmt

import (
	"context"
	"fmt"
	"github.com/qlustra/conduit/layout"
	"os"
)

type collection[L any] struct {
	binding Binding
	slot    *layout.Slot[*L]
	options CollectionOptions
}

func BindCollection[L any](owner Bound, slot *layout.Slot[*L], options CollectionOptions) (Collection[L], error) {
	b, err := childBinding(owner, slot, ScopeOptions{})
	if err != nil {
		return nil, err
	}
	return &collection[L]{b, slot, options}, nil
}
func (c *collection[L]) Binding() Binding           { return c.binding }
func (c *collection[L]) Capabilities() Capabilities { return Capabilities{OpAt, OpList, OpCreate} }
func (c *collection[L]) At(key string, p BindPolicy) (Scope[L], error) {
	child, err := c.slot.At(key)
	if err != nil {
		return nil, err
	}
	bound, err := BindScope(c, child, c.options.Scope)
	if err != nil {
		return nil, err
	}
	if p.RequireExisting {
		if err = guard(c.binding.context.root, bound.Binding().path, "dir"); err != nil {
			return nil, err
		}
		if _, err = os.Stat(bound.Binding().path); err != nil {
			return nil, err
		}
	}
	return bound, nil
}
func (c *collection[L]) List(ctx context.Context, p ListPolicy) (Inventory, error) {
	var out Inventory
	if p.Diagnostics > CollectAll {
		return out, fmt.Errorf("invalid list policy")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	b := c.binding
	if err := guard(b.context.root, b.path, "dir"); err != nil {
		return out, err
	}
	entries, err := os.ReadDir(b.path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		item := Item{Key: entry.Name(), Path: layout.NewDir(b.path).Join(entry.Name()), Kind: "dir", Present: true}
		if !entry.IsDir() {
			e := fmt.Errorf("not a directory collection member")
			out.Diagnostics.add("list", item.Path, e)
			if p.Diagnostics == FailFast {
				return out, e
			}
			continue
		}
		out.Entries = append(out.Entries, item)
		if p.Validate {
			child, e := c.At(entry.Name(), BindPolicy{RequireExisting: true})
			if e == nil {
				var d Diagnostics
				internal := &scope[L]{binding: child.Binding(), tree: child.Layout()}
				d, e = internal.internal().Validate(ctx, ValidationPolicy{Source: Disk, Diagnostics: p.Diagnostics})
				if e != nil && !d.HasErrors() {
					d.add("list", item.Path, e)
				}
				out.Diagnostics.Entries = append(out.Diagnostics.Entries, d.Entries...)
			} else {
				out.Diagnostics.add("list", item.Path, e)
			}
			if e != nil && p.Diagnostics == FailFast {
				return out, e
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return out, out.Diagnostics.Err()
}

// Create validates a prepared child before reserving its destination. Successful
// creation and post-apply failure both retain Value and the applied Outcome.
func Create[L, I any](ctx context.Context, target Collection[L], key string, input I, config Creation[L, I], p CreatePolicy) (Created[L], error) {
	var result Created[L]
	c, ok := target.(*collection[L])
	if !ok {
		return result, fmt.Errorf("%w: collection implementation", ErrUnsupported)
	}
	if p.Destination > AbsentOrEmpty || p.Parent > CreateMissing || p.Unsupported > SkipUnsupported {
		return result, fmt.Errorf("invalid create policy")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	child, err := c.At(key, BindPolicy{})
	if err != nil {
		return result, err
	}
	result.Value = child
	path := child.Binding().path
	b := c.binding
	checkDestination := func() error {
		if e := guard(b.context.root, path, "dir"); e != nil {
			return e
		}
		_, e := os.Lstat(path)
		if os.IsNotExist(e) {
			return nil
		}
		if e != nil {
			return e
		}
		if p.Destination == AbsentOnly {
			return fmt.Errorf("destination exists: %w", os.ErrExist)
		}
		entries, e := os.ReadDir(path)
		if e != nil {
			return e
		}
		if len(entries) > 0 {
			return fmt.Errorf("destination is not empty: %w", os.ErrExist)
		}
		return nil
	}
	if err = checkDestination(); err != nil {
		return result, err
	}
	if err = guard(b.context.root, b.path, "dir"); err != nil {
		return result, err
	}
	if p.Parent == FailIfMissing {
		if _, err = os.Stat(b.path); err != nil {
			return result, err
		}
	}
	admit := func(out *Outcome) error {
		if p.Parent == FailIfMissing {
			if _, e := os.Stat(b.path); e != nil {
				return e
			}
		} else {
			_, before := os.Stat(b.path)
			ioPolicy := b.context.io
			ioPolicy.EnsurePolicy = layout.EnsureScaffold
			if e := layout.NewDir(b.path).Ensure(ioPolicy); e != nil {
				out.add("create", b.path, "failed", false, e)
				out.Effects[len(out.Effects)-1].MayHaveApplied = true
				return e
			}
			if os.IsNotExist(before) {
				out.add("create", b.path, "created", true, nil)
			}
		}
		if e := checkDestination(); e != nil {
			return e
		}
		e := os.Mkdir(path, b.context.io.DirMode)
		if os.IsExist(e) && p.Destination == AbsentOrEmpty {
			return checkDestination()
		}
		if e != nil {
			return e
		}
		out.add("create", path, "created", true, nil)
		return nil
	}
	// Creation can initialize only the selected member, even if a caller has
	// manually inserted a rebound child into the slot cache.
	preparation := &scope[L]{binding: child.Binding(), tree: child.Layout()}
	preparation.binding.context = &bindingContext{root: path, io: b.context.io, origin: child.Layout()}
	if err = checkBinding(preparation.binding); err != nil {
		return result, err
	}
	result.Outcome, err = initialize(ctx, preparation, input, Initialization[L, I](config), BootstrapPolicy{Dirty: ReplaceDirty, Unsupported: p.Unsupported}, admit)
	return result, err
}
