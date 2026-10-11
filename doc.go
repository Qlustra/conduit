// Package conduit compiles declared filesystem trees and operates their live
// Layout managers through Sail.
//
// Compile[T] recognizes a tree declaration. Structure.Compose constructs and
// binds a fresh tree to a filesystem root without touching disk. Layout.Tree
// exposes its live carriers; Layout methods explicitly ensure, load, discover,
// scan, sync, default, render, unload, or validate the whole tree.
//
// Declarations use layout tags for participation roles and path tags for local
// paths. Untagged fields do not participate. Package layout contains filesystem
// carriers and local operation contracts; package formats contains typed files.
package conduit
