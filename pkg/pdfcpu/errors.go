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

package pdfcpu

import (
	"errors"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// VersionRequirementError reports an element requiring a newer PDF version.
type VersionRequirementError = model.VersionRequirementError

var (
	// ErrInvalidPageNumber signals an invalid page number.
	ErrInvalidPageNumber = errors.New("invalid page number")

	// ErrMissingAnnotation signals a missing required annotation.
	ErrMissingAnnotation = errors.New("missing annotation")

	// ErrMissingContext signals a missing required Go context.
	ErrMissingContext = model.ErrMissingContext

	// ErrMissingImageReader signals a missing required image reader.
	ErrMissingImageReader = model.ErrMissingImageReader

	// ErrMissingOptimizationContext signals a missing required optimization context.
	ErrMissingOptimizationContext = errors.New("missing optimization context")

	// ErrMissingPageNumbers signals missing required page numbers.
	ErrMissingPageNumbers = errors.New("missing page numbers")

	// ErrMissingPDFContext signals a missing required PDF context.
	ErrMissingPDFContext = model.ErrMissingPDFContext

	// ErrMissingPDFInfo signals missing required PDF info.
	ErrMissingPDFInfo = errors.New("missing PDF info")

	// ErrMissingReadContext signals a missing required PDF read context.
	ErrMissingReadContext = errors.New("missing PDF read context")

	// ErrMissingReader signals a missing required reader.
	ErrMissingReader = errors.New("missing reader")

	// ErrMissingStreamDict signals a missing required PDF stream dictionary.
	ErrMissingStreamDict = errors.New("missing PDF stream dictionary")

	// ErrMissingWatermarkConfiguration signals a missing required watermark configuration.
	ErrMissingWatermarkConfiguration = errors.New("missing watermark configuration")

	// ErrMissingWatermarks signals missing required watermarks.
	ErrMissingWatermarks = errors.New("missing watermarks")

	// ErrMissingWriteContext signals a missing required PDF write context.
	ErrMissingWriteContext = errors.New("missing PDF write context")

	// ErrMissingXRefTable signals a missing required PDF cross-reference table.
	ErrMissingXRefTable = model.ErrMissingXRefTable

	// ErrNoSignatures signals that a PDF has no signatures to process.
	ErrNoSignatures = errors.New("no signatures present")

	// ErrUnsupportedResource signals that extraction skipped a resource because
	// pdfcpu does not support its type or filter. Use errors.Is to identify it.
	ErrUnsupportedResource = errors.New("unsupported resource")

	// ErrUnsupportedVersion reports an unsupported PDF version for the requested
	// operation. It is a stable sentinel intended for use with errors.Is.
	ErrUnsupportedVersion = errors.New("PDF 2.0 unsupported for this operation")

	// ErrVersionTooLow signals that an element requires a newer declared PDF version.
	ErrVersionTooLow = model.ErrVersionTooLow
)
