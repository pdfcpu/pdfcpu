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

// ErrVersionTooLow signals that an element requires a newer declared PDF version.
var ErrVersionTooLow = errors.New("PDF version too low")

// VersionRequirementError reports an element requiring a newer PDF version.
type VersionRequirementError struct {
	Element         string
	ActualVersion   Version
	RequiredVersion Version
}

// Error implements error.
func (e *VersionRequirementError) Error() string {
	return fmt.Sprintf("%s: unsupported in version %s, requires version %s", e.Element, e.ActualVersion, e.RequiredVersion)
}

// Is classifies this error as ErrVersionTooLow.
func (e *VersionRequirementError) Is(target error) bool {
	return target == ErrVersionTooLow
}
