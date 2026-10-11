/*
Copyright 2020 The pdfcpu Authors.

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
	"maps"
	"reflect"
	"testing"
)

func checkContentResources(t *testing.T, content string, want map[string][]string) {
	t.Helper()
	for _, relaxed := range []bool{false, true} {
		checkContentResourcesMode(t, content, want, relaxed)
	}
}

func checkContentResourcesMode(t *testing.T, content string, want map[string][]string, relaxed bool) {
	t.Helper()
	got, err := parseContent(content, relaxed, false)
	if err != nil {
		t.Fatal(err)
	}
	expected := NewPageResourceNames()
	for kind, names := range want {
		for _, name := range names {
			expected[kind][name] = true
		}
	}
	for kind, names := range expected {
		if !maps.Equal(got[kind], names) {
			t.Errorf("%s resources = %v, want %v", kind, got[kind], names)
		}
	}
}

// TestContentResourceNameEscapes checks resource names are decoded consistently with dictionary keys.
func TestContentResourceNameEscapes(t *testing.T) {
	checkContentResources(t, `/Im#201 Do /F#31 12 Tf /CS#30 cs /GS#31 gs /P#31 scn /SH#31 sh /Span /MC#31 BDC`,
		map[string][]string{
			"XObject": {"Im 1"}, "Font": {"F1"}, "ColorSpace": {"CS0"}, "ExtGState": {"GS1"},
			"Pattern": {"P1"}, "Shading": {"SH1"}, "Properties": {"MC1"},
		})
	checkContentResources(t, `/Im#2f#23 Do /#44eviceRGB cs`, map[string][]string{"XObject": {"Im/#"}})
	checkContentResources(t, "BI /CS /CS#300 ID x EI Q", map[string][]string{"ColorSpace": {"CS00"}})
}

// TestInlineImageEndBoundaries checks EI does not depend on a following Q operator.
func TestInlineImageEndBoundaries(t *testing.T) {
	for _, tail := range []string{"", " Q", "\nBT /F1 12 Tf ET", "   /F1 12 Tf", "/F1 12 Tf", "% comment\n/F1 12 Tf"} {
		t.Run(tail, func(t *testing.T) {
			want := map[string][]string{}
			if len(tail) > 2 {
				want["Font"] = []string{"F1"}
			}
			checkContentResources(t, "BI /CS /G ID x EI"+tail, want)
		})
	}
	checkContentResources(t, "BI /CS /G ID x EI /F1 12 Tf BI /CS /G ID y EI Q /Im1 Do",
		map[string][]string{"Font": {"F1"}, "XObject": {"Im1"}})
	// A longer regular token is not an end marker.
	checkContentResources(t, "BI /CS /G ID x EIx /Fake Do x EI /Real Do",
		map[string][]string{"XObject": {"Real"}})
}

// TestInlineImageMissingEndWhitespace preserves compatibility with images in issue 1080.
func TestInlineImageMissingEndWhitespace(t *testing.T) {
	image := "BI /W 1 /H 1 /CS /RGB /BPC 8 /I true ID \xdb\xdb#EI"
	for _, tail := range []string{"", "\n", "\nQ /Im1 Do", "\nQ/Im1 Do"} {
		t.Run(tail, func(t *testing.T) {
			want := map[string][]string{}
			if len(tail) > 1 {
				want["XObject"] = []string{"Im1"}
			}
			checkContentResourcesMode(t, image+tail, want, true)
			if _, err := parseContent(image+tail, false, false); !errors.Is(err, errBIExpressionCorrupt) {
				t.Fatalf("strict parsing error = %v, want corrupt BI expression", err)
			}
		})
	}
	checkContentResourcesMode(t, image+"\nQ /F1 12 Tf BI /CS /G ID y EI Q /Im1 Do",
		map[string][]string{"Font": {"F1"}, "XObject": {"Im1"}}, true)
	checkContentResourcesMode(t, "BI /CS /G ID xEI Query /Fake Do x EI /Real Do",
		map[string][]string{"XObject": {"Real"}}, true)
}

// TestInlineImageASCII85EndMarker keeps EI-like encoded bytes inside the image until its EOD marker.
func TestInlineImageASCII85EndMarker(t *testing.T) {
	for _, filter := range []string{"/A85", "/ASCII85Decode", "[/A85 /Fl]", "[/ASCII85Decode /FlateDecode]", "/#4185"} {
		t.Run(filter, func(t *testing.T) {
			// Both EI] and EI) occur at line starts inside the qpdf-278-0.pdf image.
			content := "BI /W 4 /H 1 /BPC 8 /CS /RGB /F " + filter + " ID\r\nEI]ab\r\nEI)cd\r\nEI /X\r\n~>\r\nEI /Im1 Do"
			checkContentResources(t, content, map[string][]string{"XObject": {"Im1"}})
		})
	}
	checkContentResources(t, "BI /Filter /A85 ID z~> EI /F1 12 Tf BI /F /A85 ID z~> EI Q /Im1 Do",
		map[string][]string{"Font": {"F1"}, "XObject": {"Im1"}})
	// Only the first filter describes the encoding of bytes present in the content stream.
	checkContentResources(t, "BI /F [/Fl /A85] ID x EI /Im1 Do", map[string][]string{"XObject": {"Im1"}})
	content := "BI /F /A85 ID z~>EI Q /Im1 Do"
	checkContentResourcesMode(t, content, map[string][]string{"XObject": {"Im1"}}, true)
	if _, err := parseContent(content, false, false); !errors.Is(err, errBIExpressionCorrupt) {
		t.Fatalf("strict parsing error = %v, want corrupt BI expression", err)
	}
}

// TestInlineImageASCII85MalformedEnd rejects missing EOD markers and invalid following operators.
func TestInlineImageASCII85MalformedEnd(t *testing.T) {
	for _, content := range []string{
		"BI /F /A85 ID z EI Q", "BI /F /A85 ID z~>",
		"BI /F /A85 ID z~> Q EI Q", "BI /F /A85 ID z~> EIx Q",
		"BI /F", "BI /F [", "BI /F [123] ID z~> EI Q",
	} {
		t.Run(content, func(t *testing.T) {
			for _, relaxed := range []bool{false, true} {
				if _, err := parseContent(content, relaxed, false); !errors.Is(err, errBIExpressionCorrupt) {
					t.Errorf("relaxed=%t: error = %v, want corrupt BI expression", relaxed, err)
				}
			}
		})
	}
}

// TestInlineImagePDF20Length keeps EI-like image bytes out of the resource scan.
func TestInlineImagePDF20Length(t *testing.T) {
	for _, key := range []string{"L", "Length"} {
		for _, relaxed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/relaxed=%t", key, relaxed), func(t *testing.T) {
				content := "BI /W 13 /H 1 /BPC 8 /CS /G /" + key + " 13 ID x EI /Fake Do EI /Real Do"
				got, err := parseContent(content, relaxed, true)
				if err != nil {
					t.Fatal(err)
				}
				if !maps.Equal(got["XObject"], map[string]bool{"Real": true}) {
					t.Fatalf("XObject resources = %v, want Real only", got["XObject"])
				}
			})
		}
	}
}

// TestInlineImagePDF20LengthBoundaries checks ID separators and ASCII filter whitespace.
func TestInlineImagePDF20LengthBoundaries(t *testing.T) {
	for _, content := range []string{
		"BI /L 1 ID\r\nx EI /Real Do",
		"BI /L 2 ID  x EI /Real Do",
		"BI /L 3 /F /A85 ID \r\n z~> EI /Real Do",
		"BI /L 3 /Filter /ASCIIHexDecode ID \n 61> EI /Real Do",
	} {
		t.Run(content, func(t *testing.T) {
			got, err := parseContent(content, false, true)
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(got["XObject"], map[string]bool{"Real": true}) {
				t.Fatalf("XObject resources = %v, want Real only", got["XObject"])
			}
		})
	}
}

// TestInlineImagePDF20InvalidLength rejects missing or incorrect byte counts.
func TestInlineImagePDF20InvalidLength(t *testing.T) {
	for _, content := range []string{
		"BI /CS /G ID x EI /Real Do",
		"BI /L -1 ID x EI /Real Do",
		"BI /L x ID x EI /Real Do",
		"BI /L 1 /Length 2 ID x EI /Real Do",
		"BI /L 1 ID ab EI /Real Do",
		"BI /L 100 ID x EI /Real Do",
	} {
		t.Run(content, func(t *testing.T) {
			if _, err := parseContent(content, false, true); !errors.Is(err, errBIExpressionCorrupt) {
				t.Fatalf("error = %v, want corrupt BI expression", err)
			}
		})
	}
	// The Length entry was introduced in PDF 2.0; earlier files may omit it.
	if _, err := parseContent("BI /CS /G ID x EI /Real Do", false, false); err != nil {
		t.Fatal(err)
	}
}

// TestInlineImageColorSpaceArrays checks lookup strings are skipped and names are decoded.
func TestInlineImageColorSpaceArrays(t *testing.T) {
	for _, cs := range []string{
		`[/I /RGB 1 <000000FFFFFF>]`,
		`[/Indexed /DeviceRGB 1 (ab(cd))]`,
		`[/#49 /#52GB 1 (]/IDxx)]`,
	} {
		t.Run(cs, func(t *testing.T) {
			checkContentResources(t, "BI /CS "+cs+" ID x EI /Im1 Do", map[string][]string{"XObject": {"Im1"}})
		})
	}
	// Preserve a resource reference even in the tolerated nonstandard named-base form.
	checkContentResources(t, `BI /CS [/Indexed /CS#30 1 (]/IDxx)] ID x EI Q /Im1 Do`,
		map[string][]string{"ColorSpace": {"CS0"}, "XObject": {"Im1"}})
}

// TestContentNestedStringsAndDictionaries checks string contents never become operators or delimiters.
func TestContentNestedStringsAndDictionaries(t *testing.T) {
	for _, content := range []string{
		`(a(b)/Fake Do) Tj`,
		`[(a(b)/Fake Do) 10 (c)] TJ`,
		`(a\)b(c)/Fake Do) Tj`,
		`(a\\) Tj`,
		`/Span <</ActualText (<<)>> BDC`,
		`/Span <</ActualText (>> /Fake Do)>> BDC`,
		`/Span <</ActualText <3E3E>>> BDC`,
		`/Span <</Nested [<< /Text (a(b)<<>>) >>]>> BDC`,
		"/Span << % >> ( <<\n/ActualText (text)>> BDC",
	} {
		t.Run(content, func(t *testing.T) {
			checkContentResources(t, content+" /F1 12 Tf /Im1 Do",
				map[string][]string{"Font": {"F1"}, "XObject": {"Im1"}})
		})
	}
}

// TestContentMalformedExpressions checks incomplete strings, dictionaries, arrays, and name escapes return errors.
func TestContentMalformedExpressions(t *testing.T) {
	for _, content := range []string{
		`/Im# Do`, `/F#GG 12 Tf`, `/Im#00 Do`,
		`BI /CS /CS#G0 ID x EI Q`, `BI /CS [/I /CS#0 0 (x)] ID x EI Q`,
		`(a(b)`, `(abc\`, `/Span << /Text (a(b) >>`, `/Span << /Text <123`,
		`/Span << /Nested <<>>`, `/Span << /Text (x) >`,
		`BI /CS [`, `BI /CS [/I /RGB 0 (abc`, `BI /CS [/I /RGB 0 <abc`,
		`BI /CS [/I /RGB 0 (abc)`, `BI /CS [[] ID x EI Q`, `BI /CS /G ID x EIx Q`,
	} {
		t.Run(content, func(t *testing.T) {
			if _, err := parseContent(content, false, false); err == nil {
				t.Fatal("expected corrupt content error")
			}
		})
	}
}

// TestParseContent verifies parse content.
func TestParseContent(t *testing.T) {
	s := `/CS0 cs/DeviceGray CS/Span<</ActualText <FEFF000900090009>>>, Span<</ActualText<FEFF0009>>>, Span<</ActualText<FEFF0020>>>,
	Span<</ActualText<FEFF0020002E>>>, Span<</ActualText<FEFF002E>>>, Span<</ActualText<FEFF00090009>>> BDC
	/a1 BMC/a2 MP /a3 /MC0 BDC/P0 scn/RelativeColorimetric ri/P1 SCN/GS0 gs[(Q[i,j]/2.)16.6(The/]maxi\)-)]TJ/CS1 CS/a4<</A<FEFF>>> BDC /a5 <</A<FEFF>>>
	BDC (0.5*\(1/8\)*64 or +/4.\))Tj/T1_0 1 Tf <00150015> Tj /Im5 Do/a5 << /A <FEFF> >> BDC/a6/MC1 DP /a7<<>>DP
	BI /IM true/W 1/CS/CS2/H 1/BPC 1 ID EI Q /Pattern cs/Span<</ActualText<FEFF0009>>> BDC/SH1 sh`

	want := NewPageResourceNames()
	want["ColorSpace"]["CS0"] = true
	want["ColorSpace"]["CS1"] = true
	want["ColorSpace"]["CS2"] = true
	want["ExtGState"]["GS0"] = true
	want["Font"]["T1_0"] = true
	want["Pattern"]["P0"] = true
	want["Pattern"]["P1"] = true
	want["Properties"]["MC0"] = true
	want["Properties"]["MC1"] = true
	want["Shading"]["SH1"] = true
	want["XObject"]["Im5"] = true

	got, err := parseContent(s, false, false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}

// TestParseCorruptContent verifies safe handling of truncated content expressions.
func TestParseCorruptContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"TJ", "[(text)"},
		{"BI", "BI"},
		{"BIData", "BI ID"},
		{"BIColorSpace", "BI /CS "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseContent(tt.content, false, false); err == nil {
				t.Fatal("expected corrupt content error")
			}
		})
	}
}

// TestContentPDFWhitespace checks token and inline-image boundaries in both validation modes.
func TestContentPDFWhitespace(t *testing.T) {
	for _, ws := range []byte{0, 9, 10, 12, 13, 32} {
		t.Run(fmt.Sprintf("%02x", ws), func(t *testing.T) {
			sep := string(ws)
			checkContentResources(t, "q"+sep+"/Im0"+sep+"Do"+sep+"Q", map[string][]string{"XObject": {"Im0"}})
			checkContentResources(t, "BI"+sep+"/CS /G ID"+sep+"x"+sep+"EI"+sep+"Q /Im1 Do",
				map[string][]string{"XObject": {"Im1"}})
			checkContentResources(t, "BI /F /A85 ID z~>"+sep+"EI Q /Im1 Do",
				map[string][]string{"XObject": {"Im1"}})
		})
	}
}

// TestContentNonPDFWhitespace keeps Unicode spaces and non-PDF whitespace bytes within tokens and image data.
func TestContentNonPDFWhitespace(t *testing.T) {
	for _, suffix := range []string{"\x0b", "\x85", "\xa0", "\u0085", "\u00a0", "\u2003", "\u2028", "\u0100", "\xff"} {
		t.Run(fmt.Sprintf("%x", suffix), func(t *testing.T) {
			checkContentResources(t, "q /Im0 "+suffix+"Do Q q /Im1 Do Q", map[string][]string{"XObject": {"Im1"}})
			checkContentResources(t, "q BI"+suffix+"/Im1 Do Q", map[string][]string{"XObject": {"Im1"}})
			// Misleading EI sequences remain image bytes; resources inside them must not escape into content.
			checkContentResources(t, "BI /CS /G ID EI"+suffix+"Q/Fake Do\nEI Q /Real Do",
				map[string][]string{"XObject": {"Real"}})
			checkContentResources(t, "BI /CS /G ID EI"+suffix+"Q(\nEI Q", map[string][]string{})
			// The relaxed missing-separator exception still requires a complete Q token.
			checkContentResources(t, "BI /CS /G ID xEI Q"+suffix+"/Fake Do\nEI Q /Real Do",
				map[string][]string{"XObject": {"Real"}})
			for _, relaxed := range []bool{false, true} {
				if _, err := parseContent("BI /F /A85 ID z~>"+suffix+"EI Q", relaxed, false); !errors.Is(err, errBIExpressionCorrupt) {
					t.Errorf("relaxed=%t: got %v, want corrupt BI expression", relaxed, err)
				}
			}
		})
	}
}
