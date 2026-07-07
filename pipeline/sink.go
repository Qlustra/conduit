package pipeline

import "github.com/qlustra/conduit/layout"

type sink[K comparable] interface {
	getKind() K
	isKind(kind K) bool
	getName() string
	getKey() string
	validateCardinality(cardinality taskCardinality) bool
}

type baseSink[K comparable] struct {
	key  string
	kind K
	lctx layout.Context
}

func (s baseSink[K]) getName() string {
	return ""
}

func (s baseSink[K]) getKind() K {
	return s.kind
}

func (s baseSink[K]) isKind(kind K) bool {
	return s.kind == kind
}

func (s baseSink[K]) getKey() string {
	return s.key
}

func (s baseSink[K]) resolveLayoutContext(fallback layout.Context) layout.Context {
	return resolveLayoutContext(s.lctx, fallback)
}

func (s baseSink[K]) validateCardinality(_ taskCardinality) bool {
	return true
}

func getSinkId[K comparable](s sink[K]) string {
	if len(s.getKey()) == 0 {
		return s.getName()
	}
	return s.getKey()
}
