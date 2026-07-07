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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func requireContext(ctx *model.Context) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}
	return nil
}

func requireContextWithXRefTable(ctx *model.Context) error {
	if err := requireContext(ctx); err != nil {
		return err
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}
	return nil
}

func requireStreamDict(sd *types.StreamDict) error {
	if sd == nil {
		return ErrMissingStreamDict
	}
	return nil
}

func requireOptimizedContext(ctx *model.Context) error {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return err
	}
	if ctx.Optimize == nil {
		return ErrMissingOptimizationContext
	}
	return nil
}
