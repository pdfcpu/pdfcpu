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
	"fmt"
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PropertiesList returns a list of document properties as recorded in the document info dict.
func PropertiesList(ctx *model.Context) ([]string, error) {
	list := make([]string, 0, len(ctx.Properties))
	keys := make([]string, len(ctx.Properties))
	i := 0
	for k := range ctx.Properties {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := ctx.Properties[k]
		list = append(list, fmt.Sprintf("%s = %s", k, v))
	}
	return list, nil
}

// PropertiesAdd adds properties into the document info dict.
// Returns true if at least one property was added.
func PropertiesAdd(ctx *model.Context, properties map[string]string) error {
	if err := ensureInfoDictAndFileID(ctx); err != nil {
		return err
	}

	d, _ := ctx.DereferenceDict(*ctx.Info)

	for k, v := range properties {
		k1 := types.UTF8ToCP1252(k)
		d[k1] = types.StringLiteral(v)
		ctx.Properties[k1] = v
	}

	return nil
}

// PropertiesRemove deletes specified properties.
// Returns true if at least one property was removed.
func PropertiesRemove(ctx *model.Context, properties []string) (bool, error) {
	if ctx.Info == nil {
		return false, nil
	}
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(properties) == 0 {
		// Remove all properties.
		for k := range ctx.Properties {
			k1 := types.UTF8ToCP1252(k)
			delete(d, k1)
		}
		ctx.Properties = map[string]string{}
		return true, nil
	}

	var removed bool
	for _, k := range properties {
		k1 := types.UTF8ToCP1252(k)
		_, ok := d[k1]
		if ok && !removed {
			delete(d, k1)
			delete(ctx.Properties, k1)
			removed = true
		}
	}

	return removed, nil
}
