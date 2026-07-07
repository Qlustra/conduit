// Package pipeline runs buffered byte and typed-layout tasks in declaration
// order.
//
// Byte tasks operate on layout files, directories of files, or in-memory blobs.
// Typed tasks operate on cached entries from layout.Slot and layout.FileSlot.
// Each task snapshots its configured inputs when it runs, performs in-memory
// processing first, and only then executes its sink operations.
package pipeline
