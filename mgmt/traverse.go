package mgmt

import (
	"context"
	"errors"
	"fmt"
	"github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var skipChildren = errors.New("skip managed traversal children")

type children interface {
	CachedChildren() []layout.CachedChild
	ChildKind() string
	BindChild(string) (any, error)
}
type snapshotter interface{ SnapshotState() (func(), error) }
type node struct {
	value            any
	path, name, kind string
	meta             semantics
	mutable          bool
}

func parseSemantics(tag string) (semantics, error) {
	var m semantics
	for _, part := range strings.FieldsFunc(tag, func(r rune) bool { return r == ',' || r == ';' }) {
		switch strings.TrimSpace(part) {
		case "":
		case "required":
			m.required = true
		case "detect":
			m.detect = true
		case "empty":
			m.empty = true
		case "role=seeded":
			m.role = "seeded"
		case "role=derived":
			m.role = "derived"
		default:
			return m, fmt.Errorf("unknown manage annotation %q", part)
		}
	}
	return m, nil
}
func nodesOf(b Binding) ([]node, error) {
	var ns []node
	err := walk(reflect.ValueOf(b.input), b.path, ".", b.meta, map[nodeIdentity]bool{}, func(n node) error { ns = append(ns, n); return nil })
	return ns, err
}
func walk(v reflect.Value, base, name string, m semantics, seen map[nodeIdentity]bool, visit func(node) error) error {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	mutable := v.Kind() == reflect.Pointer || v.CanAddr()
	value := v.Interface()
	if v.Kind() != reflect.Pointer && v.CanAddr() {
		value = v.Addr().Interface()
	}
	if id := identity(value); id != (nodeIdentity{}) {
		if seen[id] {
			return nil
		}
		seen[id] = true
	}
	n := node{value: value, path: base, name: name, meta: m, kind: "group", mutable: mutable}
	if p, ok := value.(layout.Pather); ok {
		n.path = p.Path()
	}
	if _, ok := value.(children); ok {
		n.kind = "collection"
	} else if _, ok := value.(layout.ContentState); ok {
		n.kind = "file"
	} else {
		switch value.(type) {
		case *layout.Dir, layout.Dir:
			n.kind = "dir"
		case *layout.File, layout.File, *layout.Exec, layout.Exec:
			n.kind = "file"
		}
	}
	if err := visit(n); err != nil {
		if errors.Is(err, skipChildren) {
			return nil
		}
		return err
	}
	if c, ok := value.(children); ok {
		for _, child := range c.CachedChildren() {
			if err := walk(reflect.ValueOf(child.Value), child.Path, child.Name, semantics{}, seen, visit); err != nil {
				return err
			}
		}
		return nil
	}
	if n.kind != "group" {
		return nil
	}
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("unsupported layout value %T", value)
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("layout")
		if tag == "" && !f.Anonymous {
			continue
		}
		sm, err := parseSemantics(f.Tag.Get("manage"))
		if err != nil {
			return fmt.Errorf("%s.%s: %w", name, f.Name, err)
		}
		childBase := base
		if tag != "" {
			childBase = filepath.Join(base, tag)
		}
		if err = walk(v.Field(i), childBase, name+"."+f.Name, sm, seen, visit); err != nil {
			return err
		}
	}
	return nil
}
func discover(ctx context.Context, b Binding, record func(node, error) error) error {
	return walk(reflect.ValueOf(b.input), b.path, ".", b.meta, map[nodeIdentity]bool{}, func(n node) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		c, ok := n.value.(children)
		if !ok {
			return nil
		}
		if e := guard(b.context.root, n.path, "dir"); e != nil {
			return record(n, e)
		}
		entries, err := os.ReadDir(n.path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return record(n, err)
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			kind := c.ChildKind()
			valid := kind == "dir" && entry.IsDir() || kind == "file" && entry.Type().IsRegular() || kind == "link" && entry.Type()&os.ModeSymlink != 0
			if !valid {
				continue
			}
			if _, err = c.BindChild(entry.Name()); err != nil {
				if e := record(n, err); e != nil {
					return e
				}
			}
		}
		return nil
	})
}
func physicalKind(n node) string {
	if n.kind == "collection" || n.kind == "dir" {
		return "dir"
	}
	if n.kind == "file" {
		return "file"
	}
	return ""
}
