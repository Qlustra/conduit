## v0.1.0

First stable-shaped release.

Conduit is a Go library for composing filesystem-backed layouts as typed structures, then operating on them through explicit deep phases such as ensure, discover, load, validate, render, and sync.

This release marks the point where the library’s core model feels complete enough to treat as stable in practice:
- typed filesystem nodes for directories, files, executables, links, and dynamic slots
- explicit traversal phases for structure creation, discovery, loading, validation, rendering, scanning, and synchronization
- support for both static layouts and dynamic collections of directories, files, and links
- reporting hooks for observing deep operations path by path

This version also reflects a security and correctness pass over path handling and symlink behavior:
- path semantics were tightened where ambiguity or unsafe behavior was possible
- symlink handling was reviewed and hardened
- documentation was brought into line with implementation
- a small number of breaking changes were introduced where earlier behavior was too loose

From this point on, the changelog tracks notable changes from one stable-shaped release to the next.
