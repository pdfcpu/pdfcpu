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

import "testing"

func TestIsPDFASubtype(t *testing.T) {
	tests := []struct {
		subtype  string
		expected bool
	}{
		{"GTS_PDFA1", true},
		{"GTS_PDFA2", true},
		{"GTS_PDFA3", true},
		{"GTS_PDFA4", true},
		{"GTS_PDFX", false},
		{"GTS_PDFE1", false},
		{"ISO_PDFE1", false},
		{"CUSTOM_SUBTYPE", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.subtype, func(t *testing.T) {
			result := IsPDFASubtype(tt.subtype)
			if result != tt.expected {
				t.Errorf("IsPDFASubtype(%q) = %v, expected %v", tt.subtype, result, tt.expected)
			}
		})
	}
}

func TestIsValidOutputIntentSubtype(t *testing.T) {
	tests := []struct {
		subtype  string
		expected bool
	}{
		{"GTS_PDFA1", true},
		{"GTS_PDFA2", true},
		{"GTS_PDFA3", true},
		{"GTS_PDFA4", true},
		{"GTS_PDFX", true},
		{"GTS_PDFE1", true},
		{"ISO_PDFE1", true},
		{"CUSTOM_SUBTYPE", true}, // Lenient - allows custom
		{"", false},              // Empty not allowed
	}

	for _, tt := range tests {
		t.Run(tt.subtype, func(t *testing.T) {
			result := IsValidOutputIntentSubtype(tt.subtype)
			if result != tt.expected {
				t.Errorf("IsValidOutputIntentSubtype(%q) = %v, expected %v", tt.subtype, result, tt.expected)
			}
		})
	}
}

func TestGetPartFromSubtype(t *testing.T) {
	tests := []struct {
		subtype  string
		expected *int
	}{
		{"GTS_PDFA1", intPtr(1)},
		{"GTS_PDFA2", intPtr(2)},
		{"GTS_PDFA3", intPtr(3)},
		{"GTS_PDFA4", intPtr(4)},
		{"GTS_PDFX", nil},
		{"GTS_PDFE1", nil},
		{"INVALID", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.subtype, func(t *testing.T) {
			result := GetPartFromSubtype(tt.subtype)
			if !equalIntPtr(result, tt.expected) {
				t.Errorf("GetPartFromSubtype(%q) = %v, expected %v", tt.subtype, result, tt.expected)
			}
		})
	}
}

func TestGetSubtypeFromPart(t *testing.T) {
	tests := []struct {
		part     int
		expected string
	}{
		{1, "GTS_PDFA1"},
		{2, "GTS_PDFA2"},
		{3, "GTS_PDFA3"},
		{4, "GTS_PDFA4"},
		{0, ""},
		{5, ""},
		{-1, ""},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.part+'0')), func(t *testing.T) {
			result := GetSubtypeFromPart(tt.part)
			if result != tt.expected {
				t.Errorf("GetSubtypeFromPart(%d) = %q, expected %q", tt.part, result, tt.expected)
			}
		})
	}
}

func TestIsValidConformance(t *testing.T) {
	tests := []struct {
		part        int
		conformance string
		expected    bool
	}{
		// PDF/A-1
		{1, "A", true},
		{1, "B", true},
		{1, "U", false}, // U not valid for PDF/A-1
		{1, "E", false},
		{1, "F", false},

		// PDF/A-2
		{2, "A", true},
		{2, "B", true},
		{2, "U", true},
		{2, "E", false},
		{2, "F", false},

		// PDF/A-3
		{3, "A", true},
		{3, "B", true},
		{3, "U", true},
		{3, "E", false},
		{3, "F", false},

		// PDF/A-4
		{4, "A", false},
		{4, "B", false},
		{4, "U", false},
		{4, "E", true},
		{4, "F", true},

		// Invalid part
		{0, "A", false},
		{5, "B", false},
	}

	for _, tt := range tests {
		name := string(rune(tt.part+'0')) + tt.conformance
		t.Run(name, func(t *testing.T) {
			result := IsValidConformance(tt.part, tt.conformance)
			if result != tt.expected {
				t.Errorf("IsValidConformance(%d, %q) = %v, expected %v", tt.part, tt.conformance, result, tt.expected)
			}
		})
	}
}

