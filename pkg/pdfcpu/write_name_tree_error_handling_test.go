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
	"bytes"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestWriteRootAddsBindNameTreesContext(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Root = types.NewIndirectRef(1, 0)
	ctx.RootDict = types.NewDict()
	ctx.Table[0] = model.NewXRefTableEntryGen0(nil)
	ctx.Names["JavaScript"] = &model.Node{
		D: types.NewDict(),
	}

	err = writeRootObject(ctx)

	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"write root: bind name trees",
		`name tree "JavaScript"`,
		"root dictionary",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}
