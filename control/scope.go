package control

import (
	"errors"
	"fmt"

	"github.com/qlustra/conduit/internal/composer"
	"github.com/qlustra/conduit/internal/controlbinding"
	"github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/internal/managed"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail"
)

type scopeState[L any] struct {
	layout   *managed.Layout[L]
	register func() error
}

// Layout returns the focused live Layout retained by this scope. A ProjectCtrl
// inherits this API. The returned type is also exposed as conduit.Layout[L].
func (s *ScopeCtrl[L]) Layout() *managed.Layout[L] { return s.state.layout }

// PrepareLayout supplies the typed binding factory used by Conduit's control
// interpreter. It reads declaration metadata without composing live storage.
func (s *ScopeCtrl[L]) PrepareLayout(charter *sail.Charter[layoutdecl.Layout]) (controlbinding.ScopePlan, error) {
	chart, err := sail.Populate[L](charter)
	if err != nil {
		return controlbinding.ScopePlan{}, err
	}
	if err := composer.ValidateChart(chart); err != nil {
		return controlbinding.ScopePlan{}, err
	}
	return controlbinding.ScopePlan{Chart: chart, Bind: func(tree any) (any, error) {
		value, ok := tree.(*L)
		if !ok {
			return nil, fmt.Errorf("scope: expected *%T, got %T", *new(L), tree)
		}
		return managed.Bind(charter, chart, value)
	}}, nil
}

// BindScope attaches the prepared live handler and its controller registration.
// This is the interpreter's binding protocol, not automatic synchronization.
func (s *ScopeCtrl[L]) BindScope(handler any, register func() error) error {
	live, ok := handler.(*managed.Layout[L])
	if !ok || live == nil || live.Tree() == nil {
		return fmt.Errorf("scope: a compatible composed Layout is required")
	}
	s.state = scopeState[L]{layout: live, register: register}
	return nil
}

func (s *ScopeCtrl[L]) RegisterControls() error {
	if s.state.layout == nil {
		return fmt.Errorf("scope: a bound control is required")
	}
	if s.state.register != nil {
		return s.state.register()
	}
	return nil
}

func (s *ScopeCtrl[L]) Ensure(ctx layout.Context) (layout.ResultCode, error) {
	return s.state.layout.Ensure(ctx)
}

// Load orchestrates layout loading and registration of discovered controls.
func (s *ScopeCtrl[L]) Load(ctx layout.Context) (layout.ResultCode, error) {
	code, err := s.state.layout.Load(ctx)
	// A failing traversal may still have composed members. Register the live
	// state produced by this operation while preserving its original failure.
	if registerErr := s.RegisterControls(); registerErr != nil {
		err = errors.Join(err, registerErr)
		code = layout.LoadFailed
	}
	return code, err
}

// Discover orchestrates layout discovery and registration of member controls.
func (s *ScopeCtrl[L]) Discover(ctx layout.Context) (layout.ResultCode, error) {
	code, err := s.state.layout.Discover(ctx)
	if registerErr := s.RegisterControls(); registerErr != nil {
		err = errors.Join(err, registerErr)
		code = layout.DiscoverFailed
	}
	return code, err
}

func (s *ScopeCtrl[L]) Scan(ctx layout.Context) (layout.ResultCode, error) {
	return s.state.layout.Scan(ctx)
}
func (s *ScopeCtrl[L]) Sync(ctx layout.Context) (layout.ResultCode, error) {
	return s.state.layout.Sync(ctx)
}
func (s *ScopeCtrl[L]) Validate(opts layout.ValidateOptions) (layout.ResultCode, error) {
	return s.state.layout.Validate(opts)
}
func (s *ScopeCtrl[L]) Default() error { return s.state.layout.Default() }
func (s *ScopeCtrl[L]) Render() error  { return s.state.layout.Render() }
func (s *ScopeCtrl[L]) Unload() error  { return s.state.layout.Unload() }
