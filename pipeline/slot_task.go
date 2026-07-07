package pipeline

import (
	"sync"
)

type slotTaskRuntime[I any] struct {
	stepperRuntime[I, inputSource[I], slotStep[I], slotSinkOperation, slotSink]
}

func createSlotStepperRuntime[I any](snapshot *stepperState[I, inputSource[I], slotStep[I], slotSinkOperation, slotSink], runMu *sync.Mutex) Runtime {
	return newStepperRuntime[I, inputSource[I], slotStep[I], slotSinkOperation, slotSink](snapshot, runMu, sinkSlotItems[I])
}

// slotStepper is a typed pipeline task.
type slotStepper[I any] struct {
	*stepperTask[I, inputSource[I], slotStep[I], slotSinkOperation, slotSink, slotTaskRuntime[I]]
}

func newSlotStepper[I any](name string) *slotStepper[I] {
	return &slotStepper[I]{
		stepperTask: newStepperTask[I, inputSource[I], slotStep[I], slotSinkOperation, slotSink, slotTaskRuntime[I]](name, multiTask, createSlotStepperRuntime[I]),
	}
}
