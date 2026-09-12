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

// Package contextutil provides shared checks for required Go contexts.
package contextutil

import (
	"context"
	"errors"
)

// ErrMissingContext signals a missing required Go context.
var ErrMissingContext = errors.New("missing context")

// Check returns ErrMissingContext for nil, or the context's cancellation or deadline error.
func Check(c context.Context) error {
	if c == nil {
		return ErrMissingContext
	}
	return c.Err()
}
