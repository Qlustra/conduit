package mgmt

import (
	"context"
	"fmt"
	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
)

type DetectionPolicy struct{ All bool }
type Detection struct {
	Root        string
	Match       bool
	Boundary    bool
	Signals     []Item
	Diagnostics Diagnostics
}
type project[L any] struct {
	Space[L]
	signals []string
}

func BindProject[L any](root layout.Dir, tree *L, options ProjectOptions) (Project[L], error) {
	s, err := BindSpace(root, tree, options.Space)
	if err != nil {
		return nil, err
	}
	p := &project[L]{s, options.Signals}
	signals, err := p.signalNodes()
	if err != nil {
		return nil, err
	}
	if len(signals) == 0 {
		return nil, fmt.Errorf("project has no detection signals")
	}
	return p, nil
}
func (p *project[L]) Capabilities() Capabilities {
	return append(p.Space.Capabilities(), OpDetect)
}
func (p *project[L]) signalNodes() ([]node, error) {
	ns, err := nodesOf(p.Binding())
	if err != nil {
		return nil, err
	}
	var out []node
	if len(p.signals) == 0 {
		for _, n := range ns {
			if n.meta.detect {
				out = append(out, n)
			}
		}
		return out, nil
	}
	for _, signal := range p.signals {
		found := false
		for _, n := range ns {
			relative, e := filepath.Rel(p.Binding().context.root, n.path)
			if e == nil && relative == signal && n.kind != "group" {
				out = append(out, n)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown project signal %q", signal)
		}
	}
	return out, nil
}
func (p *project[L]) Detect(ctx context.Context, policy DetectionPolicy) (Detection, error) {
	out := Detection{Root: p.Root().Path()}
	ns, err := p.signalNodes()
	if err != nil {
		return out, err
	}
	matched := 0
	for _, n := range ns {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		info, e := os.Lstat(n.path)
		item := Item{Path: n.path, Kind: physicalKind(n), Present: e == nil}
		if e == nil {
			out.Boundary = true
			if e = guard(p.Binding().context.root, n.path, physicalKind(n)); e != nil {
				out.Diagnostics.add("detect", n.path, e)
			} else if info != nil {
				matched++
			}
		} else if !os.IsNotExist(e) {
			out.Boundary = true
			out.Diagnostics.add("detect", n.path, e)
		}
		out.Signals = append(out.Signals, item)
	}
	out.Match = matched > 0
	if policy.All {
		out.Match = matched == len(ns) && len(ns) > 0
	}
	return out, out.Diagnostics.Err()
}
func OpenProject[L any](ctx context.Context, root string, options ProjectOptions, policy DetectionPolicy) (Project[L], error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	tree := new(L)
	if err = conduit.Compose(absolute, tree); err != nil {
		return nil, err
	}
	p, err := BindProject(layout.NewDir(absolute), tree, options)
	if err != nil {
		return nil, err
	}
	d, err := p.Detect(ctx, policy)
	if err != nil {
		return nil, err
	}
	if !d.Match {
		return nil, fmt.Errorf("root is not a matching project: %s", absolute)
	}
	return p, nil
}

// LocateProject stops at the nearest candidate boundary, including a wrong-kind
// or inaccessible marker. It never silently falls back past a broken project.
func LocateProject[L any](ctx context.Context, start string, options ProjectOptions, policy DetectionPolicy) (Project[L], Detection, error) {
	path, err := filepath.Abs(start)
	if err != nil {
		return nil, Detection{}, err
	}
	for {
		if err = ctx.Err(); err != nil {
			return nil, Detection{}, err
		}
		tree := new(L)
		if err = conduit.Compose(path, tree); err != nil {
			return nil, Detection{}, err
		}
		p, e := BindProject(layout.NewDir(path), tree, options)
		if e != nil {
			return nil, Detection{}, e
		}
		d, e := p.Detect(ctx, policy)
		if d.Boundary || e != nil {
			if e == nil && !d.Match {
				e = fmt.Errorf("broken project boundary at %s", path)
			}
			return p, d, e
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil, d, os.ErrNotExist
		}
		path = parent
	}
}
