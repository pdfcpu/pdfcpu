/*
Copyright 2018 The pdfcpu Authors.

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
	"testing"
)

func doTestParseDictOK(parseString string, t *testing.T) {
	_, err := parseObject(&parseString)
	if err != nil {
		t.Errorf("parseDict failed: <%v>\n", err)
		return
	}
}

func doTestParseDictFail(parseString string, t *testing.T) {
	s := parseString
	o, err := parseObject(&parseString)
	if err == nil {
		t.Errorf("parseDict should have returned an error for %s\n%v\n", s, o)
	}
}

func doTestParseDictGeneral(t *testing.T) {
	doTestParseDictOK("<</Type /Pages /Count 24 /Kids [6 0 R 16 0 R 21 0 R 27 0 R 30 0 R 32 0 R 34 0 R 36 0 R 38 0 R 40 0 R 42 0 R 44 0 R 46 0 R 48 0 R 50 0 R 52 0 R 54 0 R 56 0 R 58 0 R 60 0 R 62 0 R 64 0 R 69 0 R 71 0 R] /MediaBox [0 0 595.2756 841.8898]>>", t)
	doTestParseDictOK("<< /Key1 <abc> /Key2 <d> >>", t)
	doTestParseDictFail("<<", t)
	doTestParseDictFail("<<>", t)
	doTestParseDictOK("<<>>", t)
	doTestParseDictOK("<<     >>", t)
	doTestParseDictOK("<</Key1/Value1/key1/Value2>>", t)
	doTestParseDictOK("<</Type/Page/Parent 2 0 R/Resources<</Font<</F1 5 0 R/F2 7 0 R/F3 9 0 R>>/XObject<</Image11 11 0 R>>/ProcSet[/PDF/Text/ImageB/ImageC/ImageI]>>/MediaBox[ 0 0 595.32 841.92]/Contents 4 0 R/Group<</Type/Group/S/Transparency/CS/DeviceRGB>>/Tabs/S/StructParents 0>>", t)
}

func doTestParseDictNameObjects(t *testing.T) {
	// Name Objects
	doTestParseDictOK("<</Title \x0a/Type /Outline\x0a/Key /Value>>", t)
	doTestParseDictOK("<</Key1 /Value1\x0a/Title \x0a/Type /Outline\x0a/Key /Value>>", t)
	doTestParseDictOK("<</S/A>>", t) // empty name
	doTestParseDictOK("<</K1 / /K2 /Name2>>", t)
	doTestParseDictOK("<</Key/Value>>", t)
	doTestParseDictOK("<< /Key	/Value>>", t)
	doTestParseDictOK("<<	/Key/Value	>>", t)
	doTestParseDictOK("<<	/Key	/Value	>>", t)
	doTestParseDictOK("<</Key1/Value1/Key2/Value2>>", t)
}

func doTestParseDictStringLiteral(t *testing.T) {
	// String literals
	doTestParseDictOK("<</Key1(abc)/Key2(def)>>..", t)
	doTestParseDictOK("<</Key1(abc(inner1<<>>inner2)def)    >>..", t)
}

func doTestParseDictHexLiteral(t *testing.T) {
	// Hex literals
	doTestParseDictFail("<</Key<>>", t)
	doTestParseDictFail("<</Key<a4>>", t)
	doTestParseDictFail("<</Key<    >", t)
	doTestParseDictFail("<</Key<ade>", t)
	doTestParseDictFail("<</Key<ABG>>>", t)
	doTestParseDictFail("<</Key<   ABG>>>", t)
	doTestParseDictFail("<</Key<0ab><bcf098>", t)
	doTestParseDictOK("<</Key1<abc>/Key2<def>>>", t)
	doTestParseDictOK("<< /Key1 <abc> /Key2 <def> >>", t)
	doTestParseDictOK("<</Key1<AB>>>", t)
	doTestParseDictOK("<</Key1<ABC>>>", t)
	doTestParseDictOK("<</Key1<0ab>>>", t)
	doTestParseDictOK("<</Key<>>>", t)
	doTestParseDictOK("<< /Panose <01 05 02 02 03 00 00 00 00 00 00 00> >>", t)
	doTestParseDictOK("<< /Panose < 0 0 2 6 6 6 5 6 5 2 2 4> >>", t)
	doTestParseDictOK("<</Key <FEFF ABC2>>>", t)
}

func doTestParseDictDict(t *testing.T) {
	// Dictionaries
	doTestParseDictOK("<</Key<</Sub1 1/Sub2 2>>>>", t)
	doTestParseDictOK("<</Key<</Sub1(xyz)>>>>", t)
	doTestParseDictOK("<</Key<</Sub1[]>>>>", t)
	doTestParseDictOK("<</Key<</Sub1[1]>>>>", t)
	doTestParseDictOK("<</Key<</Sub1[(Go)]>>>>", t)
	doTestParseDictOK("<</Key<</Sub1[(Go)]/Sub2[(rocks!)]>>>>", t)
	doTestParseDictOK("<</A[/B1 /B2<</C 1>>]>>", t)
	doTestParseDictOK("<</A[/B1 /B2<</C 1>> /B3]>>", t)
	doTestParseDictOK("<</Name1[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]>>", t)
	doTestParseDictOK("<</A[/DictName<</A 123 /B<c0ff>>>]>>", t)
}

func doTestParseDictArray(t *testing.T) {
	// Arrays
	doTestParseDictOK("<</A[/B]>>", t)
	doTestParseDictOK("<</Key1[<abc><def>12.24 (gopher)]>>", t)
	doTestParseDictOK("<</Key1[<abc><def>12.24 (gopher)] /Key2[(abc)2.34[<c012>2 0 R]]>>", t)
	doTestParseDictOK("<</Key1[1 2 3 [4]]>>", t)
	doTestParseDictOK("<</K[<</Obj 71 0 R/Type/OBJR>>269 0 R]/P 258 0 R/S/Link/Pg 19 0 R>>", t)
}

func doTestParseDictBool(t *testing.T) {
	// null, true, false
	doTestParseDictOK("<</Key1 true>>", t)
	doTestParseDictOK("<</Key1 			false>>", t)
	doTestParseDictOK("<</Key1 null /Key2 true /Key3 false>>", t)
	doTestParseDictFail("<</Key1 TRUE>>", t)
}

func doTestParseDictNumerics(t *testing.T) {
	// Numerics
	doTestParseDictOK("<</Key1 16>>", t)
	doTestParseDictOK("<</Key1 .034>>", t)
	doTestParseDictFail("<</Key1 ,034>>", t)
}

func doTestParseDictIndirectRefs(t *testing.T) {
	// Indirect object references
	doTestParseDictOK("<</Key1 32 0 R>>", t)
	doTestParseDictOK("<</Key1 32 0 R/Key2 32 /Key3 3.34>>", t)
}

func doTestParseDictWithComments(t *testing.T) {

	doTestParseDictOK(`<</Root 1 0 R/Info%comment after name
<</Subject(Compacted Syntax v3.0)%comment after literal string end
/Title<436f6d7061637465642073796e746178>%comment after hex string end
/Keywords(PDF,Compacted,Syntax,ISO 32000-2:2020)/CreationDate(D:20200317)/Author(Peter Wyatt)/Creator<48616e642d65646974>/Producer<48616e642d65646974>>>/ID[<18D6B641245C03FABE67D93AD879D6EC><6264992C92074533A46A019C7CF9BFB6>]/Size 7>>`, t)

	doTestParseDictOK(`<</Type/Page/Parent 3 0 R/MediaBox[%comment after array start token
+0 .0 999 999.]%comment after array end token
/CropBox[+0 .0 999%comment after an integer
999.]/Contents[5 0 R]/UserUnit +0.88/Annots null%comment after null
/Resources<</Pattern<<>>/ProcSet[null]/ExtGState<</ 6 0 R>>/Font<</F1<</Type/Font/Subtype/Type1/BaseFont/Times-Bold/Encoding/WinAnsiEncoding>>>>>>>>`, t)

}

func TestParseDict(t *testing.T) {
	doTestParseDictGeneral(t)
	doTestParseDictNameObjects(t)
	doTestParseDictStringLiteral(t)
	doTestParseDictHexLiteral(t)
	doTestParseDictDict(t)
	doTestParseDictArray(t)
	doTestParseDictBool(t)
	doTestParseDictNumerics(t)
	doTestParseDictIndirectRefs(t)
	doTestParseDictWithComments(t)
}
