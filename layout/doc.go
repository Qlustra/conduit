// Package layout provides filesystem carriers and local operation contracts.
//
// Dir, File, and Exec are stateless path handles. Link, Format, and TextTemplate
// retain explicit disk and memory state. Slot, FileSlot, and LinkSlot cache typed
// directory, regular-file, and symlink members and supply them to Sail.
//
// FileAPI, DirAPI, ExecAPI, and the typed content and collection APIs describe
// each carrier's complete exported method set. ContentFamily and
// CollectionFamily select the instantiated API through private carrier
// witnesses, retaining content, render-context, and member types.
//
// Declare carriers with layout role tags and separate path tags, then compile
// and compose the tree through the root conduit package. Composition configures
// slots to construct missing members. Carrier methods act locally; the owning
// conduit.Layout provides whole-tree operations through Sail traversal.
package layout
