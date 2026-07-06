/*
Copyright 2026 The pdfcpu Authors.

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
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendPDFObject(dst []byte, obj types.Object) ([]byte, error) {
	switch obj := obj.(type) {
	case nil:
		return append(dst, "null"...), nil
	case types.Dict:
		return appendPDFDict(dst, obj)
	case types.Array:
		return appendPDFArray(dst, obj)
	case types.IndirectRef:
		return append(dst, obj.PDFString()...), nil
	case types.Name:
		return append(dst, obj.PDFString()...), nil
	case types.Integer:
		return strconv.AppendInt(dst, int64(obj), 10), nil
	case types.Float:
		return strconv.AppendFloat(dst, obj.Value(), 'f', 12, 64), nil
	case types.Boolean:
		return strconv.AppendBool(dst, obj.Value()), nil
	case types.StringLiteral:
		return append(dst, obj.PDFString()...), nil
	case types.HexLiteral:
		return append(dst, obj.PDFString()...), nil
	case types.LazyObjectStreamObject:
		return appendLazyObjectStreamObject(dst, obj)
	default:
		return nil, fmt.Errorf("unsupported PDF object type %T", obj)
	}
}

func appendLazyObjectStreamObject(dst []byte, obj types.LazyObjectStreamObject) ([]byte, error) {
	data, err := obj.GetData()
	if err != nil {
		return nil, err
	}
	return append(dst, data...), nil
}

func appendPDFArray(dst []byte, a types.Array) ([]byte, error) {
	dst = append(dst, '[')
	for i, obj := range a {
		if i > 0 && arrayObjectNeedsSpace(obj) {
			dst = append(dst, ' ')
		}
		var err error
		dst, err = appendPDFObject(dst, obj)
		if err != nil {
			return nil, err
		}
	}
	return append(dst, ']'), nil
}

func arrayObjectNeedsSpace(obj types.Object) bool {
	switch obj.(type) {
	case types.Dict, types.Array, types.Name:
		return false
	default:
		return true
	}
}

func appendPDFDict(dst []byte, d types.Dict) ([]byte, error) {
	dst = append(dst, "<<"...)
	for _, key := range sortedDictKeys(d) {
		var err error
		dst, err = appendPDFDictEntry(dst, key, d[key])
		if err != nil {
			return nil, err
		}
	}
	return append(dst, ">>"...), nil
}

func sortedDictKeys(d types.Dict) []string {
	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendPDFDictEntry(dst []byte, key string, obj types.Object) ([]byte, error) {
	dst = append(dst, '/')
	dst = append(dst, types.EncodeName(key)...)
	if dictObjectNeedsSpace(obj) {
		dst = append(dst, ' ')
	}
	return appendPDFObject(dst, obj)
}

func dictObjectNeedsSpace(obj types.Object) bool {
	switch obj.(type) {
	case types.Dict, types.Array, types.Name, types.StringLiteral, types.HexLiteral:
		return false
	default:
		return true
	}
}
