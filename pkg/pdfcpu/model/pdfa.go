/*
Copyright 2025 The pdfcpu Authors.

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

import "fmt"

// PDF/A OutputIntent Subtypes (ISO 19005)
const (
	// GTS_PDFA1 represents PDF/A-1 conformance (ISO 19005-1:2005, based on PDF 1.4)
	GTS_PDFA1 = "GTS_PDFA1"

	// GTS_PDFA2 represents PDF/A-2 conformance (ISO 19005-2:2011, based on PDF 1.7)
	GTS_PDFA2 = "GTS_PDFA2"

	// GTS_PDFA3 represents PDF/A-3 conformance (ISO 19005-3:2012, based on PDF 1.7)
	// PDF/A-3 allows embedding of arbitrary file formats (required for ZUGFeRD)
	GTS_PDFA3 = "GTS_PDFA3"

	// GTS_PDFA4 represents PDF/A-4 conformance (ISO 19005-4:2020, based on PDF 2.0)
	GTS_PDFA4 = "GTS_PDFA4"
)

// Other OutputIntent Subtypes (non-PDF/A standards)
const (
	// GTS_PDFX represents PDF/X conformance (graphic arts exchange)
	GTS_PDFX = "GTS_PDFX"

	// GTS_PDFE1 represents PDF/E-1 conformance (engineering documents)
	GTS_PDFE1 = "GTS_PDFE1"

	// ISO_PDFE1 represents PDF/E-1 conformance (ISO variant)
	ISO_PDFE1 = "ISO_PDFE1"
)

// PDF/A Conformance Levels
const (
	// ConformanceA represents Level A (Accessible) - includes Level B plus structure/accessibility
	ConformanceA = "A"

	// ConformanceB represents Level B (Basic) - visual appearance preservation only
	ConformanceB = "B"

	// ConformanceU represents Level U (Unicode) - Level B plus reliable text extraction (PDF/A-2+)
	ConformanceU = "U"

	// ConformanceE represents Level E (Engineering) - PDF/A-4e only, adds 3D/RichMedia/JavaScript
	ConformanceE = "E"

	// ConformanceF represents Level F (File attachment) - PDF/A-4f only
	ConformanceF = "F"
)

// PDF/A subtypes recognized by pdfcpu
var pdfaSubtypes = []string{GTS_PDFA1, GTS_PDFA2, GTS_PDFA3, GTS_PDFA4}

// Valid OutputIntent subtypes (PDF/A and other standards)
var validOutputIntentSubtypes = []string{
	GTS_PDFA1, GTS_PDFA2, GTS_PDFA3, GTS_PDFA4,
	GTS_PDFX, GTS_PDFE1, ISO_PDFE1,
}

// Valid PDF/A conformance levels by part
var validConformanceLevels = map[int][]string{
	1: {ConformanceA, ConformanceB},                   // PDF/A-1: A, B
	2: {ConformanceA, ConformanceB, ConformanceU},     // PDF/A-2: A, B, U
	3: {ConformanceA, ConformanceB, ConformanceU},     // PDF/A-3: A, B, U
	4: {ConformanceE, ConformanceF},                   // PDF/A-4: E, F (base level has no letter)
}

// PDFAIdentification holds PDF/A identification from XMP metadata.
// This corresponds to the pdfaid namespace in XMP: http://www.aiim.org/pdfa/ns/id/
type PDFAIdentification struct {
	// Part represents the PDF/A version: 1, 2, 3, or 4
	Part int `xml:"http://www.aiim.org/pdfa/ns/id/ part"`

	// Conformance represents the conformance level: "A", "B", "U", "E", or "F"
	Conformance string `xml:"http://www.aiim.org/pdfa/ns/id/ conformance"`

	// Amendment represents an optional amendment identifier
	Amendment string `xml:"http://www.aiim.org/pdfa/ns/id/ amd,omitempty"`

	// Revision represents an optional revision year (e.g., 2005, 2008, 2012)
	Revision int `xml:"http://www.aiim.org/pdfa/ns/id/ rev,omitempty"`
}

// PDFAInfo holds comprehensive PDF/A identification information.
// This combines data from both XMP metadata and OutputIntent dictionaries.
type PDFAInfo struct {
	// From XMP metadata (pdfaid namespace)
	Part        *int    // PDF/A version: 1, 2, 3, or 4
	Conformance *string // Conformance level: "A", "B", "U", "E", or "F"
	Amendment   *string // Optional amendment identifier
	Revision    *int    // Optional revision year

	// From OutputIntent dictionary
	OutputIntentSubtype string // e.g., "GTS_PDFA3"

	// Validation and consistency flags
	ClaimsPDFA        bool // True if document claims PDF/A (via metadata or OutputIntent)
	MetadataPresent   bool // True if pdfaid metadata found in XMP
	OutputIntentFound bool // True if PDF/A OutputIntent found
	Consistent        bool // True if metadata matches OutputIntent
}

// IsPDFASubtype returns true if the given subtype is a PDF/A subtype.
//
// Example:
//   IsPDFASubtype("GTS_PDFA3") // returns true
//   IsPDFASubtype("GTS_PDFX")  // returns false
func IsPDFASubtype(s string) bool {
	for _, subtype := range pdfaSubtypes {
		if s == subtype {
			return true
		}
	}
	return false
}

// IsValidOutputIntentSubtype returns true if the given subtype is a known OutputIntent subtype.
// This includes PDF/A, PDF/X, and PDF/E standards.
//
// Note: This function is lenient and will return true for any non-empty string to allow
// for custom or future subtypes.
func IsValidOutputIntentSubtype(s string) bool {
	if s == "" {
		return false
	}

	for _, subtype := range validOutputIntentSubtypes {
		if s == subtype {
			return true
		}
	}

	// Be lenient - allow custom subtypes
	return true
}

// GetPartFromSubtype extracts the PDF/A part number from a PDF/A OutputIntent subtype.
// Returns nil if the subtype is not a PDF/A subtype.
//
// Example:
//   GetPartFromSubtype("GTS_PDFA3") // returns &3
//   GetPartFromSubtype("GTS_PDFX")  // returns nil
func GetPartFromSubtype(subtype string) *int {
	switch subtype {
	case GTS_PDFA1:
		part := 1
		return &part
	case GTS_PDFA2:
		part := 2
		return &part
	case GTS_PDFA3:
		part := 3
		return &part
	case GTS_PDFA4:
		part := 4
		return &part
	}
	return nil
}

// GetSubtypeFromPart returns the PDF/A OutputIntent subtype for a given part number.
// Returns empty string if part is not valid.
//
// Example:
//   GetSubtypeFromPart(3) // returns "GTS_PDFA3"
//   GetSubtypeFromPart(5) // returns ""
func GetSubtypeFromPart(part int) string {
	switch part {
	case 1:
		return GTS_PDFA1
	case 2:
		return GTS_PDFA2
	case 3:
		return GTS_PDFA3
	case 4:
		return GTS_PDFA4
	}
	return ""
}

// IsValidConformance checks if a conformance level is valid for a given PDF/A part.
//
// Example:
//   IsValidConformance(3, "B") // returns true
//   IsValidConformance(3, "E") // returns false (E is only for PDF/A-4)
//   IsValidConformance(1, "U") // returns false (U is only for PDF/A-2+)
func IsValidConformance(part int, conformance string) bool {
	validLevels, ok := validConformanceLevels[part]
	if !ok {
		return false
	}

	for _, level := range validLevels {
		if conformance == level {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of PDFAInfo.
func (info *PDFAInfo) String() string {
	if info == nil || !info.ClaimsPDFA {
		return "Not PDF/A"
	}

	var result string

	// Primary identification
	if info.Part != nil {
		result = fmt.Sprintf("PDF/A-%d", *info.Part)
		if info.Conformance != nil {
			result += fmt.Sprintf("%s", *info.Conformance)
		}
	} else if info.OutputIntentSubtype != "" {
		result = info.OutputIntentSubtype
	} else {
		result = "PDF/A (unknown version)"
	}

	// Additional details
	var details []string

	if info.MetadataPresent && info.OutputIntentFound {
		if info.Consistent {
			details = append(details, "consistent")
		} else {
			details = append(details, "inconsistent metadata/OutputIntent")
		}
	} else if info.MetadataPresent {
		details = append(details, "metadata only, no OutputIntent")
	} else if info.OutputIntentFound {
		details = append(details, "OutputIntent only, no metadata")
	}

	if len(details) > 0 {
		result += " (" + details[0] + ")"
	}

	return result
}

// NewPDFAInfo creates a new PDFAInfo instance.
func NewPDFAInfo() *PDFAInfo {
	return &PDFAInfo{
		ClaimsPDFA:        false,
		MetadataPresent:   false,
		OutputIntentFound: false,
		Consistent:        false,
	}
}

// SetFromMetadata populates PDFAInfo from XMP metadata.
func (info *PDFAInfo) SetFromMetadata(ident *PDFAIdentification) {
	if ident == nil {
		return
	}

	info.MetadataPresent = true
	info.ClaimsPDFA = true

	if ident.Part > 0 {
		info.Part = &ident.Part
	}

	if ident.Conformance != "" {
		info.Conformance = &ident.Conformance
	}

	if ident.Amendment != "" {
		info.Amendment = &ident.Amendment
	}

	if ident.Revision > 0 {
		info.Revision = &ident.Revision
	}
}

// SetFromOutputIntent populates PDFAInfo from OutputIntent dictionary.
func (info *PDFAInfo) SetFromOutputIntent(subtype string) {
	if !IsPDFASubtype(subtype) {
		return
	}

	info.OutputIntentFound = true
	info.ClaimsPDFA = true
	info.OutputIntentSubtype = subtype

	// If no metadata, infer part from OutputIntent
	if info.Part == nil {
		info.Part = GetPartFromSubtype(subtype)
	}
}

// CheckConsistency verifies that metadata and OutputIntent are consistent.
// Updates the Consistent flag.
func (info *PDFAInfo) CheckConsistency() {
	if !info.MetadataPresent || !info.OutputIntentFound {
		// Can't check consistency if we don't have both
		info.Consistent = false
		return
	}

	// Check if part from metadata matches part from OutputIntent
	metadataPart := info.Part
	outputIntentPart := GetPartFromSubtype(info.OutputIntentSubtype)

	if metadataPart != nil && outputIntentPart != nil {
		info.Consistent = (*metadataPart == *outputIntentPart)
	} else {
		info.Consistent = false
	}
}
