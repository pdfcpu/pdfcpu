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

package pdfcpu

import (
	"reflect"
	"testing"
)

func TestParseContent(t *testing.T) {
	s := `/CS0 cs/DeviceGray CS/Span<</ActualText <FEFF000900090009>>>, Span<</ActualText<FEFF0009>>>, Span<</ActualText<FEFF0020>>>,
	Span<</ActualText<FEFF0020002E>>>, Span<</ActualText<FEFF002E>>>, Span<</ActualText<FEFF00090009>>> BDC
	/a1 BMC/a2 MP /a3 /MC0 BDC/P0 scn/RelativeColorimetric ri/P1 SCN/GS0 gs[(Q[i,j]/2.)16.6(The/]maxi\)-)]TJ/CS1 CS/a4<</A<FEFF>>> BDC /a5 <</A<FEFF>>>
	BDC (0.5*\(1/8\)*64 or +/4.\))Tj/T1_0 1 Tf <00150015> Tj /Im5 Do/a5 << /A <FEFF> >> BDC/a6/MC1 DP /a7<<>>DP
	BI /IM true/W 1/CS/InlineCS/H 1/BPC 1 ID EI Q /Pattern cs/Span<</ActualText<FEFF0009>>> BDC/SH1 sh`

	want := NewPageResourceNames()
	want["ColorSpace"]["CS0"] = true
	want["ColorSpace"]["CS1"] = true
	want["ColorSpace"]["InlineCS"] = true
	want["ExtGState"]["GS0"] = true
	want["Font"]["T1_0"] = true
	want["Pattern"]["P0"] = true
	want["Pattern"]["P1"] = true
	want["Properties"]["MC0"] = true
	want["Properties"]["MC1"] = true
	want["Shading"]["SH1"] = true
	want["XObject"]["Im5"] = true

	got, err := parseContent(s)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}
