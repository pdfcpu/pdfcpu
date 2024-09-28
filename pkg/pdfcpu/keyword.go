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
	var ss []string
	for keyword, val := range ctx.KeywordList {
		if val {
			ss = append(ss, keyword)
		}
	}
	return ss, nil
}

func removeKeywordsFromMetadata(ctx *model.Context) error {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	indRef, _ := rootDict["Metadata"].(types.IndirectRef)
	entry, _ := ctx.FindTableEntryForIndRef(&indRef)
	sd, _ := entry.Object.(types.StreamDict)

	if err = sd.Decode(); err != nil {
		return err
	}

	if err = model.RemoveKeywords(&sd.Content); err != nil {
		return err
	}

	//fmt.Println(hex.Dump(sd.Content))

	if err := sd.Encode(); err != nil {
		return err
	}

	entry.Object = sd

	return nil
}

func finalizeKeywords(ctx *model.Context) error {
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil || d == nil {
		return err
	}

	ss, err := KeywordsList(ctx)
	if err != nil {
		return err
	}

	s0 := strings.Join(ss, "; ")

	s, err := types.EscapedUTF16String(s0)
	if err != nil {
		return err
	}

	d["Keywords"] = types.StringLiteral(*s)

	if ctx.CatalogXMPMeta != nil {
		removeKeywordsFromMetadata(ctx)
	}

	return nil
}

// KeywordsAdd adds keywords to the document info dict.
// Returns true if at least one keyword was added.
func KeywordsAdd(ctx *model.Context, keywords []string) error {
	if err := ensureInfoDictAndFileID(ctx); err != nil {
		return err
	}

	for _, keyword := range keywords {
		ctx.KeywordList[strings.TrimSpace(keyword)] = true
	}

	return finalizeKeywords(ctx)
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

		if ctx.CatalogXMPMeta != nil {
			removeKeywordsFromMetadata(ctx)
		}

		return true, nil
	}

	var removed bool
	for keyword := range ctx.KeywordList {
		if types.MemberOf(keyword, keywords) {
			ctx.KeywordList[keyword] = false
			removed = true
		}
	}

	if removed {
		err = finalizeKeywords(ctx)
	}

	return removed, err
}
