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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

var (
	selectedPagesRegExp *regexp.Regexp
)

func setupRegExpForPageSelection() *regexp.Regexp {
	e := "(\\d+)?-l(-\\d+)?|l(-(\\d+)-?)?"
	e = "[!n]?((-\\d+)|(\\d+(-(\\d+)?)?)|" + e + ")"
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

	//log.CLI.Printf("pageSelection: %s\n", s)

	return strings.Split(s, ","), nil
}

func handlePrefix(v string, negated bool, pageCount int, selectedPages pdf.IntSet) error {
	// -l
	if v == "l" {
		for j := 1; j <= pageCount; j++ {
			selectedPages[j] = !negated
		}
		return nil
	}

	// -l-#
	if strings.HasPrefix(v, "l-") {
		i, err := strconv.Atoi(v[2:])
		if err != nil {
			return err
		}
		if pageCount-i < 1 {
			return nil
		}
		for j := 1; j <= pageCount-i; j++ {
			selectedPages[j] = !negated
		}
		return nil
	}

	// -#
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

func handleSpecificPageOrLastXPages(s string, negated bool, pageCount int, selectedPages pdf.IntSet) error {

	// l
	if s == "l" {
		selectedPages[pageCount] = !negated
		return nil
	}

	// l-#
	if strings.HasPrefix(s, "l-") {
		pr := strings.Split(s[2:], "-")
		i, err := strconv.Atoi(pr[0])
		if err != nil {
			return err
		}
		if pageCount-i < 1 {
			return nil
		}
		j := pageCount - i

		// l-#-
		if strings.HasSuffix(s, "-") {
			j = pageCount
		}
		for i := pageCount - i; i <= j; i++ {
			selectedPages[i] = !negated
		}
		return nil
	}

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

	var thru int
	if pr[1] == "l" {
		// #-l
		thru = pageCount
		if len(pr) == 3 {
			// #-l-#
			i, err := strconv.Atoi(pr[2])
			if err != nil {
				return err
			}
			thru -= i
		}
	} else {
		// #-#
		var err error
		thru, err = strconv.Atoi(pr[1])
		if err != nil {
			return err
		}
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

func sortedPages(selectedPages pdf.IntSet) []int {
	p := []int(nil)
	for i, v := range selectedPages {
		if v {
			p = append(p, i)
		}
	}
	sort.Ints(p)
	return p
}

func logSelPages(selectedPages pdf.IntSet) {
	if !log.IsCLILoggerEnabled() {
		return
	}
	var b strings.Builder
	for _, i := range sortedPages(selectedPages) {
		fmt.Fprintf(&b, "%d,", i)
	}
	s := b.String()
	if len(s) > 1 {
		s = s[:len(s)-1]
	}
	// TODO Supress for multifile cmds
	log.CLI.Printf("pages: %s\n", s)
}

// selectedPages returns a set of used page numbers.
// key==page# => key 0 unused!
func selectedPages(pageCount int, pageSelection []string) (pdf.IntSet, error) {
	selectedPages := pdf.IntSet{}

	for _, v := range pageSelection {

		//log.Stats.Printf("pageExp: <%s>\n", v)

		if v == "even" {
			selectEvenPages(selectedPages, pageCount)
			continue
		}

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

		// -#
		if v[0] == '-' {

			v = v[1:]

			if err := handlePrefix(v, negated, pageCount, selectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// #-
		if v[0] != 'l' && strings.HasSuffix(v, "-") {

			if err := handleSuffix(v[:len(v)-1], negated, pageCount, selectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// l l-# l-#-
		if v[0] == 'l' {
			if err := handleSpecificPageOrLastXPages(v, negated, pageCount, selectedPages); err != nil {
				return nil, err
			}
			continue
		}

		pr := strings.Split(v, "-")
		if len(pr) >= 2 {
			// v contains '-' somewhere in the middle
			// #-# #-l #-l-#
			if err := parsePageRange(pr, pageCount, negated, selectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// #
		if err := handleSpecificPageOrLastXPages(pr[0], negated, pageCount, selectedPages); err != nil {
			return nil, err
		}

	}

	logSelPages(selectedPages)
	return selectedPages, nil
}

// PagesForPageSelection ensures a set of page numbers for an ascending page sequence
// where each page number may appear only once.
func PagesForPageSelection(pageCount int, pageSelection []string, ensureAllforNone bool) (pdf.IntSet, error) {
	if len(pageSelection) > 0 {
		return selectedPages(pageCount, pageSelection)
	}
	if !ensureAllforNone {
		//log.CLI.Printf("pages: none\n")
		return nil, nil
	}
	m := pdf.IntSet{}
	for i := 1; i <= pageCount; i++ {
		m[i] = true
	}
	//log.CLI.Printf("pages: all\n")
	return m, nil
}

func deletePageFromCollection(cp *[]int, p int) {
	a := []int{}
	for _, i := range *cp {
		if i != p {
			a = append(a, i)
		}
	}
	*cp = a
}

func processPageForCollection(cp *[]int, negated bool, i int) {
	if !negated {
		*cp = append(*cp, i)
	} else {
		deletePageFromCollection(cp, i)
	}
}

func collectEvenPages(cp *[]int, pageCount int) {
	for i := 2; i <= pageCount; i += 2 {
		*cp = append(*cp, i)
	}
}

func collectOddPages(cp *[]int, pageCount int) {
	for i := 1; i <= pageCount; i += 2 {
		*cp = append(*cp, i)
	}
}

func handlePrefixForCollection(v string, negated bool, pageCount int, cp *[]int) error {
	// -l
	if v == "l" {
		for j := 1; j <= pageCount; j++ {
			processPageForCollection(cp, negated, j)
		}
		return nil
	}

	// -l-#
	if strings.HasPrefix(v, "l-") {
		i, err := strconv.Atoi(v[2:])
		if err != nil {
			return err
		}
		if pageCount-i < 1 {
			return nil
		}
		for j := 1; j <= pageCount-i; j++ {
			processPageForCollection(cp, negated, j)
		}
		return nil
	}

	// -#
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
		processPageForCollection(cp, negated, j)
	}

	return nil
}

func handleSuffixForCollection(v string, negated bool, pageCount int, cp *[]int) error {
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
		processPageForCollection(cp, negated, j)
	}

	return nil
}

func handleSpecificPageOrLastXPagesForCollection(s string, negated bool, pageCount int, cp *[]int) error {

	// l
	if s == "l" {
		processPageForCollection(cp, negated, pageCount)
		return nil
	}

	// l-#
	if strings.HasPrefix(s, "l-") {
		pr := strings.Split(s[2:], "-")
		i, err := strconv.Atoi(pr[0])
		if err != nil {
			return err
		}
		if pageCount-i < 1 {
			return nil
		}
		j := pageCount - i

		// l-#-
		if strings.HasSuffix(s, "-") {
			j = pageCount
		}
		for i := pageCount - i; i <= j; i++ {
			processPageForCollection(cp, negated, i)
		}
		return nil
	}

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

	processPageForCollection(cp, negated, i)

	return nil
}

func parsePageRangeForCollection(pr []string, pageCount int, negated bool, cp *[]int) error {
	from, err := strconv.Atoi(pr[0])
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if from > pageCount {
		return nil
	}

	var thru int
	if pr[1] == "l" {
		// #-l
		thru = pageCount
		if len(pr) == 3 {
			// #-l-#
			i, err := strconv.Atoi(pr[2])
			if err != nil {
				return err
			}
			thru -= i
		}
	} else {
		// #-#
		var err error
		thru, err = strconv.Atoi(pr[1])
		if err != nil {
			return err
		}
	}

	// Handle overflow gracefully
	if thru < from {
		return nil
	}

	if thru > pageCount {
		thru = pageCount
	}

	for i := from; i <= thru; i++ {
		processPageForCollection(cp, negated, i)
	}

	return nil
}

// PagesForPageCollection returns a slice of page numbers for a page collection.
// Any page number in any order any number of times allowed.
func PagesForPageCollection(pageCount int, pageSelection []string) ([]int, error) {
	collectedPages := []int{}
	for _, v := range pageSelection {

		if v == "even" {
			collectEvenPages(&collectedPages, pageCount)
			continue
		}

		if v == "odd" {
			collectOddPages(&collectedPages, pageCount)
			continue
		}

		var negated bool
		if negation(v[0]) {
			negated = true
			//logInfoAPI.Printf("is a negated exp\n")
			v = v[1:]
		}

		// -#
		if v[0] == '-' {

			v = v[1:]

			if err := handlePrefixForCollection(v, negated, pageCount, &collectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// #-
		if v[0] != 'l' && strings.HasSuffix(v, "-") {

			if err := handleSuffixForCollection(v[:len(v)-1], negated, pageCount, &collectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// l l-# l-#-
		if v[0] == 'l' {
			if err := handleSpecificPageOrLastXPagesForCollection(v, negated, pageCount, &collectedPages); err != nil {
				return nil, err
			}
			continue
		}

		pr := strings.Split(v, "-")
		if len(pr) >= 2 {
			// v contains '-' somewhere in the middle
			// #-# #-l #-l-#
			if err := parsePageRangeForCollection(pr, pageCount, negated, &collectedPages); err != nil {
				return nil, err
			}

			continue
		}

		// #
		if err := handleSpecificPageOrLastXPagesForCollection(pr[0], negated, pageCount, &collectedPages); err != nil {
			return nil, err
		}
	}
	return collectedPages, nil
}

// PagesForPageRange returns a slice of page numbers for a page range.
func PagesForPageRange(from, thru int) []int {
	s := make([]int, thru-from+1)
	for i := 0; i < len(s); i++ {
		s[i] = from + i
	}
	return s
}