func TestPDFAInfoString(t *testing.T) {
	tests := []struct {
		name     string
		info     *PDFAInfo
		expected string
	}{
		{
			name:     "nil info",
			info:     nil,
			expected: "Not PDF/A",
		},
		{
			name:     "not PDF/A",
			info:     &PDFAInfo{ClaimsPDFA: false},
			expected: "Not PDF/A",
		},
		{
			name: "PDF/A-3B full",
			info: &PDFAInfo{
				Part:              intPtr(3),
				Conformance:       strPtr("B"),
				OutputIntentSubtype: "GTS_PDFA3",
				ClaimsPDFA:        true,
				MetadataPresent:   true,
				OutputIntentFound: true,
				Consistent:        true,
			},
			expected: "PDF/A-3B (consistent)",
		},
		{
			name: "PDF/A-2U metadata only",
			info: &PDFAInfo{
				Part:              intPtr(2),
				Conformance:       strPtr("U"),
				ClaimsPDFA:        true,
				MetadataPresent:   true,
				OutputIntentFound: false,
				Consistent:        false,
			},
			expected: "PDF/A-2U (metadata only, no OutputIntent)",
		},
		{
			name: "PDF/A OutputIntent only",
			info: &PDFAInfo{
				OutputIntentSubtype: "GTS_PDFA1",
				ClaimsPDFA:        true,
				MetadataPresent:   false,
				OutputIntentFound: true,
				Consistent:        false,
			},
			expected: "GTS_PDFA1 (OutputIntent only, no metadata)",
		},
		{
			name: "PDF/A inconsistent",
			info: &PDFAInfo{
				Part:              intPtr(2),
				Conformance:       strPtr("B"),
				OutputIntentSubtype: "GTS_PDFA3",
				ClaimsPDFA:        true,
				MetadataPresent:   true,
				OutputIntentFound: true,
				Consistent:        false,
			},
			expected: "PDF/A-2B (inconsistent metadata/OutputIntent)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.String()
			if result != tt.expected {
				t.Errorf("PDFAInfo.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestPDFAInfoSetFromMetadata(t *testing.T) {
	info := NewPDFAInfo()

	// Test nil metadata
	info.SetFromMetadata(nil)
	if info.MetadataPresent || info.ClaimsPDFA {
		t.Error("SetFromMetadata(nil) should not set flags")
	}

	// Test valid metadata
	ident := &PDFAIdentification{
		Part:        3,
		Conformance: "B",
		Revision:    2012,
	}

	info.SetFromMetadata(ident)

	if !info.MetadataPresent {
		t.Error("MetadataPresent should be true")
	}
	if !info.ClaimsPDFA {
		t.Error("ClaimsPDFA should be true")
	}
	if info.Part == nil || *info.Part != 3 {
		t.Errorf("Part = %v, expected 3", info.Part)
	}
	if info.Conformance == nil || *info.Conformance != "B" {
		t.Errorf("Conformance = %v, expected B", info.Conformance)
	}
	if info.Revision == nil || *info.Revision != 2012 {
		t.Errorf("Revision = %v, expected 2012", info.Revision)
	}
}

func TestPDFAInfoSetFromOutputIntent(t *testing.T) {
	// Test non-PDF/A subtype
	info := NewPDFAInfo()
	info.SetFromOutputIntent("GTS_PDFX")
	if info.OutputIntentFound || info.ClaimsPDFA {
		t.Error("SetFromOutputIntent should not set flags for non-PDF/A subtype")
	}

	// Test PDF/A subtype
	info = NewPDFAInfo()
	info.SetFromOutputIntent("GTS_PDFA3")

	if !info.OutputIntentFound {
		t.Error("OutputIntentFound should be true")
	}
	if !info.ClaimsPDFA {
		t.Error("ClaimsPDFA should be true")
	}
	if info.OutputIntentSubtype != "GTS_PDFA3" {
		t.Errorf("OutputIntentSubtype = %q, expected GTS_PDFA3", info.OutputIntentSubtype)
	}
	if info.Part == nil || *info.Part != 3 {
		t.Errorf("Part = %v, expected 3 (inferred from OutputIntent)", info.Part)
	}
}

func TestPDFAInfoCheckConsistency(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *PDFAInfo
		expected bool
	}{
		{
			name: "consistent - matching metadata and OutputIntent",
			setup: func() *PDFAInfo {
				info := NewPDFAInfo()
				info.SetFromMetadata(&PDFAIdentification{Part: 3, Conformance: "B"})
				info.SetFromOutputIntent("GTS_PDFA3")
				return info
			},
			expected: true,
		},
		{
			name: "inconsistent - mismatched parts",
			setup: func() *PDFAInfo {
				info := NewPDFAInfo()
				info.SetFromMetadata(&PDFAIdentification{Part: 2, Conformance: "B"})
				info.SetFromOutputIntent("GTS_PDFA3")
				return info
			},
			expected: false,
		},
		{
			name: "inconsistent - metadata only",
			setup: func() *PDFAInfo {
				info := NewPDFAInfo()
				info.SetFromMetadata(&PDFAIdentification{Part: 3, Conformance: "B"})
				return info
			},
			expected: false,
		},
		{
			name: "inconsistent - OutputIntent only",
			setup: func() *PDFAInfo {
				info := NewPDFAInfo()
				info.SetFromOutputIntent("GTS_PDFA3")
				return info
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tt.setup()
			info.CheckConsistency()

			if info.Consistent != tt.expected {
				t.Errorf("Consistent = %v, expected %v", info.Consistent, tt.expected)
			}
		})
	}
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
