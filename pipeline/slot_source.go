package pipeline

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
)

type slotType uint8

const (
	slotTypeUnknown = iota
	slotTypeSlot
	slotTypeFile
)

func itemFromSlotEntry[T any](slot *layout.Slot[T], entry layout.SlotEntry[T]) Item[T] {
	return itemFromSlotEntryName(slot, entry.Name, entry.Item)
}

func itemFromSlotEntryName[T any](slot *layout.Slot[T], name string, value T) Item[T] {
	dir := slot.Root().Dir(name)
	path := itemPathFromDir(dir, name)
	return Item[T]{Key: name, Name: dir.Base(), Path: path, Dir: dir, Value: value}
}

func itemFromFileSlotEntry[T any](slot *layout.FileSlot[T], entry layout.FileSlotEntry[T]) Item[T] {
	return itemFromFileSlotEntryName(slot, entry.Name, entry.Item)
}

func itemFromFileSlotEntryName[T any](slot *layout.FileSlot[T], name string, value T) Item[T] {
	file := slot.Root().File(name)
	path := itemPathFromFile(file, name)
	return Item[T]{Key: name, Name: file.Base(), Path: path, File: file, Value: value}
}

type slotSource[T any] struct {
	slotType slotType
	slot     *layout.Slot[T]
	fileSlot *layout.FileSlot[T]
}

func createSlotSourceFromSlot[T any](slot *layout.Slot[T]) slotSource[T] {
	return slotSource[T]{
		slotType: slotTypeSlot,
		slot:     slot,
	}
}

func createSlotSourceFromFileSlot[T any](slot *layout.FileSlot[T]) slotSource[T] {
	return slotSource[T]{
		slotType: slotTypeFile,
		fileSlot: slot,
	}
}

func (s slotSource[T]) snapshotItems() ([]Item[T], error) {
	items := make([]Item[T], 0)
	if s.slotType == slotTypeSlot {
		if s.slot == nil {
			return items, fmt.Errorf("missing slot")
		}
		for _, entry := range s.slot.Entries() {
			items = append(items, itemFromSlotEntry(s.slot, entry))
		}
		return items, nil
	} else if s.slotType == slotTypeFile {
		if s.fileSlot == nil {
			return items, fmt.Errorf("missing file slot")
		}
		for _, entry := range s.fileSlot.Entries() {
			items = append(items, itemFromFileSlotEntry(s.fileSlot, entry))
		}
		return items, nil
	}

	return items, fmt.Errorf("unknown slot type")
}

func (s slotSource[T]) at(key string) (Item[T], error) {
	if s.slotType == slotTypeSlot {
		if s.slot == nil {
			return Item[T]{}, fmt.Errorf("slot entries source is nil")
		}
		value, err := s.slot.At(key)
		if err != nil {
			return Item[T]{}, err
		}
		return itemFromSlotEntryName(s.slot, key, value), nil
	} else if s.slotType == slotTypeFile {
		if s.fileSlot == nil {
			return Item[T]{}, fmt.Errorf("file slot entries source is nil")
		}
		value, err := s.fileSlot.At(key)
		if err != nil {
			return Item[T]{}, err
		}
		return itemFromFileSlotEntryName(s.fileSlot, key, value), nil
	}

	return Item[T]{}, fmt.Errorf("unknown slot type")
}
func (s slotSource[T]) put(key string, value T) {
	if s.slotType == slotTypeSlot {
		if s.slot != nil {
			s.slot.Put(key, value)
		}
	} else if s.slotType == slotTypeFile {
		if s.fileSlot != nil {
			s.fileSlot.Put(key, value)
		}
	}
}
