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
	"errors"
	"path/filepath"
	"strings"
	"testing"

	corefont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func useMissingGlobalFontDirectory(t *testing.T) {
	t.Helper()
	originalDir := corefont.UserFontDir
	corefont.UserFontDir = filepath.Join(t.TempDir(), "missing")
	if err := corefont.ReloadUserFonts(); err == nil {
		t.Fatal("expected missing global font directory error")
	}
	t.Cleanup(func() {
		corefont.UserFontDir = originalDir
		if err := corefont.ReloadUserFonts(); err != nil {
			t.Errorf("restore global font directory: %v", err)
		}
	})
}

func TestStatelessPrimitiveFontLookupsDoNotUseGlobalRepository(t *testing.T) {
	useMissingGlobalFontDirectory(t)
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	indRef := types.NewIndirectRef(7, 0)
	ctx.XRefTable.Table[7] = model.NewXRefTableEntryGen0(types.Dict{
		"Subtype":  types.Name("Type1"),
		"BaseFont": types.Name("Demo"),
	})

	fontName, _, _, err := FormFontDetails(ctx.XRefTable, *indRef)
	if err != nil {
		t.Fatal(err)
	}
	if fontName != "Demo" {
		t.Fatalf("expected font name Demo, got %q", fontName)
	}

	if _, err := fontIndRef(ctx.XRefTable, "Demo", ""); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v from font reference creation, got %v", corefont.ErrUnknownFont, err)
	}
	pdf := &PDF{XRefTable: ctx.XRefTable}
	if _, err := pdf.ensureFont("F0", "Demo", "", model.FontMap{"Demo": {}}); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v from font creation, got %v", corefont.ErrUnknownFont, err)
	}

	pdf.Optimize = &model.OptimizationContext{
		FontObjects:     map[int]*model.FontObject{7: {FontName: "Demo"}},
		FormFontObjects: map[int]*model.FontObject{},
	}
	pdf.FontResIDs = map[int]types.Dict{1: {}}
	if _, err := pdf.idForFontName("Demo", "", model.FontMap{}, model.FontMap{}, 1); err != nil {
		t.Fatal(err)
	}

	fd := types.Dict{"F0": *indRef}
	if _, _, _, _, _, err := fontAttrs(ctx, fd, "F0", "text", map[string]types.IndirectRef{}); !errors.Is(err, corefont.ErrUnknownFont) {
		t.Fatalf("expected %v from replacement font creation, got %v", corefont.ErrUnknownFont, err)
	}
}

func TestFormFontValidationUsesStatelessRepository(t *testing.T) {
	useMissingGlobalFontDirectory(t)
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	pdf := &PDF{XRefTable: ctx.XRefTable}

	unsupported := &FormFont{pdf: pdf, Name: "Demo", Size: 12}
	if err := unsupported.validate(); err == nil || !strings.Contains(err.Error(), "font Demo is unsupported") {
		t.Fatalf("expected unsupported stateless font, got %v", err)
	}
	unsupported.Script = "Latn"
	if err := unsupported.validateScriptSupport(ctx.XRefTable.FontRepository()); err == nil ||
		!strings.Contains(err.Error(), "userfont Demo not available") {
		t.Fatalf("expected unavailable stateless user font, got %v", err)
	}

	if err := (&FormFont{pdf: pdf, Name: "Helvetica", Size: 12}).validate(); err != nil {
		t.Fatalf("validate core font: %v", err)
	}

	pdf.Header = &HorizontalBand{Height: 10, Font: &FormFont{Name: "Helvetica", Size: 12}}
	pdf.Footer = &HorizontalBand{Height: 10, Font: &FormFont{Name: "Helvetica", Size: 12}}
	if err := pdf.validateHeader(); err != nil {
		t.Fatalf("validate header font: %v", err)
	}
	if err := pdf.validateFooter(); err != nil {
		t.Fatalf("validate footer font: %v", err)
	}
}

func TestPDFValidateWrapsTopLevelPhase(t *testing.T) {
	pdf := &PDF{
		Paper: "bogus",
		Conf:  model.NewDefaultConfiguration(),
	}

	err := pdf.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page boundaries") {
		t.Fatalf("expected page boundaries context, got %q", err.Error())
	}
}

func TestPDFValidateWrapsPagePhase(t *testing.T) {
	pdf := &PDF{
		Conf: model.NewDefaultConfiguration(),
		Pages: map[string]*PDFPage{
			"2": {},
		},
	}

	err := pdf.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 2: validate") {
		t.Fatalf("expected page validation context, got %q", err.Error())
	}
}
