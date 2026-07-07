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

// ErrUnsupportedVersion reports an unsupported PDF version for the requested
// operation. It is a stable sentinel intended for use with errors.Is.
var ErrUnsupportedVersion = errors.New("PDF 2.0 unsupported for this operation")

// ErrUnsupportedResource signals that extraction skipped a resource because
// pdfcpu does not support its type or filter. Use errors.Is to identify it.
var ErrUnsupportedResource = errors.New("unsupported resource")

// ErrNoSignatures signals that a PDF has no signatures to process.
var ErrNoSignatures = errors.New("no signatures present")

// ErrMissingPDFContext signals a missing required PDF context.
var ErrMissingPDFContext = model.ErrMissingPDFContext

// ErrMissingPDFInfo signals missing required PDF info.
var ErrMissingPDFInfo = errors.New("missing PDF info")

// ErrMissingReadContext signals a missing required PDF read context.
var ErrMissingReadContext = errors.New("missing PDF read context")

// ErrMissingWriteContext signals a missing required PDF write context.
var ErrMissingWriteContext = errors.New("missing PDF write context")

// ErrMissingXRefTable signals a missing required PDF cross-reference table.
var ErrMissingXRefTable = model.ErrMissingXRefTable

// ErrMissingStreamDict signals a missing required PDF stream dictionary.
var ErrMissingStreamDict = errors.New("missing PDF stream dictionary")

// ErrMissingOptimizationContext signals a missing required optimization context.
var ErrMissingOptimizationContext = errors.New("missing optimization context")

// ErrMissingPageNumbers signals missing required page numbers.
var ErrMissingPageNumbers = errors.New("missing page numbers")

// ErrMissingAnnotation signals a missing required annotation.
var ErrMissingAnnotation = errors.New("missing annotation")

// ErrMissingReader signals a missing required reader.
var ErrMissingReader = errors.New("missing reader")

// ErrMissingImageReader signals a missing required image reader.
var ErrMissingImageReader = model.ErrMissingImageReader

// ErrMissingWatermarkConfiguration signals a missing required watermark configuration.
var ErrMissingWatermarkConfiguration = errors.New("missing watermark configuration")

// ErrMissingWatermarks signals missing required watermarks.
var ErrMissingWatermarks = errors.New("missing watermarks")

// ErrInvalidPageNumber signals an invalid page number.
var ErrInvalidPageNumber = errors.New("invalid page number")
