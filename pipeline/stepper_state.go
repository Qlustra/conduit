package pipeline

import (
	"fmt"
)

type stepperState[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K]] struct {
	name        string
	configErr   error
	source      S
	steps       []P
	sinks       map[string]N
	sinkOrder   []N
	cardinality taskCardinality
}

func newStepperState[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K]](name string, cardinality taskCardinality) *stepperState[I, S, P, K, N] {
	return &stepperState[I, S, P, K, N]{
		name:        name,
		cardinality: cardinality,
		sinks:       make(map[string]N),
	}
}

func (s *stepperState[I, S, P, K, N]) taskName() string {
	return s.name
}

func (s *stepperState[I, S, P, K, N]) taskConfigErr() error {
	return s.configErr
}

func (s *stepperState[I, S, P, K, N]) addSink(sinkLabel string, sink N) {
	s.sinks[sinkLabel] = sink
	s.sinkOrder = append(s.sinkOrder, sink)
}

func (s *stepperState[I, S, P, K, N]) hasSinks() bool {
	return len(s.sinks) > 0
}

func (s *stepperState[I, S, P, K, N]) hasSource() bool {
	return any(s.source) != nil
}

func (s *stepperState[I, S, P, SO, N]) addExclusiveSink(sinkLabel string, sink N) {
	if _, ok := s.sinks[sinkLabel]; ok {
		s.configErr = fmt.Errorf("duplicate sink name %s", sinkLabel)
		return
	}
	s.addSink(sinkLabel, sink)
}

func (s *stepperState[I, S, P, SO, N]) addOnlySink(sinkLabel string, sink N) {
	if s.configErr != nil {
		return
	}
	if len(s.sinks) > 0 {
		s.configErr = fmt.Errorf("task %q already has a sink", s.name)
		return
	}
	s.addSink(sinkLabel, sink)
}

func (s *stepperState[I, S, P, K, N]) copySteps() []P {
	steps := make([]P, len(s.steps))
	copy(steps, s.steps)
	return steps
}

func (s *stepperState[I, S, P, K, N]) copySinks() map[string]N {
	sinks := make(map[string]N, len(s.sinks))
	for k, v := range s.sinks {
		sinks[k] = v
	}
	return sinks
}

func (s *stepperState[I, S, P, K, N]) copySinkOrder() []N {
	sinkOrder := make([]N, len(s.sinkOrder))
	copy(sinkOrder, s.sinkOrder)
	return sinkOrder
}

func (s *stepperState[I, S, P, K, N]) validate() error {
	if s.configErr != nil {
		return s.configErr
	}
	if !s.hasSinks() {
		return fmt.Errorf("missing sink")
	}
	if !s.hasSource() {
		return fmt.Errorf("missing source")
	}
	return nil
}

func (s *stepperState[I, S, P, K, N]) snapshotState() *stepperState[I, S, P, K, N] {
	return &stepperState[I, S, P, K, N]{
		name:        s.name,
		source:      s.source,
		steps:       s.copySteps(),
		sinks:       s.copySinks(),
		sinkOrder:   s.copySinkOrder(),
		configErr:   s.configErr,
		cardinality: s.cardinality,
	}
}

func (s *stepperState[I, S, P, K, N]) snapshotItems() ([]Item[I], error) {
	items := make([]Item[I], 0)
	var err error
	if s.hasSource() {
		items, err = s.source.snapshotItems()
		if err != nil {
			return nil, fmt.Errorf("snapshot source items: %w", err)
		}
	}
	return items, nil
}
