package mgmt

import (
	"fmt"
	"github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type semantics struct {
	role                    string
	required, detect, empty bool
}
type bindingContext struct {
	root   string
	io     layout.Context
	origin any
}

// Binding is an immutable operational binding to existing layout state.
type Binding struct {
	context *bindingContext
	input   any
	path    string
	meta    semantics
	options ScopeOptions
}

func (b Binding) Root() layout.Dir {
	if b.context == nil {
		return layout.Dir{}
	}
	return layout.NewDir(b.context.root)
}
func (b Binding) Path() string { return b.path }

type scope[L any] struct {
	binding Binding
	tree    *L
}

func (s *scope[L]) Binding() Binding           { return s.binding }
func (s *scope[L]) Layout() *L                 { return s.tree }
func (s *scope[L]) Capabilities() Capabilities { return capabilities(s.binding) }

type space[L any] struct{ scope *scope[L] }

func (s *space[L]) Binding() Binding           { return s.scope.Binding() }
func (s *space[L]) Root() layout.Dir           { return s.Binding().Root() }
func (s *space[L]) Layout() *L                 { return s.scope.tree }
func (s *space[L]) Scope() Scope[L]            { return s.scope }
func (s *space[L]) Capabilities() Capabilities { return s.scope.Capabilities() }

