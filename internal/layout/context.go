package layout

import "os"

type Context struct {
	DirMode  os.FileMode
	FileMode os.FileMode
	ExecMode os.FileMode
}

var DefaultContext = Context{
	DirMode:  0o755,
	FileMode: 0o644,
	ExecMode: 0o755,
}
