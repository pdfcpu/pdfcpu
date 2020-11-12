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
)

// PropertiesList returns a list of document properties as recorded in the document info dict.
func PropertiesList(xRefTable *XRefTable) ([]string, error) {
	list := make([]string, 0, len(xRefTable.Properties))
	keys := make([]string, len(xRefTable.Properties))
	i := 0
	for k := range xRefTable.Properties {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := xRefTable.Properties[k]
		list = append(list, fmt.Sprintf("%s = %s", k, v))
	}
	return list, nil
}

// PropertiesAdd adds properties into the document info dict.
// Returns true if at least one property was added.
func PropertiesAdd(xRefTable *XRefTable, properties map[string]string) error {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return err
	}
	for k, v := range properties {
		k1 := UTF8ToCP1252(k)
		v1 := UTF8ToCP1252(v)
		d[k1] = StringLiteral(v1)
		xRefTable.Properties[k1] = v1
	}
	return nil
}

// PropertiesRemove deletes specified properties.
// Returns true if at least one property was removed.
func PropertiesRemove(xRefTable *XRefTable, properties []string) (bool, error) {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(properties) == 0 {
		// Remove all properties.
		for k := range xRefTable.Properties {
			k1 := UTF8ToCP1252(k)
			delete(d, k1)
		}
		xRefTable.Properties = map[string]string{}
		return true, nil
	}

	var removed bool
	for _, k := range properties {
		k1 := UTF8ToCP1252(k)
		_, ok := d[k1]
		if ok && !removed {
			delete(d, k1)
			delete(xRefTable.Properties, k1)
			removed = true
		}
	}

	return removed, nil
}
