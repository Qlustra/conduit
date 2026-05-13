// Package formats provides codec-backed typed files for Conduit layouts.
//
// The types in this package are thin wrappers around layout.Format that bind a
// concrete serialization format to a file path. Use JSONFile, YAMLFile, or
// TOMLFile when a layout field should carry typed content in memory and be read
// from or written to disk explicitly.
package formats
