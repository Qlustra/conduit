// Package layout models filesystem structure as explicit Go values.
//
// Layouts are ordinary structs with layout tags that are bound to a root path
// through Compose. Once composed, callers explicitly move state between memory
// and disk through operations such as EnsureDeep, LoadDeep, DiscoverDeep,
// ScanDeep, SyncDeep, DefaultDeep, RenderDeep, and ValidateDeep.
//
// The package exposes raw node types such as Dir, File, Exec, and Link, along
// with higher-level stateful wrappers such as Format, Slot, FileSlot, and
// TextTemplate.
package layout
