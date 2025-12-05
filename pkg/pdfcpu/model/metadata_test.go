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

import (
	"encoding/xml"
	"testing"
)

func TestGetPDFAIdentification(t *testing.T) {
	tests := []struct {
		name        string
		desc        Description
		expectNil   bool
		expectedPart        int
		expectedConformance string
	}{
		{
			name: "no PDF/A info",
			desc: Description{
				Producer: "pdfcpu",
			},
			expectNil: true,
		},
		{
			name: "PDF/A-3B",
			desc: Description{
				PDFAPart:        3,
				PDFAConformance: "B",
			},
			expectNil:           false,
			expectedPart:        3,
			expectedConformance: "B",
		},
		{
			name: "PDF/A-2U with revision",
			desc: Description{
				PDFAPart:        2,
				PDFAConformance: "U",
				PDFARevision:    2011,
			},
			expectNil:           false,
			expectedPart:        2,
			expectedConformance: "U",
		},
		{
			name: "only conformance (invalid, but handled)",
			desc: Description{
				PDFAConformance: "A",
			},
			expectNil:           false,
			expectedPart:        0,
			expectedConformance: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.desc.GetPDFAIdentification()

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Part != tt.expectedPart {
				t.Errorf("Part = %d, expected %d", result.Part, tt.expectedPart)
			}

			if result.Conformance != tt.expectedConformance {
				t.Errorf("Conformance = %q, expected %q", result.Conformance, tt.expectedConformance)
			}
		})
	}
}

func TestXMPMetadataParsing_PDFA(t *testing.T) {
	// Test parsing XMP metadata with PDF/A information
	xmpData := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/"
        xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
        xmlns:xap="http://ns.adobe.com/xap/1.0/">
      <pdfaid:part>3</pdfaid:part>
      <pdfaid:conformance>B</pdfaid:conformance>
      <pdfaid:rev>2012</pdfaid:rev>
      <pdf:Producer>pdfcpu</pdf:Producer>
      <xap:CreatorTool>Test Tool</xap:CreatorTool>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

	var xmp XMPMeta
	err := xml.Unmarshal([]byte(xmpData), &xmp)
	if err != nil {
		t.Fatalf("Failed to parse XMP: %v", err)
	}

	// Check that PDF/A fields were parsed
	desc := xmp.RDF.Description

	if desc.PDFAPart != 3 {
		t.Errorf("PDFAPart = %d, expected 3", desc.PDFAPart)
	}

	if desc.PDFAConformance != "B" {
		t.Errorf("PDFAConformance = %q, expected 'B'", desc.PDFAConformance)
	}

	if desc.PDFARevision != 2012 {
		t.Errorf("PDFARevision = %d, expected 2012", desc.PDFARevision)
	}

	// Check GetPDFAIdentification
	ident := desc.GetPDFAIdentification()
	if ident == nil {
		t.Fatal("GetPDFAIdentification returned nil")
	}

	if ident.Part != 3 {
		t.Errorf("ident.Part = %d, expected 3", ident.Part)
	}

	if ident.Conformance != "B" {
		t.Errorf("ident.Conformance = %q, expected 'B'", ident.Conformance)
	}

	if ident.Revision != 2012 {
		t.Errorf("ident.Revision = %d, expected 2012", ident.Revision)
	}
}

func TestXMPMetadataParsing_NoPDFA(t *testing.T) {
	// Test parsing XMP metadata without PDF/A information
	xmpData := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
        xmlns:xap="http://ns.adobe.com/xap/1.0/">
      <pdf:Producer>pdfcpu</pdf:Producer>
      <xap:CreatorTool>Test Tool</xap:CreatorTool>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

	var xmp XMPMeta
	err := xml.Unmarshal([]byte(xmpData), &xmp)
	if err != nil {
		t.Fatalf("Failed to parse XMP: %v", err)
	}

	desc := xmp.RDF.Description

	// Check that PDF/A fields are zero values
	if desc.PDFAPart != 0 {
		t.Errorf("PDFAPart = %d, expected 0", desc.PDFAPart)
	}

	if desc.PDFAConformance != "" {
		t.Errorf("PDFAConformance = %q, expected empty", desc.PDFAConformance)
	}

	// GetPDFAIdentification should return nil
	ident := desc.GetPDFAIdentification()
	if ident != nil {
		t.Errorf("GetPDFAIdentification should return nil for non-PDF/A metadata, got %+v", ident)
	}
}
