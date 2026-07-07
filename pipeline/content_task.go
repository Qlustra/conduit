package pipeline

import "sync"

type byteTaskRuntime struct {
	stepperRuntime[Blob, inputSource[Blob], byteStep, byteSinkOperation, byteSink]
}

func createByteStepperRuntime(snapshot *stepperState[Blob, inputSource[Blob], byteStep, byteSinkOperation, byteSink], runMu *sync.Mutex) Runtime {
	return newStepperRuntime(snapshot, runMu, sinkByteItems)
}

func newByteStepperTask(name string, cardinality taskCardinality) *byteStepper {
	return &byteStepper{
		stepperTask: newStepperTask[Blob, inputSource[Blob], byteStep, byteSinkOperation, byteSink, byteTaskRuntime](name, cardinality, createByteStepperRuntime),
	}
}
