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

import (
	"errors"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var (
	// ErrAttachmentOutputCollision signals attachments resolving to the same output path.
	ErrAttachmentOutputCollision = errors.New("attachment output collision")

	// ErrBookletImageOutputConflict signals that booklet output aliases an image input.
	ErrBookletImageOutputConflict = errors.New("booklet image output aliases input")

	// ErrCircularBookmarks signals a circular bookmark tree.
	ErrCircularBookmarks = pdfcpu.ErrCircularBookmarks

	// ErrDuplicateCertificateDestination signals certificate inputs targeting the same installed file.
	ErrDuplicateCertificateDestination = errors.New("duplicate certificate destination")

	// ErrDuplicatePostScriptName aliases the lower-layer duplicate-name sentinel.
	ErrDuplicatePostScriptName = font.ErrDuplicatePostScriptName

	// ErrExistingBookmarks signals that adding bookmarks would conflict with existing bookmarks.
	ErrExistingBookmarks = pdfcpu.ErrExistingBookmarks

	// ErrGridImageOutputConflict signals that grid output aliases an image input.
	ErrGridImageOutputConflict = errors.New("grid image output aliases input")

	// ErrImportImagesOutputConflict signals that import-images output aliases an image input.
	ErrImportImagesOutputConflict = errors.New("import images output aliases image input")

	// ErrInvalidBookmark signals an invalid bookmark tree.
	ErrInvalidBookmark = pdfcpu.ErrInvalidBookmark

	// ErrInvalidBookmarkJSON signals malformed bookmark JSON data.
	ErrInvalidBookmarkJSON = pdfcpu.ErrInvalidBookmarkJSON

	// ErrInvalidCSV signals malformed or incomplete CSV form data.
	ErrInvalidCSV = errors.New("invalid csv input file")

	// ErrInvalidCutConfiguration signals an invalid cut configuration.
	ErrInvalidCutConfiguration = errors.New("invalid cut configuration")

	// ErrInvalidFormData signals structurally invalid decoded form data.
	ErrInvalidFormData = errors.New("invalid form data")

	// ErrInvalidImageSelection signals an invalid image object or page resource selection.
	ErrInvalidImageSelection = errors.New("invalid image selection")

	// ErrInvalidImportConfiguration signals an invalid image import configuration.
	ErrInvalidImportConfiguration = errors.New("invalid import configuration")

	// ErrInvalidJSON signals invalid JSON form data.
	ErrInvalidJSON = errors.New("invalid JSON encoding")

	// ErrInvalidPageBoundaries signals invalid page boundaries for an operation.
	ErrInvalidPageBoundaries = errors.New("invalid page boundaries")

	// ErrInvalidPageConfiguration signals an invalid page configuration.
	ErrInvalidPageConfiguration = errors.New("invalid page configuration")

	// ErrInvalidPageLayout signals an unsupported page layout.
	ErrInvalidPageLayout = errors.New("invalid page layout")

	// ErrInvalidPageMode signals an unsupported page mode.
	ErrInvalidPageMode = errors.New("invalid page mode")

	// ErrInvalidRotation signals a rotation that is not a multiple of 90 degrees.
	ErrInvalidRotation = errors.New("invalid rotation")

	// ErrInvalidResizeConfiguration signals an invalid resize configuration.
	ErrInvalidResizeConfiguration = errors.New("invalid resize configuration")

	// ErrInvalidSplitPageNumberSequence signals an invalid split page number sequence.
	ErrInvalidSplitPageNumberSequence = errors.New("invalid split page number sequence")

	// ErrInvalidSplitSpan signals an invalid split span.
	ErrInvalidSplitSpan = errors.New("invalid split span")

	// ErrInvalidUnicodePlane signals an invalid Unicode plane.
	ErrInvalidUnicodePlane = errors.New("invalid Unicode plane")

	// ErrInvalidZoomConfiguration signals an invalid zoom configuration.
	ErrInvalidZoomConfiguration = errors.New("invalid zoom configuration")

	// ErrMissingAnnotation signals a missing required annotation.
	ErrMissingAnnotation = pdfcpu.ErrMissingAnnotation

	// ErrMissingBookmarks signals that no bookmarks were provided.
	ErrMissingBookmarks = errors.New("missing bookmarks")

	// ErrMissingBookletConfiguration signals a missing booklet configuration.
	ErrMissingBookletConfiguration = errors.New("missing booklet configuration")

	// ErrMissingBoxConfiguration signals a missing box configuration.
	ErrMissingBoxConfiguration = errors.New("missing box configuration")

	// ErrMissingCertificate signals a missing required certificate.
	ErrMissingCertificate = pdfcpu.ErrMissingCertificate

	// ErrMissingCertificateInput signals a missing required certificate input file.
	ErrMissingCertificateInput = errors.New("missing certificate input")

	// ErrMissingConfiguration signals a missing pdfcpu configuration.
	ErrMissingConfiguration = errors.New("missing configuration")

	// ErrMissingCutConfiguration signals a missing cut configuration.
	ErrMissingCutConfiguration = errors.New("missing cut configuration")

	// ErrMissingDigestFunction signals a missing required digest function.
	ErrMissingDigestFunction = errors.New("missing digest function")

	// ErrMissingFontInput signals a missing required font input file.
	ErrMissingFontInput = errors.New("missing font input")

	// ErrMissingFormInput signals a missing required form input reader or file.
	ErrMissingFormInput = errors.New("missing form input")

	// ErrMissingGridConfiguration signals a missing grid configuration.
	ErrMissingGridConfiguration = errors.New("missing grid configuration")

	// ErrMissingImageInput signals a missing required image input.
	ErrMissingImageInput = errors.New("missing image input")

	// ErrMissingImageReader signals a missing required image reader.
	ErrMissingImageReader = pdfcpu.ErrMissingImageReader

	// ErrMissingJSONInput signals a missing required JSON input file.
	ErrMissingJSONInput = errors.New("missing JSON input")

	// ErrMissingJSONOutput signals a missing required JSON output file.
	ErrMissingJSONOutput = errors.New("missing JSON output")

	// ErrMissingJSONReader signals a missing required JSON input reader.
	ErrMissingJSONReader = errors.New("missing JSON reader")

	// ErrMissingJSONWriter signals a missing required JSON output writer.
	ErrMissingJSONWriter = errors.New("missing JSON writer")

	// ErrMissingNUpConfiguration signals a missing n-up configuration.
	ErrMissingNUpConfiguration = errors.New("missing n-up configuration")

	// ErrMissingPageBoundaries signals missing required page boundaries.
	ErrMissingPageBoundaries = errors.New("missing page boundaries")

	// ErrMissingPDFContext signals a missing required PDF context.
	ErrMissingPDFContext = pdfcpu.ErrMissingPDFContext

	// ErrMissingPDFInput signals a missing required PDF input file.
	ErrMissingPDFInput = errors.New("missing PDF input")

	// ErrMissingPDFOutput signals a missing required PDF output file.
	ErrMissingPDFOutput = errors.New("missing PDF output")

	// ErrMissingPDFReadSeeker signals a missing required PDF input reader.
	ErrMissingPDFReadSeeker = errors.New("missing PDF read seeker")

	// ErrMissingPDFReadWriteSeeker signals a missing required PDF input/output seeker.
	ErrMissingPDFReadWriteSeeker = errors.New("missing PDF read write seeker")

	// ErrMissingPDFWriter signals a missing required PDF output writer.
	ErrMissingPDFWriter = errors.New("missing PDF writer")

	// ErrMissingReader signals a missing required reader.
	ErrMissingReader = pdfcpu.ErrMissingReader

	// ErrMissingResizeConfiguration signals a missing resize configuration.
	ErrMissingResizeConfiguration = errors.New("missing resize configuration")

	// ErrMissingSplitPageNumbers signals missing split page numbers.
	ErrMissingSplitPageNumbers = errors.New("missing split page numbers")

	// ErrMissingWatermarkConfiguration signals a missing required watermark configuration.
	ErrMissingWatermarkConfiguration = pdfcpu.ErrMissingWatermarkConfiguration

	// ErrMissingWatermarks signals missing required watermarks.
	ErrMissingWatermarks = pdfcpu.ErrMissingWatermarks

	// ErrMissingXRefTable signals a missing required PDF cross-reference table.
	ErrMissingXRefTable = pdfcpu.ErrMissingXRefTable

	// ErrMissingZoomConfiguration signals a missing zoom configuration.
	ErrMissingZoomConfiguration = errors.New("missing zoom configuration")

	// ErrNoAttachmentAdded signals that no attachment was added.
	ErrNoAttachmentAdded = errors.New("no attachment added")

	// ErrNoAttachmentRemoved signals that no attachment was removed.
	ErrNoAttachmentRemoved = errors.New("no attachment removed")

	// ErrNoBookmarks signals that a PDF has no bookmarks to process.
	ErrNoBookmarks = pdfcpu.ErrNoBookmarks

	// ErrNoCertificates signals that a certificate file contains no usable certificates.
	ErrNoCertificates = pdfcpu.ErrNoCertificates

	// ErrNoFormData signals missing form data for a form fill operation.
	ErrNoFormData = errors.New("missing form data")

	// ErrNoFormFieldsAffected signals that a form operation did not change any fields.
	ErrNoFormFieldsAffected = errors.New("no form fields affected")

	// ErrNoKeywordRemoved signals that a remove operation did not match any keyword.
	ErrNoKeywordRemoved = errors.New("no keyword removed")

	// ErrNoOp is retained for source compatibility.
	// Deprecated: viewer-preferences reset operations are idempotent and no longer return ErrNoOp.
	ErrNoOp = errors.New("no operation")

	// ErrNoPropertyRemoved signals that a remove operation did not match any property.
	ErrNoPropertyRemoved = errors.New("no property removed")

	// ErrNoSignatures signals that a PDF has no signatures to process.
	ErrNoSignatures = pdfcpu.ErrNoSignatures

	// Deprecated: use ErrNoBookmarks.
	ErrNoOutlines = ErrNoBookmarks

	// ErrNUpImageOutputConflict signals that n-up output aliases an image input.
	ErrNUpImageOutputConflict = errors.New("n-up image output aliases input")

	// Deprecated: use ErrExistingBookmarks.
	ErrOutlines = ErrExistingBookmarks

	// ErrUnknownFont aliases the lower-layer unknown-font sentinel.
	ErrUnknownFont = font.ErrUnknownFont

	// ErrUnsupportedCertificateFile signals an unsupported certificate input file.
	ErrUnsupportedCertificateFile = pdfcpu.ErrUnsupportedCertificateFile

	// ErrUnsupportedFontFile signals an unsupported font input file.
	ErrUnsupportedFontFile = errors.New("unsupported font file")

	// ErrUnsupportedFormDataFormat signals an unsupported form data format.
	ErrUnsupportedFormDataFormat = errors.New("unsupported data format")

	// ErrUpdateImagesOutputConflict signals that update-images output aliases the image input.
	ErrUpdateImagesOutputConflict = errors.New("update images output aliases image input")

	// ErrUserFontNotFound is retained as a compatibility alias for ErrUnknownFont.
	ErrUserFontNotFound = font.ErrUnknownFont
)
