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
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// CollectPages creates a new PDF Context for a custom PDF page sequence of the PDF represented by ctx.
func CollectPages(ctx *Context, collectedPages []int) (*Context, error) {

	log.Debug.Printf("CollectPages %v\n", collectedPages)

	ctxDest, err := CreateContextWithXRefTable(nil, PaperSize["A4"])
	if err != nil {
		return nil, err
	}

	usePgCache := true
	if err := AddPages(ctx, ctxDest, collectedPages, usePgCache); err != nil {
		return nil, err
	}

	return ctxDest, nil
}
