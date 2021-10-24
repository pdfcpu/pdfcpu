/*
Copyright 2018 The pdfcpu Authors.

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

import "github.com/pdfcpu/pdfcpu/pkg/log"

func rotatePage(xRefTable *XRefTable, i, j int) error {

	log.Debug.Printf("rotate page:%d\n", i)

	consolidateRes := false
	d, _, inhPAttrs, err := xRefTable.PageDict(i, consolidateRes)
	if err != nil {
		return err
	}

	d.Update("Rotate", Integer((inhPAttrs.Rotate+j)%360))

	return nil
}

// RotatePages rotates all selected pages by a multiple of 90 degrees.
func RotatePages(ctx *Context, selectedPages IntSet, rotation int) error {

	for k, v := range selectedPages {
		if v {
			err := rotatePage(ctx.XRefTable, k, rotation)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
