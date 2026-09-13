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
)

// ErrInputSizeLimit reports a PDF input exceeding its configured byte limit.
var ErrInputSizeLimit = errors.New("PDF input size limit exceeded")

// CheckInputSize validates the input byte limit and checks a PDF input size against it.
func (l ResourceLimits) CheckInputSize(size int64) error {
	if l.MaxInputBytes < 0 {
		return fmt.Errorf("maxInputBytes must be >= 0: %d", l.MaxInputBytes)
	}
	if l.MaxInputBytes > 0 && size > l.MaxInputBytes {
		return fmt.Errorf("maximum %d bytes: %w", l.MaxInputBytes, ErrInputSizeLimit)
	}
	return nil
}
