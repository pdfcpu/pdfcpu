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
	"context"
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PropertiesAdd adds properties into the document info dict and supports cancellation.
func PropertiesAdd(c context.Context, ctx *model.Context, properties map[string]string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := preparePropertiesInfo(c, ctx); err != nil {
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
		if err := contextutil.Check(c); err != nil {
			return err
		}
		s, err := types.EscapedUTF16String(v)
		if err != nil {
			return fmt.Errorf("Info dictionary property %q: encode value: %w", k, err)
		}
		d[k] = types.StringLiteral(*s)
		ctx.Properties[k] = *s
	}

	return contextutil.Check(c)
}

func preparePropertiesInfo(c context.Context, ctx *model.Context) error {
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

// PropertiesRemove deletes specified document properties.
// It supports cancellation and returns true if at least one property was removed.
// If properties is empty, it removes all properties and catalog XMP metadata.
func PropertiesRemove(c context.Context, ctx *model.Context, properties []string) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if len(properties) == 0 {
		return removeAllProperties(c, ctx)
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
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		_, ok := d[k]
		if ok {
			delete(d, k)
			delete(ctx.Properties, k)
			removed = true
		}
	}

	return removed, contextutil.Check(c)
}

func removeAllProperties(c context.Context, ctx *model.Context) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
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
			if err := contextutil.Check(c); err != nil {
				return false, err
			}
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

	return removed, contextutil.Check(c)
}
