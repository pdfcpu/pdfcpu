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
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type stampFailingWriter struct {
	err error
}

func (w stampFailingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

// TestParseWatermarkDetailsPreservesScaleNumericCause verifies shared scale parsing wraps strconv errors.
func TestParseWatermarkDetailsPreservesScaleNumericCause(t *testing.T) {
	_, err := ParseTextWatermarkDetails("draft", "scalefactor:nope", true, types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), "scale factor must be a float value") {
		t.Fatalf("expected scale-factor context, got %q", err)
	}
}

func stampContentTestWatermark() *model.Watermark {
	wm := model.DefaultWatermarkConfig()
	wm.Bb = types.RectForDim(100, 100)
	wm.Vp = types.RectForFormat("A4")
	return wm
}

func TestAddWatermarksRejectsMissingInputs(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		wm      *model.Watermark
		wantErr error
	}{
		{
			name:    "context",
			wm:      model.DefaultWatermarkConfig(),
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "xref table",
			ctx:     &model.Context{},
			wm:      model.DefaultWatermarkConfig(),
			wantErr: ErrMissingXRefTable,
		},
		{
			name:    "watermark configuration",
			ctx:     &model.Context{XRefTable: &model.XRefTable{}},
			wantErr: ErrMissingWatermarkConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddWatermarks(tt.ctx, nil, tt.wm)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAddWatermarksDocumentSetupErrorIncludesPhaseContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	err := AddWatermarks(ctx, nil, model.DefaultWatermarkConfig())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "prepare optional content") {
		t.Fatalf("expected optional content context in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "missing root dict") {
		t.Fatalf("expected catalog detail in %q", err.Error())
	}
}

func TestAddWatermarksCorruptOptionalContentDoesNotPanic(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.RootDict["OCProperties"] = types.Name("broken")

	err := AddWatermarks(ctx, nil, model.DefaultWatermarkConfig())
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"prepare optional content", "optional content properties"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestAddWatermarksResourceErrorsIncludeWatermarkType(t *testing.T) {
	tests := []struct {
		name    string
		wm      *model.Watermark
		wantErr error
		want    string
	}{
		{
			name: "missing image reader",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Mode = model.WMImage
				return wm
			}(),
			wantErr: ErrMissingImageReader,
			want:    "prepare resources: image",
		},
		{
			name: "empty PDF reader",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Mode = model.WMPDF
				wm.PDF = bytes.NewReader(nil)
				return wm
			}(),
			wantErr: ErrEmptyInput,
			want:    "prepare resources: PDF: read source",
		},
		{
			name: "missing PDF source",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Mode = model.WMPDF
				return wm
			}(),
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "prepare resources: PDF: missing PDF source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddWatermarks(testOptimizeContext(t), nil, tt.wm)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestAddWatermarksPageErrorIncludesDestinationPage(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.PageCount = 1

	err := AddWatermarks(ctx, nil, model.DefaultWatermarkConfig())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1") {
		t.Fatalf("expected destination page context in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "page dictionary") {
		t.Fatalf("expected page dictionary context in %q", err.Error())
	}
}

func TestAddWatermarksRejectsInvalidDestinationStartPage(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.PageCount = 1
	wm := model.DefaultWatermarkConfig()
	wm.PdfMultiStartPageNrDest = 0

	err := AddWatermarks(ctx, nil, wm)
	if !errors.Is(err, ErrInvalidPageNumber) {
		t.Fatalf("expected %v, got %v", ErrInvalidPageNumber, err)
	}
	if !strings.Contains(err.Error(), "page 0") {
		t.Fatalf("expected destination page context in %q", err.Error())
	}
}

func TestCreateFormRejectsIncompleteConfiguration(t *testing.T) {
	ctx := testOptimizeContext(t)
	ocg := types.NewIndirectRef(1, 0)
	tests := []struct {
		name string
		wm   func() *model.Watermark
		want string
	}{
		{
			name: "missing optional content group",
			wm:   model.DefaultWatermarkConfig,
			want: "missing optional content group",
		},
		{
			name: "missing font resource",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Ocg = ocg
				wm.Vp = types.RectForFormat("A4")
				wm.TextString = "draft"
				wm.TextLines = []string{"draft"}
				return wm
			},
			want: "create resource dictionary: missing font resource",
		},
		{
			name: "missing PDF resource",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Mode = model.WMPDF
				wm.Ocg = ocg
				wm.Vp = types.RectForFormat("A4")
				return wm
			},
			want: "calculate bounding box: destination page 1: missing PDF stamp resource",
		},
		{
			name: "invalid image dimensions",
			wm: func() *model.Watermark {
				wm := model.DefaultWatermarkConfig()
				wm.Mode = model.WMImage
				wm.Ocg = ocg
				wm.Vp = types.RectForFormat("A4")
				return wm
			},
			want: "calculate bounding box: invalid image dimensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createForm(ctx, 1, 1, tt.wm(), false)
			if !errors.Is(err, ErrMissingWatermarkConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingWatermarkConfiguration, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestPDFFormContentPreservesWriterError(t *testing.T) {
	wantErr := errors.New("form write failed")
	resDict := types.NewIndirectRef(1, 0)
	wm := model.DefaultWatermarkConfig()
	wm.Mode = model.WMPDF
	wm.PdfPageNrSrc = 1
	wm.Bb = types.RectForDim(100, 100)
	wm.Width = 100
	wm.PdfRes[1] = model.PdfResources{
		Content: []byte("content"),
		ResDict: resDict,
		Bb:      types.RectForDim(100, 100),
	}

	err := pdfFormContent(stampFailingWriter{err: wantErr}, 1, *wm)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "write transform") {
		t.Fatalf("expected write context in %q", err.Error())
	}
}

func TestAddWatermarksInitializesMissingCaches(t *testing.T) {
	ctx := testOptimizeContext(t)
	addOptimizeTestPage(t, ctx)
	wm, err := ParseTextWatermarkDetails("draft", "", false, types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	wm.FCache = nil
	wm.Objs = nil

	if err := AddWatermarks(ctx, nil, wm); err != nil {
		t.Fatal(err)
	}
	if wm.FCache == nil || wm.Objs == nil {
		t.Fatal("expected initialized watermark caches")
	}
}

func TestPageWatermarkResourcesRejectIncompleteState(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
		want    string
	}{
		{
			name: "missing page dictionary",
			fn: func() error {
				return insertPageResourcesForWM(nil, *model.DefaultWatermarkConfig(), "GS0", "Fm0")
			},
			want: "missing page dictionary",
		},
		{
			name: "missing graphics state",
			fn: func() error {
				wm := model.DefaultWatermarkConfig()
				wm.Form = types.NewIndirectRef(1, 0)
				return insertPageResourcesForWM(types.Dict{}, *wm, "GS0", "Fm0")
			},
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "ExtGState: missing reference",
		},
		{
			name: "missing form",
			fn: func() error {
				wm := model.DefaultWatermarkConfig()
				wm.ExtGState = types.NewIndirectRef(1, 0)
				return insertPageResourcesForWM(types.Dict{}, *wm, "GS0", "Fm0")
			},
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "XObject: missing form reference",
		},
		{
			name: "missing resource dictionary",
			fn: func() error {
				wm := model.DefaultWatermarkConfig()
				wm.ExtGState = types.NewIndirectRef(1, 0)
				wm.Form = types.NewIndirectRef(2, 0)
				gsID, xoID := "GS0", "Fm0"
				return updatePageResourcesForWM(testOptimizeContext(t), nil, *wm, &gsID, &xoID)
			},
			want: "missing page resource dictionary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestUpdatePageWatermarkResourcesPreservesDereferenceErrors(t *testing.T) {
	wm := model.DefaultWatermarkConfig()
	wm.ExtGState = types.NewIndirectRef(1, 0)
	wm.Form = types.NewIndirectRef(2, 0)
	gsID, xoID := "GS0", "Fm0"
	resDict := types.Dict{"ExtGState": types.Name("broken")}

	err := updatePageResourcesForWM(testOptimizeContext(t), resDict, *wm, &gsID, &xoID)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ExtGState: dereference dictionary") {
		t.Fatalf("expected ExtGState context in %q", err.Error())
	}
	if !errors.Is(err, model.ErrExpectedDict) {
		t.Fatalf("expected dereference detail in %q", err.Error())
	}
}

func TestUpdatePageWatermarkResourcesAllocatesDistinctIDs(t *testing.T) {
	wm := model.DefaultWatermarkConfig()
	wm.ExtGState = types.NewIndirectRef(3, 0)
	wm.Form = types.NewIndirectRef(4, 0)
	gsDict := types.Dict{"GS0": *types.NewIndirectRef(1, 0)}
	xoDict := types.Dict{"Fm0": *types.NewIndirectRef(2, 0)}
	resDict := types.Dict{"ExtGState": gsDict, "XObject": xoDict}
	gsID, xoID := "GS0", "Fm0"

	if err := updatePageResourcesForWM(testOptimizeContext(t), resDict, *wm, &gsID, &xoID); err != nil {
		t.Fatal(err)
	}
	if gsID != "GS1" || xoID != "Fm1" {
		t.Fatalf("expected GS1/Fm1, got %s/%s", gsID, xoID)
	}
	if !gsDict.HasEntry(gsID) || !xoDict.HasEntry(xoID) {
		t.Fatal("expected inserted resource references")
	}
}

func TestAddWatermarksResourceDictionaryErrorIncludesPagePhase(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	pageDict["Resources"] = types.Dict{"ExtGState": types.Name("broken")}
	wm, err := ParseTextWatermarkDetails("draft", "", false, types.POINTS)
	if err != nil {
		t.Fatal(err)
	}

	err = AddWatermarks(ctx, nil, wm)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page 1", "update resources", "ExtGState: dereference dictionary"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestInsertPageWatermarkContentsRejectsIncompleteState(t *testing.T) {
	tests := []struct {
		name    string
		page    types.Dict
		wm      *model.Watermark
		wantErr error
		want    string
	}{
		{
			name: "missing page dictionary",
			wm:   stampContentTestWatermark(),
			want: "missing page dictionary",
		},
		{
			name:    "missing watermark",
			page:    types.Dict{},
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "generate content",
		},
		{
			name: "existing contents",
			page: types.Dict{"Contents": types.Array{}},
			wm:   stampContentTestWatermark(),
			want: "already has Contents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := insertPageContentsForWM(testOptimizeContext(t), tt.page, tt.wm, "GS0", "Fm0")
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestPatchPageWatermarkContentRejectsUnsupportedFilter(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.NewDict(),
		Raw:  []byte("encoded"),
		FilterPipeline: []types.PDFFilter{
			{Name: filter.JBIG2},
			{Name: filter.ASCIIHex},
		},
	}

	err := patchFirstContentStreamForWatermark(&sd, "GS0", "Fm0", stampContentTestWatermark(), true)
	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "decode stream") {
		t.Fatalf("expected decode context in %q", err.Error())
	}
}

func TestUpdatePageWatermarkContentsHandlesDirectStream(t *testing.T) {
	ctx := testOptimizeContext(t)
	sd, err := ctx.NewStreamDictForBuf([]byte("q Q"))
	if err != nil {
		t.Fatal(err)
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	pageDict := types.Dict{"Contents": *sd}

	if err := updatePageContentsForWM(ctx, pageDict, stampContentTestWatermark(), "GS0", "Fm0"); err != nil {
		t.Fatal(err)
	}
	if _, ok := pageDict["Contents"].(types.StreamDict); !ok {
		t.Fatalf("expected direct stream dictionary, got %T", pageDict["Contents"])
	}
}

func TestUpdatePageWatermarkContentsRejectsMalformedObjects(t *testing.T) {
	ctx := testOptimizeContext(t)
	tests := []struct {
		name string
		obj  types.Object
		want string
	}{
		{
			name: "missing xref entry",
			obj:  *types.NewIndirectRef(999, 0),
			want: "Contents: content stream obj#999: missing xref entry",
		},
		{
			name: "invalid contents type",
			obj:  types.Name("broken"),
			want: "Contents: expected stream dictionary or array",
		},
		{
			name: "invalid array entry",
			obj:  types.Array{types.Name("broken")},
			want: "content array entry 1: expected indirect reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updatePageContentsForWM(ctx, types.Dict{"Contents": tt.obj}, stampContentTestWatermark(), "GS0", "Fm0")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestAddWatermarksContentErrorIncludesPagePhase(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	pageDict["Contents"] = *types.NewIndirectRef(999, 0)
	wm, err := ParseTextWatermarkDetails("draft", "", false, types.POINTS)
	if err != nil {
		t.Fatal(err)
	}

	err = AddWatermarks(ctx, nil, wm)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page 1", "update contents", "Contents: content stream obj#999"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestRemoveArtifactsRejectsUnsupportedFilter(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.NewDict(),
		Raw:  []byte("encoded"),
		FilterPipeline: []types.PDFFilter{
			{Name: filter.JBIG2},
			{Name: filter.ASCIIHex},
		},
	}

	_, _, _, err := removeArtifacts(&sd, 1)
	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "decode content stream") {
		t.Fatalf("expected decode context in %q", err.Error())
	}
}

func TestRemoveArtifactsHandlesDirectAndMalformedContents(t *testing.T) {
	ctx := testOptimizeContext(t)
	sd, err := ctx.NewStreamDictForBuf([]byte("/Artifact <</Subtype /Watermark /Type /Pagination >>BDC EMC"))
	if err != nil {
		t.Fatal(err)
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	resDict := types.Dict{"ExtGState": types.Dict{}, "XObject": types.Dict{}}

	found, obj, err := removeArtifacts1(ctx, *sd, nil, resDict, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected watermark artifact")
	}
	if _, ok := obj.(types.StreamDict); !ok {
		t.Fatalf("expected direct stream dictionary, got %T", obj)
	}

	if found, _, err := removeArtifacts1(ctx, types.Array{}, nil, resDict, 1); err != nil || found {
		t.Fatalf("expected empty array to be ignored, found=%t err=%v", found, err)
	}
	_, _, err = removeArtifacts1(ctx, types.Array{types.Name("broken")}, nil, resDict, 1)
	if err == nil || !strings.Contains(err.Error(), "content array entry 1: expected indirect reference") {
		t.Fatalf("expected malformed array context, got %v", err)
	}
}

func TestRemoveArtifactsReportsMissingContentObject(t *testing.T) {
	_, _, err := removeArtifacts1(
		testOptimizeContext(t),
		types.Array{*types.NewIndirectRef(999, 0)},
		nil,
		types.Dict{"ExtGState": types.Dict{}, "XObject": types.Dict{}},
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "content array entry 1: content stream obj#999: missing xref entry") {
		t.Fatalf("expected object identity in error, got %v", err)
	}
}

func TestAddWatermarksUpdateErrorIncludesRemovalContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	pageDict["Resources"] = types.Dict{"ExtGState": types.Dict{}, "XObject": types.Dict{}}
	pageDict["Contents"] = *types.NewIndirectRef(999, 0)
	wm, err := ParseTextWatermarkDetails("draft", "", false, types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	wm.Update = true

	err = AddWatermarks(ctx, nil, wm)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page 1", "update existing watermark", "Contents: content stream obj#999"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestHandleLinkRejectsMissingPageState(t *testing.T) {
	wm := *model.DefaultWatermarkConfig()
	wm.OnTop = true
	wm.URL = "https://example.com"

	err := handleLink(testOptimizeContext(t), nil, types.Dict{}, 1, wm)
	if err == nil || !strings.Contains(err.Error(), "missing page dictionary reference") {
		t.Fatalf("expected page reference context, got %v", err)
	}
	err = handleLink(testOptimizeContext(t), types.NewIndirectRef(1, 0), nil, 1, wm)
	if err == nil || !strings.Contains(err.Error(), "missing page dictionary") {
		t.Fatalf("expected page dictionary context, got %v", err)
	}
}

func TestCreateFontResourceRejectsIncompleteState(t *testing.T) {
	wm := model.DefaultWatermarkConfig()
	wm.FontName = ""
	err := createFontResForWM(testOptimizeContext(t), wm, map[string]types.IndirectRef{})
	if !errors.Is(err, ErrMissingWatermarkConfiguration) {
		t.Fatalf("expected %v, got %v", ErrMissingWatermarkConfiguration, err)
	}

	wm.FontName = "CustomFont"
	err = createFontResForWM(&model.Context{XRefTable: testOptimizeContext(t).XRefTable}, wm, map[string]types.IndirectRef{})
	if !errors.Is(err, ErrMissingOptimizationContext) {
		t.Fatalf("expected %v, got %v", ErrMissingOptimizationContext, err)
	}
	if !strings.Contains(err.Error(), "font CustomFont") {
		t.Fatalf("expected font identity in %q", err.Error())
	}
}

func TestUpdateUserfontsPreservesFontContextAndCause(t *testing.T) {
	if err := pdffont.UpdateUserfonts(nil, nil); !errors.Is(err, model.ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", model.ErrMissingXRefTable, err)
	}

	ctx := testOptimizeContext(t)
	indRef, err := ctx.IndRefForNewObject(types.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	ctx.UsedGIDs["CustomFont"] = map[uint16]bool{1: true}
	err = pdffont.UpdateUserfonts(ctx.XRefTable, map[string]types.IndirectRef{"CustomFont": *indRef})
	if !errors.Is(err, pdffont.ErrCorruptFontDict) {
		t.Fatalf("expected %v, got %v", pdffont.ErrCorruptFontDict, err)
	}
	for _, want := range []string{"font CustomFont", "inspect font dictionary"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
	if count := strings.Count(err.Error(), pdffont.ErrCorruptFontDict.Error()); count != 1 {
		t.Fatalf("expected one corrupt font dictionary cause, got %d in %q", count, err.Error())
	}

	ctx = testOptimizeContext(t)
	ctx.UsedGIDs["Zulu"] = map[uint16]bool{1: true}
	ctx.UsedGIDs["Alpha"] = map[uint16]bool{1: true}
	err = pdffont.UpdateUserfonts(ctx.XRefTable, map[string]types.IndirectRef{
		"Zulu":  *types.NewIndirectRef(999, 0),
		"Alpha": *types.NewIndirectRef(998, 0),
	})
	if err == nil || !strings.Contains(err.Error(), "font Alpha") {
		t.Fatalf("expected alphabetically first font error, got %v", err)
	}
}

func TestAddWatermarkMapsRejectInvalidDirectInputs(t *testing.T) {
	if err := AddWatermarksMap(nil, map[int]*model.Watermark{1: model.DefaultWatermarkConfig()}); !errors.Is(err, ErrMissingPDFContext) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
	}
	if err := AddWatermarksSliceMap(&model.Context{}, map[int][]*model.Watermark{1: {model.DefaultWatermarkConfig()}}); !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}

	ctx := testOptimizeContext(t)
	addOptimizeTestPage(t, ctx)
	if err := AddWatermarksMap(ctx, nil); !errors.Is(err, ErrMissingWatermarks) {
		t.Fatalf("expected %v, got %v", ErrMissingWatermarks, err)
	}
	if err := AddWatermarksSliceMap(ctx, nil); !errors.Is(err, ErrMissingWatermarks) {
		t.Fatalf("expected %v, got %v", ErrMissingWatermarks, err)
	}
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
		want    string
	}{
		{
			name:    "map nil watermark",
			fn:      func() error { return AddWatermarksMap(ctx, map[int]*model.Watermark{1: nil}) },
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "page 1",
		},
		{
			name: "map invalid page",
			fn: func() error {
				return AddWatermarksMap(ctx, map[int]*model.Watermark{2: model.DefaultWatermarkConfig()})
			},
			wantErr: ErrInvalidPageNumber,
			want:    "page 2",
		},
		{
			name:    "slice map empty page",
			fn:      func() error { return AddWatermarksSliceMap(ctx, map[int][]*model.Watermark{1: nil}) },
			wantErr: ErrMissingWatermarks,
			want:    "page 1",
		},
		{
			name: "slice map nil watermark",
			fn: func() error {
				return AddWatermarksSliceMap(ctx, map[int][]*model.Watermark{1: {model.DefaultWatermarkConfig(), nil}})
			},
			wantErr: ErrMissingWatermarkConfiguration,
			want:    "page 1, watermark 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestSortedWatermarkPagesSupportsMapVariants(t *testing.T) {
	want := []int{1, 2, 3}
	tests := []struct {
		name string
		got  []int
	}{
		{
			name: "watermark map",
			got:  sortedWatermarkPages(map[int]*model.Watermark{3: nil, 1: nil, 2: nil}),
		},
		{
			name: "watermark slice map",
			got:  sortedWatermarkPages(map[int][]*model.Watermark{2: nil, 3: nil, 1: nil}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.got) != len(want) {
				t.Fatalf("expected %v, got %v", want, tt.got)
			}
			for i := range want {
				if tt.got[i] != want[i] {
					t.Fatalf("expected %v, got %v", want, tt.got)
				}
			}
		})
	}
}

func TestAddWatermarkMapsRejectInconsistentSharedSettings(t *testing.T) {
	watermark := func(onTop bool, opacity float64) *model.Watermark {
		wm := model.DefaultWatermarkConfig()
		wm.OnTop = onTop
		wm.Opacity = opacity
		return wm
	}
	ctx := testOptimizeContext(t)
	ctx.PageCount = 2
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "map OnTop",
			fn: func() error {
				return AddWatermarksMap(ctx, map[int]*model.Watermark{
					2: watermark(true, 1),
					1: watermark(false, 1),
				})
			},
			want: "page 2: inconsistent OnTop",
		},
		{
			name: "map opacity",
			fn: func() error {
				return AddWatermarksMap(ctx, map[int]*model.Watermark{
					2: watermark(false, .5),
					1: watermark(false, 1),
				})
			},
			want: "page 2: inconsistent Opacity",
		},
		{
			name: "slice map opacity",
			fn: func() error {
				return AddWatermarksSliceMap(ctx, map[int][]*model.Watermark{
					1: {watermark(false, 1), watermark(false, .5)},
				})
			},
			want: "page 1, watermark 1: inconsistent Opacity",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %v", tt.want, err)
			}
		})
	}
}

func TestAddWatermarkMapsIncludeResourceIdentity(t *testing.T) {
	imageWM := model.DefaultWatermarkConfig()
	imageWM.Mode = model.WMImage
	ctx := testOptimizeContext(t)
	addOptimizeTestPage(t, ctx)
	err := AddWatermarksMap(ctx, map[int]*model.Watermark{1: imageWM})
	if !errors.Is(err, ErrMissingImageReader) {
		t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
	}
	for _, want := range []string{"page 1", "prepare resources", "image"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}

	imageWM = model.DefaultWatermarkConfig()
	imageWM.Mode = model.WMImage
	ctx = testOptimizeContext(t)
	addOptimizeTestPage(t, ctx)
	err = AddWatermarksSliceMap(ctx, map[int][]*model.Watermark{1: {model.DefaultWatermarkConfig(), imageWM}})
	if !errors.Is(err, ErrMissingImageReader) {
		t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
	}
	if !strings.Contains(err.Error(), "page 1, watermark 1: prepare resources: image") {
		t.Fatalf("expected watermark identity in %q", err.Error())
	}
}

func TestRemoveWatermarksRejectsMissingContext(t *testing.T) {
	if err := RemoveWatermarks(nil, nil); !errors.Is(err, ErrMissingPDFContext) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
	}
	if err := RemoveWatermarks(&model.Context{}, nil); !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
}

func assertEmptyResourceCategory(t *testing.T, ctx *model.Context, resources types.Dict, category string) {
	t.Helper()
	d, err := ctx.DereferenceDict(resources[category])
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || d.Len() != 0 {
		t.Fatalf("expected empty page %s dictionary, got %v", category, d)
	}
}

func assertResourceBinding(t *testing.T, ctx *model.Context, resources types.Dict, category, name string) {
	t.Helper()
	d, err := ctx.DereferenceDict(resources[category])
	if err != nil {
		t.Fatal(err)
	}
	if !d.HasEntry(name) {
		t.Fatalf("expected shared %s resource binding to remain", category)
	}
}

func assertObjectInUse(t *testing.T, ctx *model.Context, indRef *types.IndirectRef) {
	t.Helper()
	entry, found := ctx.FindTableEntryForIndRef(indRef)
	if !found || entry == nil || entry.Free {
		t.Fatal("expected shared resource object to remain in use")
	}
}

func TestRemoveResourceEntryOnlyUnlinksPageNames(t *testing.T) {
	ctx := testOptimizeContext(t)
	gsRef, err := ctx.IndRefForNewObject(types.Dict{"Type": types.Name("ExtGState")})
	if err != nil {
		t.Fatal(err)
	}
	formRef, err := ctx.IndRefForNewObject(types.Dict{"Subtype": types.Name("Form")})
	if err != nil {
		t.Fatal(err)
	}
	pageResources := types.Dict{
		"ExtGState": types.Dict{"GS0": *gsRef},
		"XObject":   types.Dict{"Fm0": *formRef},
	}
	sharedResources := types.Dict{
		"ExtGState": types.Dict{"GS0": *gsRef},
		"XObject":   types.Dict{"Fm0": *formRef},
	}

	if err := removeExtGStates(ctx, pageResources, []string{"GS0"}, 1); err != nil {
		t.Fatal(err)
	}
	if err := removeForms(ctx, pageResources, []string{"Fm0"}, 1); err != nil {
		t.Fatal(err)
	}
	assertEmptyResourceCategory(t, ctx, pageResources, "ExtGState")
	assertEmptyResourceCategory(t, ctx, pageResources, "XObject")
	assertResourceBinding(t, ctx, sharedResources, "ExtGState", "GS0")
	assertResourceBinding(t, ctx, sharedResources, "XObject", "Fm0")
	assertObjectInUse(t, ctx, gsRef)
	assertObjectInUse(t, ctx, formRef)
}

func TestRemoveWatermarksErrorsIncludeOperationAndPageContext(t *testing.T) {
	t.Run("missing optional content", func(t *testing.T) {
		err := RemoveWatermarks(testOptimizeContext(t), nil)
		if !errors.Is(err, errNoWatermark) {
			t.Fatalf("expected %v, got %v", errNoWatermark, err)
		}
		if !strings.Contains(err.Error(), "locate optional content groups") {
			t.Fatalf("expected lookup phase in %q", err.Error())
		}
	})

	t.Run("page contents", func(t *testing.T) {
		ctx := testOptimizeContext(t)
		pageDict := addOptimizeTestPage(t, ctx)
		pageDict["Resources"] = types.Dict{"ExtGState": types.Dict{}, "XObject": types.Dict{}}
		pageDict["Contents"] = *types.NewIndirectRef(999, 0)
		if _, err := prepareOCPropertiesInRoot(ctx, false); err != nil {
			t.Fatal(err)
		}

		err := RemoveWatermarks(ctx, types.IntSet{1: true})
		if err == nil {
			t.Fatal("expected error")
		}
		for _, want := range []string{"remove page watermarks", "page 1", "Contents: content stream obj#999"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q in %q", want, err.Error())
			}
		}
	})
}

