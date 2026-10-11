package contracts

// Composable accepts the physical path of a composed carrier.
type Composable interface {
	ComposePath(string)
}

// ComposeBaseAware accepts the base shared by the composed tree and its members.
type ComposeBaseAware interface {
	SetComposeBase(string)
}

// DeclaredPathAware accepts a carrier's authored path fragment.
type DeclaredPathAware interface {
	SetDeclaredPath(string)
}

// MemberComposer prepares typed member storage at a physical path. The target
// is a pointer to the collection's member value and may hold a nil pointer.
type MemberComposer interface {
	ComposeMember(target any, path, composeBase string) error
}

// MemberCompositionAware accepts the composer used to construct cache misses.
type MemberCompositionAware interface {
	SetMemberComposer(MemberComposer)
}