func BindSpace[L any](root layout.Dir, tree *L, options SpaceOptions) (Space[L], error) {
	if tree == nil {
		return nil, fmt.Errorf("layout must not be nil")
	}
	path, err := filepath.Abs(root.Path())
	if err != nil {
		return nil, err
	}
	b := Binding{context: &bindingContext{path, ioDefaults(options.Context), tree}, input: tree, path: path, options: options.Scope}
	if err := checkBinding(b); err != nil {
		return nil, err
	}
	return &space[L]{&scope[L]{b, tree}}, nil
}
func BindScope[L any](owner Bound, tree *L, options ScopeOptions) (Scope[L], error) {
	b, err := childBinding(owner, tree, options)
	if err != nil {
		return nil, err
	}
	return &scope[L]{b, tree}, nil
}
func BindSubspace[L any](parent Bound, root layout.Dir, tree *L, options SpaceOptions) (Space[L], error) {
	b, err := childBinding(parent, tree, options.Scope)
	if err != nil {
		return nil, err
	}
	path, err := filepath.Abs(root.Path())
	if err != nil {
		return nil, err
	}
	if !within(b.context.root, path) {
		return nil, fmt.Errorf("subspace %s escapes %s", path, b.context.root)
	}
	options.Context = inheritContext(b.context.io, options.Context)
	b.context = &bindingContext{path, ioDefaults(options.Context), tree}
	b.path = path
	if err = checkBinding(b); err != nil {
		return nil, err
	}
	return &space[L]{&scope[L]{b, tree}}, nil
}
func inheritContext(parent, child layout.Context) layout.Context {
	if reflect.ValueOf(child).IsZero() {
		return parent
	}
	return child
}
func childBinding(owner Bound, tree any, options ScopeOptions) (Binding, error) {
	if owner == nil || owner.Binding().context == nil {
		return Binding{}, fmt.Errorf("owner is not bound")
	}
	v := reflect.ValueOf(tree)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() {
		return Binding{}, fmt.Errorf("input must be a non-nil pointer")
	}
	b := owner.Binding()
	b.input = tree
	b.meta = semantics{}
	b.options = options
	found := false
	nodes, err := nodesOf(owner.Binding())
	if err != nil {
		return Binding{}, err
	}
	for _, n := range nodes {
		if identity(n.value) == identity(tree) {
			b.path = n.path
			b.meta = n.meta
			found = true
			break
		}
	}
	if !found { // An explicit overlay may be independently composed under this root.
		if p, ok := tree.(layout.Pather); ok {
			b.path = p.Path()
		} else {
			return Binding{}, fmt.Errorf("input is not a participant of the owner; bind an explicit Space for an independent layout")
		}
	}
	if err = checkBinding(b); err != nil {
		return Binding{}, err
	}
	return b, nil
}
func checkBinding(b Binding) error {
	nodes, err := nodesOf(b)
	if err != nil {
		return err
	}
	physical := map[string]string{}
	for _, n := range nodes {
		if n.meta.role != "" && n.kind != "file" {
			return fmt.Errorf("role annotation requires a content participant: %s", n.path)
		}
		if n.meta.empty && n.kind != "file" {
			return fmt.Errorf("empty annotation requires a file: %s", n.path)
		}
		if n.meta.detect && n.kind == "group" {
			return fmt.Errorf("detection requires a physical node: %s", n.path)
		}
		if n.path == "" || n.path == "." {
			return fmt.Errorf("uncomposed participant %s", n.name)
		}
		if !within(b.context.root, n.path) {
			return fmt.Errorf("participant %s escapes root %s", n.path, b.context.root)
		}
	}
	// Files cannot also serve as declared directories or as ancestors of
	// another participant, even when none of those paths exists yet.
	for _, n := range nodes {
		kind := physicalKind(n)
		if kind == "" {
			continue
		}
		path := filepath.Clean(n.path)
		if prior, ok := physical[path]; ok && prior != kind {
			return fmt.Errorf("conflicting declared kinds at %s: %s and %s", path, prior, kind)
		}
		physical[path] = kind
	}
	for path := range physical {
		for parent := filepath.Dir(path); parent != path; parent = filepath.Dir(parent) {
			if physical[parent] == "file" {
				return fmt.Errorf("declared file %s is an ancestor of %s", parent, path)
			}
			if parent == filepath.Dir(parent) {
				break
			}
		}
	}
	if len(nodes) == 0 {
		return fmt.Errorf("layout has no supported participants")
	}
	for _, op := range b.options.Ops {
		if !intrinsic(b).Has(op) {
			return fmt.Errorf("%w: %s", ErrUnsupported, op)
		}
	}
	return nil
}
func within(root, path string) bool {
	r, e := filepath.Rel(root, path)
	return e == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

type nodeIdentity struct {
	typ reflect.Type
	ptr uintptr
}

func identity(value any) nodeIdentity {
	v := reflect.ValueOf(value)
	if v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil() {
		return nodeIdentity{v.Type(), v.Pointer()}
	}
	return nodeIdentity{}
}

func intrinsic(b Binding) Capabilities {
	c := Capabilities{OpInspect, OpLoad, OpValidate, OpScaffold, OpSave, OpBootstrap}
	ns, err := nodesOf(b)
	if err == nil {
		for _, n := range ns {
			if n.meta.role == "derived" {
				c = append(c, OpRefresh)
				break
			}
		}
	}
	return c
}
func capabilities(b Binding) Capabilities {
	c := intrinsic(b)
	if b.options.Ops == nil {
		return c
	}
	var out Capabilities
	for _, op := range c {
		for _, allowed := range b.options.Ops {
			if op == allowed {
				out = append(out, op)
				break
			}
		}
	}
	return out
}
func require(b Binding, op Operation) error {
	if !capabilities(b).Has(op) {
		return fmt.Errorf("%w: %s", ErrUnsupported, op)
	}
	return checkBinding(b)
}

// guard checks currently observable parents and leaf kind. It is not a pinned
// descriptor protocol and does not promise resistance to concurrent path swaps.
func guard(root, path, kind string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !within(root, absolute) {
		return fmt.Errorf("path escapes management root")
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if e == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink path is unsupported: %s", p)
			}
			if p != absolute && !info.IsDir() {
				return fmt.Errorf("parent is not a directory: %s", p)
			}
			if p == absolute && kind != "" && ((kind == "dir" && !info.IsDir()) || (kind == "file" && !info.Mode().IsRegular())) {
				return fmt.Errorf("wrong filesystem kind at %s: expected %s", p, kind)
			}
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
	}
	return nil
}