func TestRemovePageWatermarksUsesSortedPages(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.PageCount = 2
	err := removePageWatermarks(ctx, types.IntSet{2: true, 1: true})
	if err == nil || !strings.Contains(err.Error(), "page 1") {
		t.Fatalf("expected page 1 error, got %v", err)
	}
}

func TestFindPageWatermarksHandlesMissingAndMalformedContents(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	_, pageRef, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}

	delete(pageDict, "Contents")
	found, err := findPageWatermarks(ctx, pageRef)
	if err != nil || found {
		t.Fatalf("expected contentless page to be ignored, found=%t err=%v", found, err)
	}

	tests := []struct {
		name string
		obj  types.Object
		want string
	}{
		{name: "missing xref", obj: *types.NewIndirectRef(999, 0), want: "Contents: content stream obj#999"},
		{name: "wrong type", obj: types.Name("broken"), want: "Contents: expected stream dictionary or array"},
		{name: "malformed array", obj: types.Array{types.Name("broken")}, want: "content array entry 1: expected indirect reference"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageDict["Contents"] = tt.obj
			_, err := findPageWatermarks(ctx, pageRef)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %v", tt.want, err)
			}
		})
	}
}

func TestFindPageWatermarksPreservesUnsupportedFilter(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	_, pageRef, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Contents"] = types.StreamDict{
		Dict: types.NewDict(),
		Raw:  []byte("encoded"),
		FilterPipeline: []types.PDFFilter{
			{Name: filter.JBIG2},
			{Name: filter.ASCIIHex},
		},
	}

	_, err = findPageWatermarks(ctx, pageRef)
	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "decode content stream") {
		t.Fatalf("expected decode phase in %q", err.Error())
	}
}

func TestDetectWatermarksIncludesPageTreeObjectContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	pageDict := addOptimizeTestPage(t, ctx)
	pageDict["Contents"] = *types.NewIndirectRef(999, 0)
	if _, err := prepareOCPropertiesInRoot(ctx, false); err != nil {
		t.Fatal(err)
	}

	err := DetectWatermarks(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page tree watermarks", "page tree child 1 obj#", "Contents: content stream obj#999"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

var _ io.Writer = stampFailingWriter{}
