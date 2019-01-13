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
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/log"
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

var (
	selectedPagesRegExp *regexp.Regexp
)

func setupRegExpForPageSelection() *regexp.Regexp {

	e := "[!n]?((-\\d+)|(\\d+(-(\\d+)?)?))"

	e = "\\Qeven\\E|\\Qodd\\E|" + e

	exp := "^" + e + "(," + e + ")*$"

	re, _ := regexp.Compile(exp)

	return re
}

func init() {

	selectedPagesRegExp = setupRegExpForPageSelection()
}

// ParsePageSelection ensures a correct page selection expression.
func ParsePageSelection(s string) ([]string, error) {

	if s == "" {
		return nil, nil
	}

	// Ensure valid comma separated expression of:{ {even|odd}{!}{-}# | {even|odd}{!}#-{#} }*
	//
	// Negated expressions:
	// '!' negates an expression
	// since '!' needs to be part of a single quoted string in bash
	// as an alternative also 'n' works instead of "!"
	//
	// Extract all but page 4 may be expressed as: "1-,!4" or "1-,n4"
	//
	// The pageSelection is evaluated strictly from left to right!
	// e.g. "!3,1-5" extracts pages 1-5 whereas "1-5,!3" extracts pages 1,2,4,5
	//

	if !selectedPagesRegExp.MatchString(s) {
		return nil, errors.Errorf("-pages \"%s\" => syntax error\n", s)
	}

	log.API.Printf("pageSelection: %s\n", s)

	return strings.Split(s, ","), nil
}

func handlePrefix(v string, negated bool, pageCount int, selectedPages pdf.IntSet) error {

	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		i = pageCount
	}

	// identified
	// -# ... select all pages up to and including #
	// or !-# ... deselect all pages up to and including #
	for j := 1; j <= i; j++ {
		selectedPages[j] = !negated
	}

	return nil
}

func handleSuffix(v string, negated bool, pageCount int, selectedPages pdf.IntSet) error {

	// must be #- ... select all pages from here until the end.
	// or !#- ... deselect all pages from here until the end.

	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		return nil
	}

	for j := i; j <= pageCount; j++ {
		selectedPages[j] = !negated
	}

	return nil
}

func handleSpecificPage(s string, negated bool, pageCount int, selectedPages pdf.IntSet) error {

	// must be # ... select a specific page
	// or !# ... deselect a specific page

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		return nil
	}

	selectedPages[i] = !negated

	return nil
}

func negation(c byte) bool {
	return c == '!' || c == 'n'
}

func selectEvenPages(selectedPages pdf.IntSet, pageCount int) {
	for i := 2; i <= pageCount; i += 2 {
		_, found := selectedPages[i]
		if !found {
			selectedPages[i] = true
		}
	}
}

func selectOddPages(selectedPages pdf.IntSet, pageCount int) {
	for i := 1; i <= pageCount; i += 2 {
		_, found := selectedPages[i]
		if !found {
			selectedPages[i] = true
		}
	}
}

func parsePageRange(pr []string, pageCount int, negated bool, selectedPages pdf.IntSet) error {

	from, err := strconv.Atoi(pr[0])
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if from > pageCount {
		return nil
	}

	thru, err := strconv.Atoi(pr[1])
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if thru < from {
		return nil
	}

	if thru > pageCount {
		thru = pageCount
	}

	for i := from; i <= thru; i++ {
		selectedPages[i] = !negated
	}

	return nil
}

func selectedPages(pageCount int, pageSelection []string) (pdf.IntSet, error) {

	selectedPages := pdf.IntSet{}

	for _, v := range pageSelection {

		//log.Stats.Printf("pageExp: <%s>\n", v)

		// Special case "even" only for len(pageSelection) == 1
		if v == "even" {
			selectEvenPages(selectedPages, pageCount)
			continue
		}

		// Special case "odd" only for len(pageSelection) == 1
		if v == "odd" {
			selectOddPages(selectedPages, pageCount)
			continue
		}

		var negated bool
		if negation(v[0]) {
			negated = true
			//logInfoAPI.Printf("is a negated exp\n")
			v = v[1:]
		}

		if v[0] == '-' {

			v = v[1:]

			err := handlePrefix(v, negated, pageCount, selectedPages)
			if err != nil {
				return nil, err
			}

			continue
		}

		if strings.HasSuffix(v, "-") {

			err := handleSuffix(v[:len(v)-1], negated, pageCount, selectedPages)
			if err != nil {
				return nil, err
			}

			continue
		}

		// if v contains '-' somewhere in the middle
		// this must be #-# ... select a page range
		// or !#-# ... deselect a page range

		pr := strings.Split(v, "-")
		if len(pr) == 2 {

			err := parsePageRange(pr, pageCount, negated, selectedPages)
			if err != nil {
				return nil, err
			}

			continue
		}

		err := handleSpecificPage(pr[0], negated, pageCount, selectedPages)
		if err != nil {
			return nil, err
		}

	}

	return selectedPages, nil
}

func pagesForPageSelection(pageCount int, pageSelection []string) (pdf.IntSet, error) {

	if pageSelection == nil || len(pageSelection) == 0 {
		log.Info.Println("pagesForPageSelection: empty pageSelection")
		return nil, nil
	}

	return selectedPages(pageCount, pageSelection)
}

// Split, Extract, Stamp, Watermark, Rotate: No page selection means all pages are selected.
// EnsureSelectedPages selects all pages.
func ensureSelectedPages(ctx *pdf.Context, selectedPages *pdf.IntSet) {

	if selectedPages != nil && len(*selectedPages) > 0 {
		return
	}

	m := pdf.IntSet{}
	for i := 1; i <= ctx.PageCount; i++ {
		m[i] = true
	}

	*selectedPages = m
}
