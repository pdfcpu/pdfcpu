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

package model

import (
	"errors"
	"fmt"
	"testing"
)

type validationCauseError struct {
	value string
}

func (e *validationCauseError) Error() string {
	return e.value
}

func TestWithValidationErrorObjectRejectsUnknownAttribution(t *testing.T) {
	if err := WithValidationErrorObject(nil, 10); err != nil {
		t.Fatalf("got %v, want nil", err)
	}

	cause := errors.New("validation failed")
	for _, objNr := range []int{-1, 0} {
		if err := WithValidationErrorObject(cause, objNr); err != cause {
			t.Fatalf("object %d: got %v, want original error", objNr, err)
		}
	}
}

func TestWithValidationErrorObjectPreservesErrorContracts(t *testing.T) {
	sentinel := errors.New("validation failed")
	cause := &validationCauseError{value: "invalid entry"}
	err := WithValidationErrorObject(fmt.Errorf("%w: %w", sentinel, cause), 20)

	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want sentinel %v", err, sentinel)
	}

	var causeErr *validationCauseError
	if !errors.As(err, &causeErr) {
		t.Fatalf("got %T, want *validationCauseError", err)
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("got %T, want *ValidationError", err)
	}
	if validationErr.ObjectNumber() != 20 {
		t.Fatalf("object number = %d, want 20", validationErr.ObjectNumber())
	}
}

func TestWithValidationErrorObjectPreservesInnerAttribution(t *testing.T) {
	inner := WithValidationErrorObject(errors.New("invalid referenced object"), 20)
	wrapped := fmt.Errorf("validate parent: %w", inner)
	outer := WithValidationErrorObject(wrapped, 10)

	var validationErr *ValidationError
	if !errors.As(outer, &validationErr) {
		t.Fatalf("got %T, want *ValidationError", outer)
	}
	if validationErr.ObjectNumber() != 20 {
		t.Fatalf("object number = %d, want inner object 20", validationErr.ObjectNumber())
	}
}

func TestDigestedSpecViolationMessageUsesTypedAttribution(t *testing.T) {
	err := WithValidationErrorObject(errors.New("invalid structure element"), 20)
	got := digestedSpecViolationMessage(err)
	if want := "spec violation (obj#:20): invalid structure element"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDigestedSpecViolationMessageOmitsUnknownObject(t *testing.T) {
	got := digestedSpecViolationMessage(errors.New("invalid structure element"))
	if want := "spec violation: invalid structure element"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
