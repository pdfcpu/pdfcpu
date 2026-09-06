/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import "fmt"

// ProgressStage identifies a stable, coarse operation stage.
type ProgressStage string

const (
	// ProgressStageReading identifies input reading.
	ProgressStageReading ProgressStage = "reading"

	// ProgressStageValidating identifies PDF validation.
	ProgressStageValidating ProgressStage = "validating"

	// ProgressStageOptimizing identifies PDF optimization.
	ProgressStageOptimizing ProgressStage = "optimizing"

	// ProgressStageProcessing identifies operation-specific processing.
	ProgressStageProcessing ProgressStage = "processing"

	// ProgressStageWriting identifies output serialization.
	ProgressStageWriting ProgressStage = "writing"

	// ProgressStageCommitting identifies publication of staged output.
	ProgressStageCommitting ProgressStage = "committing"

	// ProgressStageCleanup identifies cleanup after an operation.
	ProgressStageCleanup ProgressStage = "cleanup"

	// ProgressStageRollback identifies restoration after an operation failure.
	ProgressStageRollback ProgressStage = "rollback"
)

// ProgressEvent describes the current stage of one API operation.
type ProgressEvent struct {
	// Stage identifies the operation stage that is about to begin.
	Stage ProgressStage

	// Input identifies the current input when known.
	Input string

	// Item is the one-based current input position when known.
	Item int

	// Total is the total number of inputs when known.
	Total int
}

// ProgressObserver receives ordered events synchronously before each reported stage begins.
// Returning an error aborts the operation. File operations treat observer failures like processing failures and clean
// up staged output before returning the error.
type ProgressObserver func(ProgressEvent) error

// ProgressOptions configures optional per-call progress observation.
type ProgressOptions struct {
	// Observer receives progress events. A nil observer disables observation.
	Observer ProgressObserver

	// Input identifies the current input for stream-based operations.
	Input string

	// Item is the one-based current input position when known.
	Item int

	// Total is the total number of inputs when known.
	Total int
}

// ProgressError reports a failure returned by a ProgressObserver.
type ProgressError struct {
	// Event is the event rejected by the observer.
	Event ProgressEvent

	// Err is the error returned by the observer.
	Err error
}

// Error implements error.
func (e *ProgressError) Error() string {
	return fmt.Sprintf("report %s progress: %v", e.Event.Stage, e.Err)
}

// Unwrap returns the observer failure.
func (e *ProgressError) Unwrap() error {
	return e.Err
}

func reportProgress(options ProgressOptions, stage ProgressStage) error {
	if options.Observer == nil {
		return nil
	}

	event := ProgressEvent{
		Stage: stage,
		Input: options.Input,
		Item:  options.Item,
		Total: options.Total,
	}
	if err := options.Observer(event); err != nil {
		return &ProgressError{Event: event, Err: err}
	}
	return nil
}
