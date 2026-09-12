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

package primitives

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	corefont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestFormFontAcceptsFractionalSize(t *testing.T) {
	var f FormFont
	if err := json.Unmarshal([]byte(`{"name":"Helvetica","size":10.125}`), &f); err != nil {
		t.Fatal(err)
	}
	if f.Size != 10.125 {
		t.Fatalf("font size = %g, want 10.125", f.Size)
	}
}

func TestTextFieldCombEscapesEachCell(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	tf := TextField{
		Value:       "A(B",
		Comb:        true,
		MaxLen:      3,
		BoundingBox: types.RectForDim(60, 20),
		Font: &FormFont{
			Name: "Helvetica",
			Size: 10,
			col:  &color.Black,
		},
		fontID: "Helv",
	}

	bb, err := tf.renderN(t.Context(), ctx.XRefTable)
	if err != nil {
		t.Fatal(err)
	}

	content := string(bb)
	if strings.Contains(content, `(\) Tj`) {
		t.Fatalf("comb appearance splits escaped PDF string literal: %s", content)
	}
	if !strings.Contains(content, `(\() Tj`) {
		t.Fatalf("comb appearance missing escaped left parenthesis cell: %s", content)
	}
	if got := strings.Count(content, " Tj "); got != 3 {
		t.Fatalf("comb appearance Tj count = %d, want 3: %s", got, content)
	}
}

func TestTextFieldMetricsUseStatelessRepository(t *testing.T) {
	useMissingGlobalFontDirectory(t)
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	tf := TextField{
		Value:       "text",
		BoundingBox: types.RectForDim(60, 20),
		Font: &FormFont{
			Name: "Demo",
			Size: 10,
			col:  &color.Black,
		},
		fontID: "F0",
	}

	if _, err := tf.renderN(t.Context(), ctx.XRefTable); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", corefont.ErrUnknownFont, err)
	}
	if _, err := textFieldLines(t.Context(), ctx.XRefTable, "text", "Demo", 10, true, 60); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v from multiline wrapping, got %v", corefont.ErrUnknownFont, err)
	}
	if err := tf.renderLines(
		t.Context(),
		ctx.XRefTable,
		ctx.XRefTable.FontRepository(),
		0,
		10,
		60,
		10,
		[]string{"text"},
		io.Discard,
	); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v from text bounding box, got %v", corefont.ErrUnknownFont, err)
	}
}
