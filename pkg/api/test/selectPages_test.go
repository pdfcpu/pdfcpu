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

package test

import (
	"fmt"
	"regexp"
	"testing"

	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var r *regexp.Regexp

func testPageSelectionSyntaxOk(t *testing.T, s string) {
	t.Helper()
	_, err := api.ParsePageSelection(s)
	if err != nil {
		t.Errorf("doTestPageSelectionSyntaxOk(%s)\n", s)
	}
}

func testPageSelectionSyntaxFail(t *testing.T, s string) {
	t.Helper()
	_, err := api.ParsePageSelection(s)
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
		testPageSelectionSyntaxOk(t, s)
	}

	psFail := []string{"1,", "1 ", "-", " -", " !"}

	for _, s := range psFail {
		testPageSelectionSyntaxFail(t, s)
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

func testSelectedPages(s string, pageCount int, compareString string, t *testing.T) {
	pageSelection, err := api.ParsePageSelection(s)
	if err != nil {
		t.Fatalf("testSelectedPages(%s) %v\n", s, err)
	}

	selectedPages, err := api.PagesForPageSelection(pageCount, pageSelection, false)
	if err != nil {
		t.Fatalf("testSelectedPages(%s) %v\n", s, err)
	}

	resultString := selectedPagesString(selectedPages, pageCount)

	if resultString != compareString {
		t.Fatalf("testSelectedPages(%s) expected:%s got%s\n", s, compareString, resultString)
	}
}

func TestSelectedPages(t *testing.T) {
	pageCount := 5

	testSelectedPages("even", pageCount, "01010", t)
	testSelectedPages("even,even", pageCount, "01010", t)
	testSelectedPages("odd", pageCount, "10101", t)
	testSelectedPages("odd,odd", pageCount, "10101", t)
	testSelectedPages("even,odd", pageCount, "11111", t)
	testSelectedPages("odd,!1", pageCount, "00101", t)
	testSelectedPages("odd,n1", pageCount, "00101", t)
	testSelectedPages("!1,odd", pageCount, "00101", t)
	testSelectedPages("n1,odd", pageCount, "00101", t)
	testSelectedPages("!1,odd,even", pageCount, "01111", t)

	testSelectedPages("1", pageCount, "10000", t)
	testSelectedPages("2", pageCount, "01000", t)
	testSelectedPages("3", pageCount, "00100", t)
	testSelectedPages("4", pageCount, "00010", t)
	testSelectedPages("5", pageCount, "00001", t)
	testSelectedPages("6", pageCount, "00000", t)

	testSelectedPages("-3", pageCount, "11100", t)
	testSelectedPages("3-", pageCount, "00111", t)
	testSelectedPages("2-4", pageCount, "01110", t)

	testSelectedPages("-2,4-", pageCount, "11011", t)
	testSelectedPages("2-4,!3", pageCount, "01010", t)
	testSelectedPages("-4,n2", pageCount, "10110", t)

	testSelectedPages("5-7", pageCount, "00001", t)
	testSelectedPages("4-", pageCount, "00011", t)
	testSelectedPages("5-", pageCount, "00001", t)
	testSelectedPages("!4", pageCount, "00000", t)

	testSelectedPages("-l", pageCount, "11111", t)
	testSelectedPages("-l-1", pageCount, "11110", t)
	testSelectedPages("2-l", pageCount, "01111", t)
	testSelectedPages("2-l-2", pageCount, "01100", t)
	testSelectedPages("2-l-3", pageCount, "01000", t)
	testSelectedPages("2-l-4", pageCount, "00000", t)
	testSelectedPages("!l", pageCount, "00000", t)
	testSelectedPages("nl", pageCount, "00000", t)
	testSelectedPages("!l-2", pageCount, "00000", t)
	testSelectedPages("nl-2", pageCount, "00000", t)
	testSelectedPages("l", pageCount, "00001", t)
	testSelectedPages("l-1", pageCount, "00010", t)
	testSelectedPages("l-1-", pageCount, "00011", t)
	testSelectedPages("!l,odd", pageCount, "10100", t)
	testSelectedPages("l,even", pageCount, "01011", t)

	testSelectedPages("1-l,!2-l-1", pageCount, "10001", t)
}

func collectedPagesString(cp []int, pageCount int) string {
	return fmt.Sprint(cp)
}

func testCollectedPages(s string, pageCount int, want string, t *testing.T) {
	pageSelection, err := api.ParsePageSelection(s)
	if err != nil {
		t.Fatalf("testCollectedPages(%s) %v\n", s, err)
	}

	collectedPages, err := api.PagesForPageCollection(pageCount, pageSelection)
	if err != nil {
		t.Fatalf("testCollectedPages(%s) %v\n", s, err)
	}

	got := collectedPagesString(collectedPages, pageCount)
	//fmt.Printf("%s\n", resultString)

	if got != want {
		t.Fatalf("testCollectedPages(%s) want:%s got%s\n", s, want, got)
	}
}

func TestCollectedPages(t *testing.T) {
	pageCount := 5

	testCollectedPages("even", pageCount, "[2 4]", t)
	testCollectedPages("even,even", pageCount, "[2 4 2 4]", t)
	testCollectedPages("odd", pageCount, "[1 3 5]", t)
	testCollectedPages("odd,odd", pageCount, "[1 3 5 1 3 5]", t)
	testCollectedPages("even,odd", pageCount, "[2 4 1 3 5]", t)
	testCollectedPages("odd,!1", pageCount, "[3 5]", t)
	testCollectedPages("odd,n1", pageCount, "[3 5]", t)
	testCollectedPages("!1,odd", pageCount, "[1 3 5]", t)
	testCollectedPages("n1,odd", pageCount, "[1 3 5]", t)
	testCollectedPages("!1,odd,even", pageCount, "[1 3 5 2 4]", t)

	testCollectedPages("1", pageCount, "[1]", t)
	testCollectedPages("2", pageCount, "[2]", t)
	testCollectedPages("3", pageCount, "[3]", t)
	testCollectedPages("4", pageCount, "[4]", t)
	testCollectedPages("5", pageCount, "[5]", t)
	testCollectedPages("6", pageCount, "[]", t)

	testCollectedPages("-3", pageCount, "[1 2 3]", t)
	testCollectedPages("3-", pageCount, "[3 4 5]", t)
	testCollectedPages("2-4", pageCount, "[2 3 4]", t)

	testCollectedPages("-2,4-", pageCount, "[1 2 4 5]", t)
	testCollectedPages("2-4,!3", pageCount, "[2 4]", t)
	testCollectedPages("-4,n2", pageCount, "[1 3 4]", t)

	testCollectedPages("5-7", pageCount, "[5]", t)
	testCollectedPages("4-", pageCount, "[4 5]", t)
	testCollectedPages("5-", pageCount, "[5]", t)
	testCollectedPages("!4", pageCount, "[]", t)

	testCollectedPages("-l", pageCount, "[1 2 3 4 5]", t)
	testCollectedPages("-l-1", pageCount, "[1 2 3 4]", t)
	testCollectedPages("2-l", pageCount, "[2 3 4 5]", t)
	testCollectedPages("2-l-2", pageCount, "[2 3]", t)
	testCollectedPages("2-l-3", pageCount, "[2]", t)
	testCollectedPages("2-l-4", pageCount, "[]", t)
	testCollectedPages("!l", pageCount, "[]", t)
	testCollectedPages("nl", pageCount, "[]", t)
	testCollectedPages("!l-2", pageCount, "[]", t)
	testCollectedPages("nl-2", pageCount, "[]", t)
	testCollectedPages("l", pageCount, "[5]", t)
	testCollectedPages("l-1", pageCount, "[4]", t)
	testCollectedPages("l-1-", pageCount, "[4 5]", t)
	testCollectedPages("!l,odd", pageCount, "[1 3 5]", t)
	testCollectedPages("l,even", pageCount, "[5 2 4]", t)

	testCollectedPages("1-3,2,1,l", pageCount, "[1 2 3 2 1 5]", t)
	testCollectedPages("1,1,1,l,l,l", pageCount, "[1 1 1 5 5 5]", t)
	testCollectedPages("1-3,2-4,3-", pageCount, "[1 2 3 2 3 4 3 4 5]", t)
	testCollectedPages("1-3,2-4,!3", pageCount, "[1 2 2 4]", t)

	testCollectedPages("1-,!l", pageCount, "[1 2 3 4]", t)
	testCollectedPages("1-,nl", pageCount, "[1 2 3 4]", t)
	testSelectedPages("1-l,!2-l-1", pageCount, "10001", t)

}
