package pipeline

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
)

// Entries describes a slot-backed set of typed entries.
type Entries[T any] interface {
	entrySet()

	// snapshot returns the current cached entries as pipeline items.
	snapshot() []Item[T]

	// target composes or retrieves a target item for key.
	target(key string) (Item[T], error)

	// put updates the backing cache for key without writing to disk.
	put(key string, value T)
}

// Entry describes one fixed slot-backed typed entry.
type Entry[T any] interface {
	entryRef()

	// target composes or retrieves the target item.
	target() (Item[T], error)

	// put updates the backing cache without writing to disk.
	put(value T)

	// key returns the fixed entry key.
	key() string
}

type slotEntries[T any] struct{ slot *layout.Slot[T] }
type fileSlotEntries[T any] struct{ slot *layout.FileSlot[T] }
type slotEntry[T any] struct {
	slot *layout.Slot[T]
	name string
}
type fileSlotEntry[T any] struct {
	slot *layout.FileSlot[T]
	name string
}

// SlotEntries returns a descriptor for all cached entries in slot.
func SlotEntries[T any](slot *layout.Slot[T]) Entries[T] {
	return slotEntries[T]{slot: slot}
}

// FileSlotEntries returns a descriptor for all cached entries in slot.
func FileSlotEntries[T any](slot *layout.FileSlot[T]) Entries[T] {
	return fileSlotEntries[T]{slot: slot}
}

// SlotEntry returns a descriptor for one named slot entry.
func SlotEntry[T any](slot *layout.Slot[T], name string) Entry[T] {
	return slotEntry[T]{slot: slot, name: name}
}

// FileSlotEntry returns a descriptor for one named file-slot entry.
func FileSlotEntry[T any](slot *layout.FileSlot[T], name string) Entry[T] {
	return fileSlotEntry[T]{slot: slot, name: name}
}

func (s slotEntries[T]) entrySet()     {}
func (s fileSlotEntries[T]) entrySet() {}
func (s slotEntry[T]) entryRef()       {}
func (s fileSlotEntry[T]) entryRef()   {}

func (s slotEntries[T]) snapshot() []Item[T] {
	items := make([]Item[T], 0)
	if s.slot == nil {
		return items
	}
	for _, entry := range s.slot.Entries() {
		items = append(items, itemFromSlotEntry(s.slot, entry))
	}
	return items
}

func (s fileSlotEntries[T]) snapshot() []Item[T] {
	items := make([]Item[T], 0)
	if s.slot == nil {
		return items
	}
	for _, entry := range s.slot.Entries() {
		items = append(items, itemFromFileSlotEntry(s.slot, entry))
	}
	return items
}

func (s slotEntries[T]) target(key string) (Item[T], error) {
	if s.slot == nil {
		return Item[T]{}, fmt.Errorf("slot entries target is nil")
	}
	value, err := s.slot.At(key)
	if err != nil {
		return Item[T]{}, err
	}
	return itemFromSlotEntryName(s.slot, key, value), nil
}

func (s fileSlotEntries[T]) target(key string) (Item[T], error) {
	if s.slot == nil {
		return Item[T]{}, fmt.Errorf("file slot entries target is nil")
	}
	value, err := s.slot.At(key)
	if err != nil {
		return Item[T]{}, err
	}
	return itemFromFileSlotEntryName(s.slot, key, value), nil
}

func (s slotEntries[T]) put(key string, value T) {
	if s.slot != nil {
		s.slot.Put(key, value)
	}
}

func (s fileSlotEntries[T]) put(key string, value T) {
	if s.slot != nil {
		s.slot.Put(key, value)
	}
}

func (s slotEntry[T]) key() string     { return s.name }
func (s fileSlotEntry[T]) key() string { return s.name }

func (s slotEntry[T]) target() (Item[T], error) {
	if s.slot == nil {
		return Item[T]{}, fmt.Errorf("slot entry target is nil")
	}
	value, err := s.slot.At(s.name)
	if err != nil {
		return Item[T]{}, err
	}
	return itemFromSlotEntryName(s.slot, s.name, value), nil
}

func (s fileSlotEntry[T]) target() (Item[T], error) {
	if s.slot == nil {
		return Item[T]{}, fmt.Errorf("file slot entry target is nil")
	}
	value, err := s.slot.At(s.name)
	if err != nil {
		return Item[T]{}, err
	}
	return itemFromFileSlotEntryName(s.slot, s.name, value), nil
}

func (s slotEntry[T]) put(value T) {
	if s.slot != nil {
		s.slot.Put(s.name, value)
	}
}

func (s fileSlotEntry[T]) put(value T) {
	if s.slot != nil {
		s.slot.Put(s.name, value)
	}
}
