package conduit

import "github.com/qlustra/conduit/internal/managed"

// Layout retains a live layout tree and provides its deep operations.
type Layout[T any] = managed.Layout[T]
