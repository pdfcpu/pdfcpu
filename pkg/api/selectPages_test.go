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

package api

import (
	"regexp"
	"testing"

	"strings"

	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

var r *regexp.Regexp

func doTestPageSelectionSyntaxOk(s string, t *testing.T) {

	_, err := ParsePageSelection(s)
	if err != nil {
		t.Errorf("doTestPageSelectionSyntaxOk(%s)\n", s)
	}
}

func doTestPageSelectionSyntaxFail(s string, t *testing.T) {

	_, err := ParsePageSelection(s)
	if err == nil {
		t.Errorf("doTestPageSelectionSyntaxFail(%s)\n", s)
	}
}

// Test the pageSelection string.
// This is used to select specific pages for extraction and trimming.
func TestPageSelectionSyntax(t *testing.T) {

	psOk := []string{"1", "!1", "n1", "1-", "!1-", "n1-", "-5", "!-5", "n-5", "3-5", "!3-5", "n3-5",
		"1,2,3", "!-5,10-15,30-", "1-,n4", "odd", "even", " 1"}

	for _, s := range psOk {
		doTestPageSelectionSyntaxOk(s, t)
	}

	psFail := []string{"1,", "1 ", "-", " -", " !"}

	for _, s := range psFail {
		doTestPageSelectionSyntaxFail(s, t)
	}

}

func selectedPagesString(sp pdfcpu.IntSet, pageCount int) string {

	s := []string{}
	var t string

	for i := 1; i <= pageCount; i++ {
		if sp[i] {
			t = "1"
		} else {
			t = "0"
		}
		s = append(s, t)
	}

	return strings.Join(s, "")
}

func doTestPageSelection(s string, pageCount int, compareString string, t *testing.T) {

	pageSelection, err := ParsePageSelection(s)
	if err != nil {
		t.Fatalf("TestPageSelection(%s) %v\n", s, err)
	}

	selectedPages, err := pagesForPageSelection(pageCount, pageSelection)
	if err != nil {
		t.Fatalf("TestPageSelection(%s) %v\n", s, err)
	}

	resultString := selectedPagesString(selectedPages, pageCount)

	if resultString != compareString {
		t.Fatalf("TestPageSelection(%s) expected:%s got%s\n", s, compareString, resultString)
	}

}

func TestPageSelection(t *testing.T) {

	pageCount := 5

	doTestPageSelection("even", pageCount, "01010", t)
	doTestPageSelection("even,even", pageCount, "01010", t)
	doTestPageSelection("odd", pageCount, "10101", t)
	doTestPageSelection("odd,odd", pageCount, "10101", t)
	doTestPageSelection("even,odd", pageCount, "11111", t)
	doTestPageSelection("odd,!1", pageCount, "00101", t)
	doTestPageSelection("odd,n1", pageCount, "00101", t)
	doTestPageSelection("!1,odd", pageCount, "00101", t)
	doTestPageSelection("n1,odd", pageCount, "00101", t)
	doTestPageSelection("!1,odd,even", pageCount, "01111", t)

	doTestPageSelection("1", pageCount, "10000", t)
	doTestPageSelection("2", pageCount, "01000", t)
	doTestPageSelection("3", pageCount, "00100", t)
	doTestPageSelection("4", pageCount, "00010", t)
	doTestPageSelection("5", pageCount, "00001", t)
	doTestPageSelection("6", pageCount, "00000", t)

	doTestPageSelection("-3", pageCount, "11100", t)
	doTestPageSelection("3-", pageCount, "00111", t)
	doTestPageSelection("2-4", pageCount, "01110", t)

	doTestPageSelection("-2,4-", pageCount, "11011", t)
	doTestPageSelection("2-4,!3", pageCount, "01010", t)
	doTestPageSelection("-4,n2", pageCount, "10110", t)

	doTestPageSelection("5-7", pageCount, "00001", t)
	doTestPageSelection("4-", pageCount, "00011", t)
	doTestPageSelection("5-", pageCount, "00001", t)
}
