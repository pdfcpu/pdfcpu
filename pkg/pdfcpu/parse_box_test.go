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

import "testing"

func doTestParseBoxListOK(s string, t *testing.T) {
	t.Helper()
	_, err := ParseBoxList(s)
	if err != nil {
		t.Errorf("parseBoxList failed: <%v> <%s>\n", err, s)
		return
	}
}

func doTestParseBoxListFail(s string, t *testing.T) {
	t.Helper()
	_, err := ParseBoxList(s)
	if err == nil {
		t.Errorf("parseBoxList should have failed: <%s>\n", s)
		return
	}
}

func TestParseBoxList(t *testing.T) {
	doTestParseBoxListOK("", t)
	doTestParseBoxListOK("m ", t)
	doTestParseBoxListOK("media,  crop", t)
	doTestParseBoxListOK("m, c, t, b, a", t)
	doTestParseBoxListOK("c,t,b,a,m", t)
	doTestParseBoxListOK("media,crop,bleed,trim,art", t)

	doTestParseBoxListFail("crap", t)
	doTestParseBoxListFail("c t b a ", t)
	doTestParseBoxListFail("media;crop;bleed;trim;art", t)

}

func doTestParseBoxOK(s string, t *testing.T) {
	t.Helper()
	_, err := ParseBox(s, POINTS)
	if err != nil {
		t.Errorf("parseBox failed: <%v> <%s>\n", err, s)
		return
	}
}

func doTestParseBoxFail(s string, t *testing.T) {
	t.Helper()
	_, err := ParseBox(s, POINTS)
	if err == nil {
		t.Errorf("parseBox should have failed: <%s>\n", s)
		return
	}
}

func TestParseBox(t *testing.T) {

	// Box by rectangle.
	doTestParseBoxOK("[0 0 200 400]", t)
	doTestParseBoxOK("[200 400 0 0]", t)
	doTestParseBoxOK("[-50 -50 200 400]", t)
	doTestParseBoxOK("[2.5 2.5 200 400]", t)
	doTestParseBoxFail("[2.5 200 400]", t)
	doTestParseBoxFail("[2.5 200 400 500 600]", t)
	doTestParseBoxFail("[-50 -50 200 x]", t)

	// Box by 1 margin value.
	doTestParseBoxOK("10.5%", t)
	doTestParseBoxOK("-10.5%", t)
	doTestParseBoxOK("10", t)
	doTestParseBoxOK("-10", t)
	doTestParseBoxOK("10 abs", t)
	doTestParseBoxOK(".5", t)
	doTestParseBoxOK(".5 abs", t)
	doTestParseBoxOK(".4 rel", t)
	doTestParseBoxFail("50%", t)
	doTestParseBoxFail("0.6 rel", t)

	// Box by 2 margin values.
	doTestParseBoxOK("10% -40%", t)
	doTestParseBoxOK("10 5", t)
	doTestParseBoxOK("10 5 abs", t)
	doTestParseBoxOK(".1 .5", t)
	doTestParseBoxOK(".1 .5 abs", t)
	doTestParseBoxOK(".1 .4 rel", t)
	doTestParseBoxFail("10% 40", t)
	doTestParseBoxFail(".5 .5 rel", t)

	// Box by 3 margin values.
	doTestParseBoxOK("10% 15.5% 10%", t)
	doTestParseBoxOK("10 5 15", t)
	doTestParseBoxOK("10 5 15 abs", t)
	doTestParseBoxOK(".1 .155 .1", t)
	doTestParseBoxOK(".1 .155 .1 abs", t)
	doTestParseBoxOK(".1 .155 .1 rel", t)
	doTestParseBoxOK(".1 .155 .6 rel", t)
	doTestParseBoxFail("10% 15.5 10%", t)
	doTestParseBoxFail(".1 .155 r .1 .1", t)
	doTestParseBoxFail(".1 .155 rel .1", t)

	// Box by 4 margin values.
	doTestParseBoxOK("40% 40% 10% 10%", t)
	doTestParseBoxOK("0.4 0.4 20 20", t)
	doTestParseBoxOK("0.4 0.4 .1 .1", t)
	doTestParseBoxOK("0.4 0.4 .1 .1 abs", t)
	doTestParseBoxOK("0.4 0.4 .1 .1 rel", t)
	doTestParseBoxOK("10% 20% 60% 70%", t)
	doTestParseBoxOK("-40% 40% 10% 10%", t)
	doTestParseBoxFail("40% 40% 70% 0%", t)
	doTestParseBoxFail("40% 40% 100 100", t)

	// Box by arbitrary relative position within parent box.
	doTestParseBoxOK("dim:30 30", t)
	doTestParseBoxOK("dim:30 30 abs", t)
	doTestParseBoxOK("dim:.3 .3 rel", t)
	doTestParseBoxOK("dim:30% 30%", t)
	doTestParseBoxOK("pos:tl, dim:30 30", t)
	doTestParseBoxOK("pos:bl, off: 5 5, dim:30 30", t)
	doTestParseBoxOK("pos:bl, off: -5 -5, dim:.3 .3 rel", t)
	doTestParseBoxFail("pos:tl", t)
	doTestParseBoxFail("off:.23 .5", t)
}

func doTestParsePageBoundariesOK(s string, t *testing.T) {
	t.Helper()
	_, err := ParsePageBoundaries(s, POINTS)
	if err != nil {
		t.Errorf("parsePageBoundaries failed: <%v> <%s>\n", err, s)
		return
	}
}

func doTestParsePageBoundariesFail(s string, t *testing.T) {
	t.Helper()
	_, err := ParsePageBoundaries(s, POINTS)
	if err == nil {
		t.Errorf("parsePageBoundaries should have failed: <%s>\n", s)
		return
	}
}

func TestParsePageBoundaries(t *testing.T) {
	doTestParsePageBoundariesOK("trim:10", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200], crop:10 20, trim:crop, art:bleed, bleed:art", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200], art:bleed, bleed:art", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200], art:bleed, trim:art", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200], art:bleed, trim:bleed", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200], trim:[30 30 170 170], art:bleed", t)
	doTestParsePageBoundariesOK("media:[0 0 200 200]", t)
	doTestParsePageBoundariesOK("media:10", t)
	doTestParsePageBoundariesFail("media:trim", t)
}
