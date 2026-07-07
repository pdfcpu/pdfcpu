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
	"bytes"
	"errors"
	"fmt"
	"sort"
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
	sort.Strings(ss)
	return ss, nil
}

func keywordMetadataStream(ctx *model.Context) (types.Dict, *types.StreamDict, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, nil, fmt.Errorf("catalog Metadata stream: access catalog: %w", err)
	}

	o, found := rootDict["Metadata"]
	if !found {
		return nil, nil, errors.New("catalog Metadata stream: missing")
	}
	sd, _, err := ctx.DereferenceStreamDict(o)
	if err != nil {
		return nil, nil, fmt.Errorf("catalog Metadata stream: dereference: %w", err)
	}
	if sd == nil {
		return nil, nil, errors.New("catalog Metadata stream: missing object")
	}
	return rootDict, sd, nil
}

func storeKeywordMetadataStream(ctx *model.Context, rootDict types.Dict, sd types.StreamDict) error {
	indRef, ok := rootDict["Metadata"].(types.IndirectRef)
	if !ok {
		rootDict["Metadata"] = sd
		return nil
	}

	entry, found := ctx.FindTableEntryForIndRef(&indRef)
	if !found || entry == nil {
		return fmt.Errorf("catalog Metadata stream obj#%d: missing xref entry", indRef.ObjectNumber.Value())
	}
	entry.Object = sd
	return nil
}

func removeKeywordsFromMetadata(ctx *model.Context) (bool, error) {
	rootDict, sd, err := keywordMetadataStream(ctx)
	if err != nil {
		return false, err
	}

	if err = sd.Decode(); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: decode: %w", err)
	}

	before := sd.Content
	if err = model.RemoveKeywords(&sd.Content); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: remove keywords: %w", err)
	}
	if bytes.Equal(before, sd.Content) {
		return false, nil
	}

	//fmt.Println(hex.Dump(sd.Content))

	if err := sd.Encode(); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: encode: %w", err)
	}

	if err := storeKeywordMetadataStream(ctx, rootDict, *sd); err != nil {
		return false, err
	}
	return true, nil
}

func finalizeKeywords(ctx *model.Context) error {
	if ctx.Info == nil {
		return errors.New("Info dictionary: missing")
	}
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		return fmt.Errorf("Info dictionary: dereference: %w", err)
	}
	if d == nil {
		return errors.New("Info dictionary: missing object")
	}

	ss, err := KeywordsList(ctx)
	if err != nil {
		return fmt.Errorf("Info dictionary Keywords: collect keywords: %w", err)
	}

	s0 := strings.Join(ss, "; ")

	s, err := types.EscapedUTF16String(s0)
	if err != nil {
		return fmt.Errorf("Info dictionary Keywords: encode text: %w", err)
	}

	d["Keywords"] = types.StringLiteral(*s)

	if ctx.CatalogXMPMeta != nil {
		if _, err := removeKeywordsFromMetadata(ctx); err != nil {
			return err
		}
	}

	return nil
}

func prepareKeywordsInfo(ctx *model.Context) error {
	if ctx.XRefTable.Version() < model.V20 {
		if err := ensureInfoDict(ctx); err != nil {
			return fmt.Errorf("Info dictionary: ensure: %w", err)
		}
	}
	if ctx.Info == nil {
		return errors.New("Info dictionary: missing")
	}
	if err := ensureFileID(ctx); err != nil {
		return fmt.Errorf("file ID: ensure: %w", err)
	}
	return nil
}

// KeywordsAdd adds keywords to the document info dict.
func KeywordsAdd(ctx *model.Context, keywords []string) error {
	if err := prepareKeywordsInfo(ctx); err != nil {
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
	if err != nil {
		return false, fmt.Errorf("Info dictionary: dereference: %w", err)
	}
	if d == nil {
		return false, errors.New("Info dictionary: missing object")
	}

	if len(keywords) == 0 {
		// Remove all keywords.
		_, removed := d["Keywords"]
		delete(d, "Keywords")

		if ctx.CatalogXMPMeta != nil {
			metadataRemoved, err := removeKeywordsFromMetadata(ctx)
			if err != nil {
				return false, err
			}
			removed = removed || metadataRemoved
		}

		for keyword, active := range ctx.KeywordList {
			removed = removed || active
			ctx.KeywordList[keyword] = false
		}

		return removed, nil
	}

	remove := types.StringSet{}
	for _, keyword := range keywords {
		remove[strings.TrimSpace(keyword)] = true
	}
	var removed bool
	for keyword := range ctx.KeywordList {
		if remove[keyword] {
			ctx.KeywordList[keyword] = false
			removed = true
		}
	}

	if removed {
		err = finalizeKeywords(ctx)
	}

	return removed, err
}
