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
	"strings"
)

// KeywordsList returns a list of keywords as recorded in the document info dict.
func KeywordsList(xRefTable *XRefTable) ([]string, error) {
	ss := strings.FieldsFunc(xRefTable.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
	}
	return ss, nil
}

// KeywordsAdd adds keywords to the document info dict.
// Returns true if at least one keyword was added.
func KeywordsAdd(xRefTable *XRefTable, keywords []string) error {

	list, err := KeywordsList(xRefTable)
	if err != nil {
		return err
	}

	for _, s := range keywords {
		if !MemberOf(s, list) {
			xRefTable.Keywords += ", " + UTF8ToCP1252(s)
		}
	}

	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return err
	}

	d["Keywords"] = StringLiteral(xRefTable.Keywords)

	return nil
}

// KeywordsRemove deletes keywords from the document info dict.
// Returns true if at least one keyword was removed.
func KeywordsRemove(xRefTable *XRefTable, keywords []string) (bool, error) {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(keywords) == 0 {
		// Remove all keywords.
		delete(d, "Keywords")
		return true, nil
	}

	kw := make([]string, len(keywords))
	for i, s := range keywords {
		kw[i] = UTF8ToCP1252(s)
	}

	// Distil document keywords.
	ss := strings.FieldsFunc(xRefTable.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })

	xRefTable.Keywords = ""
	var removed bool
	first := true

	for _, s := range ss {
		s = strings.TrimSpace(s)
		if MemberOf(s, kw) {
			removed = true
			continue
		}
		if first {
			xRefTable.Keywords = s
			first = false
			continue
		}
		xRefTable.Keywords += ", " + s
	}

	if removed {
		d["Keywords"] = StringLiteral(xRefTable.Keywords)
	}

	return removed, nil
}
