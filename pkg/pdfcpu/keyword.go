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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// KeywordsList returns a list of keywords as recorded in the document info dict.
func KeywordsList(ctx *model.Context) ([]string, error) {
	ss := strings.FieldsFunc(ctx.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
	}
	return ss, nil
}

// KeywordsAdd adds keywords to the document info dict.
// Returns true if at least one keyword was added.
func KeywordsAdd(ctx *model.Context, keywords []string) error {
	if err := ensureInfoDictAndFileID(ctx); err != nil {
		return err
	}

	list, err := KeywordsList(ctx)
	if err != nil {
		return err
	}

	for _, kw := range keywords {
		if !types.MemberOf(kw, list) {
			if len(ctx.Keywords) == 0 {
				ctx.Keywords = kw
			} else {
				ctx.Keywords += ", " + kw
			}
		}
	}

	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil || d == nil {
		return err
	}

	s, err := types.EscapeUTF16String(ctx.Keywords)
	if err != nil {
		return err
	}

	d["Keywords"] = types.StringLiteral(*s)

	return nil
}

// KeywordsRemove deletes keywords from the document info dict.
// Returns true if at least one keyword was removed.
func KeywordsRemove(ctx *model.Context, keywords []string) (bool, error) {
	if ctx.Info == nil {
		return false, nil
	}

	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(keywords) == 0 {
		// Remove all keywords.
		delete(d, "Keywords")
		return true, nil
	}

	ss := strings.FieldsFunc(ctx.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })

	ctx.Keywords = ""
	var removed bool
	first := true

	for _, s := range ss {
		s = strings.TrimSpace(s)
		if types.MemberOf(s, keywords) {
			removed = true
			continue
		}

		if first {
			ctx.Keywords = s
			first = false
			continue
		}

		ctx.Keywords += ", " + s
	}

	if removed {

		s, err := types.EscapeUTF16String(ctx.Keywords)
		if err != nil {
			return false, err
		}

		d["Keywords"] = types.StringLiteral(*s)
	}

	return removed, nil
}
