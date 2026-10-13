package control

import (
	"fmt"
	"iter"
	"reflect"
	"sort"
	"sync"

	"github.com/qlustra/conduit/internal/controlbinding"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

type collectionSource interface {
	Path() string
	ChildKind() string
	Keys() []string
	Ensure(layout.Context) error
	Remove(string)
	Delete(string, layout.Context) error
	Clear()
}

type collectionState[M, S any] struct {
	source                *S
	api                   collectionSource
	compose, require, get reflect.Value
	makeControl           func(any) (*M, error)
	mu                    sync.RWMutex
	members               map[string]*M
}

// memberSubject lets Sail interpret an authored member surface as a scope,
// including its document and nested collection declarations.
type memberSubject[M any] struct {
	Member M `control:"scope"`
}

func (c *CollectionCtrl[M, S]) MemberSubjects() subject.Children {
	return subject.Iterate[string, *memberSubject[M]](func(func(string, *memberSubject[M]) bool) {})
}
func (c *CollectionCtrl[M, S]) NewMemberSubject() any       { return new(memberSubject[M]) }
func (c *CollectionCtrl[M, S]) MemberControl(value any) any { return &value.(*memberSubject[M]).Member }

func (c *CollectionCtrl[M, S]) BindCollection(source any, makeControl func(any) (any, error)) error {
	live, ok := source.(*S)
	if !ok || live == nil || makeControl == nil {
		return fmt.Errorf("collection: a compatible source and member factory are required")
	}
	api, ok := any(live).(collectionSource)
	if !ok {
		return fmt.Errorf("collection: %T does not provide collection operations", live)
	}
	value := reflect.ValueOf(live)
	compose, require, get := value.MethodByName("ComposeMember"), value.MethodByName("Require"), value.MethodByName("Get")
	errorType := reflect.TypeFor[error]()
	for name, method := range map[string]reflect.Value{"ComposeMember": compose, "Require": require, "Get": get} {
		if !method.IsValid() || method.Type().NumIn() != 1 || method.Type().In(0) != reflect.TypeFor[string]() || method.Type().NumOut() != 2 {
			return fmt.Errorf("collection: %T must provide %s(string) with two results", live, name)
		}
		if method.Type().Out(0).Kind() != reflect.Pointer {
			return fmt.Errorf("collection: control surfaces require pointer members, got %v", method.Type().Out(0))
		}
		if name == "Get" {
			if method.Type().Out(1) != reflect.TypeFor[bool]() {
				return fmt.Errorf("collection: Get must return membership")
			}
		} else if method.Type().Out(1) != errorType {
			return fmt.Errorf("collection: %s must return error", name)
		}
	}
	c.state = &collectionState[M, S]{source: live, api: api, compose: compose, require: require, get: get,
		members: make(map[string]*M), makeControl: func(member any) (*M, error) {
			control, err := makeControl(member)
			if err != nil {
				return nil, err
			}
			typed, ok := control.(*M)
			if !ok || typed == nil {
				return nil, fmt.Errorf("collection: incompatible member control %T", control)
			}
			return typed, nil
		}}
	return nil
}

// Source exposes the bound source through its controller. Direct source
// mutation carries no controller synchronization guarantee.
func (c *CollectionCtrl[M, S]) Source() *S {
	if c.state == nil {
		return nil
	}
	return c.state.source
}

func (c *CollectionCtrl[M, S]) Path() string {
	if c.state == nil {
		return ""
	}
	return c.state.api.Path()
}

// ComposeMember constructs and registers a member control in memory. It does
// not create the member on disk. Repeated calls return the registered control.
func (c *CollectionCtrl[M, S]) ComposeMember(name string) (*M, error) {
	if c.state == nil {
		return nil, fmt.Errorf("collection: a bound control is required")
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if member, ok := c.state.members[name]; ok {
		return member, nil
	}
	results := c.state.compose.Call([]reflect.Value{reflect.ValueOf(name)})
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}
	return c.register(name, results[0].Interface())
}

// Require returns a registered control only when the source member exists on
// disk, preserving the source's Require semantics.
func (c *CollectionCtrl[M, S]) Require(name string) (*M, error) {
	if c.state == nil {
		return nil, fmt.Errorf("collection: a bound control is required")
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	results := c.state.require.Call([]reflect.Value{reflect.ValueOf(name)})
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}
	if member, ok := c.state.members[name]; ok {
		return member, nil
	}
	return c.register(name, results[0].Interface())
}

// register requires the registry lock. Source membership is changed only by
// the calling controller operation; this never checks for external drift.
func (c *CollectionCtrl[M, S]) register(name string, member any) (*M, error) {
	control, err := c.state.makeControl(member)
	if err != nil {
		return nil, fmt.Errorf("member %q: %w", name, err)
	}
	c.state.members[name] = control
	return control, nil
}

// RegisterControls is called during binding and controller-orchestrated
// discovery/loading. Ordinary registry access does not inspect the source.
func (c *CollectionCtrl[M, S]) RegisterControls() error {
	if c.state == nil {
		return fmt.Errorf("collection: a bound control is required")
	}
	c.state.mu.Lock()
	for _, name := range c.state.api.Keys() {
		if _, ok := c.state.members[name]; ok {
			continue
		}
		results := c.state.get.Call([]reflect.Value{reflect.ValueOf(name)})
		if !results[1].Bool() {
			continue
		}
		if _, err := c.register(name, results[0].Interface()); err != nil {
			c.state.mu.Unlock()
			return err
		}
	}
	c.state.mu.Unlock()
	for _, member := range c.All() {
		if scoped, ok := any(member).(interface{ RegisterControls() error }); ok {
			if err := scoped.RegisterControls(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Get looks up registered controllers without composing or discovering members.
func (c *CollectionCtrl[M, S]) Get(name string) (*M, bool) {
	if c.state == nil {
		return nil, false
	}
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	member, ok := c.state.members[name]
	return member, ok
}
func (c *CollectionCtrl[M, S]) Len() int {
	if c.state == nil {
		return 0
	}
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	return len(c.state.members)
}
func (c *CollectionCtrl[M, S]) Keys() []string {
	if c.state == nil {
		return nil
	}
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	keys := make([]string, 0, len(c.state.members))
	for key := range c.state.members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func (c *CollectionCtrl[M, S]) All() iter.Seq2[string, *M] {
	return func(yield func(string, *M) bool) {
		if c.state == nil {
			return
		}
		c.state.mu.RLock()
		items := make(map[string]*M, len(c.state.members))
		keys := make([]string, 0, len(c.state.members))
		for key, item := range c.state.members {
			keys = append(keys, key)
			items[key] = item
		}
		c.state.mu.RUnlock()
		sort.Strings(keys)
		for _, key := range keys {
			if !yield(key, items[key]) {
				return
			}
		}
	}
}

func (c *CollectionCtrl[M, S]) Remove(name string) {
	if c.state == nil {
		return
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.api.Remove(name)
	delete(c.state.members, name)
}
func (c *CollectionCtrl[M, S]) Clear() {
	if c.state == nil {
		return
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.api.Clear()
	c.state.members = make(map[string]*M)
}
func (c *CollectionCtrl[M, S]) Delete(name string, ctx layout.Context) error {
	if c.state == nil {
		return fmt.Errorf("collection: a bound control is required")
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if err := c.state.api.Delete(name, ctx); err != nil {
		return err
	}
	delete(c.state.members, name)
	return nil
}

func memberOperations[M any](member *M) (controlbinding.Operations, error) {
	ops, ok := any(member).(controlbinding.Operations)
	if !ok {
		return nil, fmt.Errorf("collection: %T must provide scope operations", member)
	}
	return ops, nil
}
