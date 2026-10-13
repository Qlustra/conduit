package control

import (
	"fmt"
	"os"

	"github.com/qlustra/conduit/internal/controlbinding"
	"github.com/qlustra/conduit/internal/managed"
	"github.com/qlustra/conduit/layout"
)

func (c *CollectionCtrl[M, S]) walk(apply func(controlbinding.Operations) error) error {
	if c.state == nil {
		return fmt.Errorf("collection: a bound control is required")
	}
	for name, member := range c.All() {
		ops, err := memberOperations(member)
		if err == nil {
			err = apply(ops)
		}
		if err != nil {
			return fmt.Errorf("member %q: %w", name, err)
		}
	}
	return nil
}

func (c *CollectionCtrl[M, S]) record(ctx layout.Context, op layout.Operation, code layout.ResultCode, err error) {
	if ctx.Reporter != nil && c.state != nil {
		ctx.Reporter.Record(layout.Entry{Op: op, Path: c.Path(), Result: code, Err: err})
	}
}

// Ensure prepares the source directory and registered members. Like Layout's
// Ensure, this neither discovers members nor loads or persists their content.
func (c *CollectionCtrl[M, S]) Ensure(ctx layout.Context) (layout.ResultCode, error) {
	code := layout.EnsureEnsured
	if c.state == nil {
		return layout.EnsureFailed, fmt.Errorf("collection: a bound control is required")
	}
	var err error
	if ctx.EnsurePolicy.Has(layout.EnsureDirs) {
		err = layout.NewDir(c.Path()).Ensure(ctx)
		if err == nil && c.state.api.ChildKind() == "dir" {
			for _, name := range c.Keys() {
				if err = layout.NewDir(c.Path()).Dir(name).Ensure(ctx); err != nil {
					break
				}
			}
		}
	}
	if err == nil {
		err = c.walk(func(ops controlbinding.Operations) error { _, err := ops.Ensure(ctx); return err })
	}
	if err != nil {
		code = layout.EnsureFailed
	}
	c.record(ctx, layout.OpEnsure, code, err)
	return code, err
}

// readMembers discovers physical members through the controller, keeping source
// composition and controller registration together. Missing cached members are
// left alone; it does not compare or reconcile independently changed state.
func (c *CollectionCtrl[M, S]) readMembers(apply func(controlbinding.Operations) error) error {
	if c.state == nil {
		return fmt.Errorf("collection: a bound control is required")
	}
	kind := c.state.api.ChildKind()
	switch kind {
	case "dir", "file", "link":
	default:
		return fmt.Errorf("collection: unsupported child kind %q", kind)
	}
	entries, err := os.ReadDir(c.Path())
	if err != nil {
		return err
	}
	for name, err := range managed.PhysicalMemberNames(c.Path(), kind, entries) {
		if err != nil {
			return err
		}
		member, err := c.ComposeMember(name)
		if err != nil {
			return err
		}
		ops, err := memberOperations(member)
		if err == nil {
			err = apply(ops)
		}
		if err != nil {
			return fmt.Errorf("member %q: %w", name, err)
		}
	}
	return nil
}

// Discover constructs member controls and refreshes their filesystem state.
func (c *CollectionCtrl[M, S]) Discover(ctx layout.Context) (layout.ResultCode, error) {
	err := c.readMembers(func(ops controlbinding.Operations) error { _, err := ops.Discover(ctx); return err })
	code := layout.DiscoverTraversed
	if os.IsNotExist(err) {
		code, err = layout.DiscoverMissing, nil
	} else if err != nil {
		code = layout.DiscoverFailed
	}
	c.record(ctx, layout.OpDiscover, code, err)
	return code, err
}

// Load discovers member controls and loads their declared content.
func (c *CollectionCtrl[M, S]) Load(ctx layout.Context) (layout.ResultCode, error) {
	err := c.readMembers(func(ops controlbinding.Operations) error { _, err := ops.Load(ctx); return err })
	code := layout.LoadTraversed
	if os.IsNotExist(err) {
		code, err = layout.LoadMissing, nil
	} else if err != nil {
		code = layout.LoadFailed
	}
	c.record(ctx, layout.OpLoad, code, err)
	return code, err
}

func (c *CollectionCtrl[M, S]) Scan(ctx layout.Context) (layout.ResultCode, error) {
	err := c.walk(func(ops controlbinding.Operations) error { _, err := ops.Scan(ctx); return err })
	code := layout.ScanTraversed
	if err != nil {
		code = layout.ScanFailed
	}
	c.record(ctx, layout.OpScan, code, err)
	return code, err
}
func (c *CollectionCtrl[M, S]) Sync(ctx layout.Context) (layout.ResultCode, error) {
	err := c.walk(func(ops controlbinding.Operations) error { _, err := ops.Sync(ctx); return err })
	code := layout.SyncTraversed
	if err != nil {
		code = layout.SyncFailed
	}
	c.record(ctx, layout.OpSync, code, err)
	return code, err
}
func (c *CollectionCtrl[M, S]) Validate(opts layout.ValidateOptions) (layout.ResultCode, error) {
	err := c.walk(func(ops controlbinding.Operations) error { _, err := ops.Validate(opts); return err })
	code := layout.ValidateTraversed
	if err != nil {
		code = layout.ValidateFailed
	}
	return code, err
}
func (c *CollectionCtrl[M, S]) Default() error {
	return c.walk(func(ops controlbinding.Operations) error { return ops.Default() })
}
func (c *CollectionCtrl[M, S]) Render() error {
	return c.walk(func(ops controlbinding.Operations) error { return ops.Render() })
}
func (c *CollectionCtrl[M, S]) Unload() error {
	return c.walk(func(ops controlbinding.Operations) error { return ops.Unload() })
}
