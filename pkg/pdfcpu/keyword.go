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
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// KeywordsList returns a list of keywords as recorded in the document info dict and supports cancellation.
func KeywordsList(c context.Context, ctx *model.Context) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	var ss []string
	for keyword, val := range ctx.KeywordList {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		if val {
			ss = append(ss, keyword)
		}
	}
	sort.Strings(ss)
	return ss, contextutil.Check(c)
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

func removeKeywordsFromMetadata(c context.Context, ctx *model.Context) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	rootDict, sd, err := keywordMetadataStream(ctx)
	if err != nil {
		return false, err
	}

	if err = sd.Decode(); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: decode: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	before := sd.Content
	if err = model.RemoveKeywords(&sd.Content); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: remove keywords: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if bytes.Equal(before, sd.Content) {
		return false, nil
	}

	//fmt.Println(hex.Dump(sd.Content))

	if err := sd.Encode(); err != nil {
		return false, fmt.Errorf("catalog Metadata stream: encode: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	if err := storeKeywordMetadataStream(ctx, rootDict, *sd); err != nil {
		return false, err
	}
	return true, contextutil.Check(c)
}

func finalizeKeywords(c context.Context, ctx *model.Context) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	ss, err := KeywordsList(c, ctx)
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
		if _, err := removeKeywordsFromMetadata(c, ctx); err != nil {
			return err
		}
	}

	return contextutil.Check(c)
}

func prepareKeywordsInfo(c context.Context, ctx *model.Context) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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
	return contextutil.Check(c)
}

// KeywordsAdd adds keywords to the document info dict and supports cancellation.
func KeywordsAdd(c context.Context, ctx *model.Context, keywords []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := prepareKeywordsInfo(c, ctx); err != nil {
		return err
	}

	for _, keyword := range keywords {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		ctx.KeywordList[strings.TrimSpace(keyword)] = true
	}

	return finalizeKeywords(c, ctx)
}

// KeywordsRemove deletes keywords from the document info dict.
// It supports cancellation and returns true if at least one keyword was removed.
func KeywordsRemove(c context.Context, ctx *model.Context, keywords []string) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
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
		return removeAllKeywords(c, ctx, d)
	}
	return removeSelectedKeywords(c, ctx, keywords)
}

func removeAllKeywords(c context.Context, ctx *model.Context, d types.Dict) (bool, error) {
	_, removed := d["Keywords"]
	delete(d, "Keywords")

	if ctx.CatalogXMPMeta != nil {
		metadataRemoved, err := removeKeywordsFromMetadata(c, ctx)
		if err != nil {
			return false, err
		}
		removed = removed || metadataRemoved
	}

	for keyword, active := range ctx.KeywordList {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		removed = removed || active
		ctx.KeywordList[keyword] = false
	}
	return removed, contextutil.Check(c)
}

func removeSelectedKeywords(c context.Context, ctx *model.Context, keywords []string) (bool, error) {
	remove := types.StringSet{}
	for _, keyword := range keywords {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		remove[strings.TrimSpace(keyword)] = true
	}
	var removed bool
	for keyword := range ctx.KeywordList {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		if remove[keyword] {
			ctx.KeywordList[keyword] = false
			removed = true
		}
	}
	if removed {
		return true, finalizeKeywords(c, ctx)
	}
	return false, contextutil.Check(c)
}
