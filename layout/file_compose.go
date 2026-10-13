package layout

import "path/filepath"

// ComposePath binds the handle to path and resets composition metadata.
func (f *File) ComposePath(path string) {
	f.path = filepath.Clean(path)
	f.composeBase = ""
	f.composedBase = false
}

func (f *File) SetComposeBase(path string) {
	f.composeBase = filepath.Clean(path)
	f.composedBase = true
}

func (f *File) SetDeclaredPath(path string) {
	f.declaredPath = path
	f.hasDeclared = true
}
