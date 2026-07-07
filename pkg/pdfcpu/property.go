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
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PropertiesAdd adds properties into the document info dict.
func PropertiesAdd(ctx *model.Context, properties map[string]string) error {
	if err := preparePropertiesInfo(ctx); err != nil {
		return err
	}
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		return fmt.Errorf("Info dictionary: dereference: %w", err)
	}
	if d == nil {
		return errors.New("Info dictionary: missing object")
	}

	for k, v := range properties {
		s, err := types.EscapedUTF16String(v)
		if err != nil {
			return fmt.Errorf("Info dictionary property %q: encode value: %w", k, err)
		}
		d[k] = types.StringLiteral(*s)
		ctx.Properties[k] = *s
	}

	return nil
}

func preparePropertiesInfo(ctx *model.Context) error {
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

// PropertiesRemove deletes specified document properties.
// If properties is empty, it removes all properties and catalog XMP metadata.
// Returns true if at least one property was removed.
func PropertiesRemove(ctx *model.Context, properties []string) (bool, error) {
	if len(properties) == 0 {
		return removeAllProperties(ctx)
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

	var removed bool
	for _, k := range properties {
		_, ok := d[k]
		if ok {
			delete(d, k)
			delete(ctx.Properties, k)
			removed = true
		}
	}

	return removed, nil
}

func removeAllProperties(ctx *model.Context) (bool, error) {
	var removed bool

	if ctx.Info != nil {
		d, err := ctx.DereferenceDict(*ctx.Info)
		if err != nil {
			return false, fmt.Errorf("Info dictionary: dereference: %w", err)
		}
		if d == nil {
			return false, errors.New("Info dictionary: missing object")
		}
		for k := range ctx.Properties {
			delete(d, types.EncodeName(k))
			removed = true
		}
		ctx.Properties = map[string]string{}
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return removed, fmt.Errorf("catalog: access: %w", err)
	}
	if _, ok := rootDict["Metadata"]; ok {
		delete(rootDict, "Metadata")
		ctx.CatalogXMPMeta = nil
		removed = true
	}

	return removed, nil
}
