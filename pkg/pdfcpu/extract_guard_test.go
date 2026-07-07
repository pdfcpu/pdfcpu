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
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func emptyStreamDict() *types.StreamDict {
	return &types.StreamDict{Dict: types.Dict{}}
}

// TestExtractFunctionsRejectMissingInput verifies stable extraction precondition errors.
func TestExtractFunctionsRejectMissingInput(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "stream length missing context",
			fn: func() error {
				_, err := StreamLength(nil, emptyStreamDict())
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "stream length missing stream dict",
			fn: func() error {
				_, err := StreamLength(ctx, nil)
				return err
			},
			wantErr: ErrMissingStreamDict,
		},
		{
			name: "colorspace string missing stream dict",
			fn: func() error {
				_, err := ColorSpaceString(ctx, nil)
				return err
			},
			wantErr: ErrMissingStreamDict,
		},
		{
			name: "colorspace components missing xref table",
			fn: func() error {
				_, err := ColorSpaceComponents(nil, emptyStreamDict())
				return err
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "extract image missing context",
			fn: func() error {
				_, err := ExtractImage(nil, emptyStreamDict(), false, "", 1, true)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "extract page images missing optimization context",
			fn: func() error {
				_, err := ExtractPageImages(ctx, 1, true)
				return err
			},
			wantErr: ErrMissingOptimizationContext,
		},
		{
			name: "extract font missing context",
			fn: func() error {
				_, err := ExtractFont(nil, model.FontObject{}, 1)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "extract page fonts missing optimization context",
			fn: func() error {
				_, err := ExtractPageFonts(ctx, 1, nil, nil)
				return err
			},
			wantErr: ErrMissingOptimizationContext,
		},
		{
			name: "extract form fonts missing optimization context",
			fn: func() error {
				_, err := ExtractFormFonts(ctx)
				return err
			},
			wantErr: ErrMissingOptimizationContext,
		},
		{
			name: "extract page content missing context",
			fn: func() error {
				_, err := ExtractPageContent(nil, 1)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "extract metadata missing context",
			fn: func() error {
				_, err := ExtractMetadata(nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestExtractImageRejectsNilStream verifies the corresponding behavior.
func TestExtractImageRejectsNilStream(t *testing.T) {
	_, err := ExtractImage(nil, nil, false, "", 1, true)
	if !errors.Is(err, ErrMissingStreamDict) {
		t.Fatalf("expected %v, got %v", ErrMissingStreamDict, err)
	}
}

// TestDereferenceRequiredStreamDictPreservesMissingStreamDict verifies the corresponding behavior.
func TestDereferenceRequiredStreamDictPreservesMissingStreamDict(t *testing.T) {
	indRef := types.NewIndirectRef(1, 0)
	_, err := dereferenceRequiredStreamDict(&model.XRefTable{}, *indRef, "image")
	if !errors.Is(err, ErrMissingStreamDict) {
		t.Fatalf("expected %v, got %v", ErrMissingStreamDict, err)
	}
	if !strings.Contains(err.Error(), "image") {
		t.Fatalf("expected image context, got %q", err.Error())
	}
}

// TestObjNrsRejectMissingOptimizationContext verifies the corresponding behavior.
func TestObjNrsRejectMissingOptimizationContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	if objNrs := ImageObjNrs(ctx, 1); len(objNrs) != 0 {
		t.Fatalf("expected no image objects, got %v", objNrs)
	}
	if objNrs := FontObjNrs(ctx, 1); len(objNrs) != 0 {
		t.Fatalf("expected no font objects, got %v", objNrs)
	}
}

// TestExtractionObjectNumbersAreSorted verifies the corresponding behavior.
func TestExtractionObjectNumbersAreSorted(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{},
		Optimize: &model.OptimizationContext{
			PageImages: []types.IntSet{{9: true, 2: true, 4: false}},
			PageFonts:  []types.IntSet{{8: true, 3: true, 5: false}},
		},
	}

	if got, want := ImageObjNrs(ctx, 1), []int{2, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image object numbers: got %v, want %v", got, want)
	}
	if got, want := FontObjNrs(ctx, 1), []int{3, 8}; !reflect.DeepEqual(got, want) {
		t.Fatalf("font object numbers: got %v, want %v", got, want)
	}
}

// TestExtractMetadataProcessesParentObjectsInOrder verifies the corresponding behavior.
func TestExtractMetadataProcessesParentObjectsInOrder(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		3:  model.NewXRefTableEntryGen0(types.Dict{"Metadata": *types.NewIndirectRef(31, 0)}),
		9:  model.NewXRefTableEntryGen0(types.Dict{"Metadata": *types.NewIndirectRef(91, 0)}),
		31: model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("three")}),
		91: model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("nine")}),
	}}}

	metadata, err := ExtractMetadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) != 2 {
		t.Fatalf("got %d metadata entries, want 2", len(metadata))
	}
	if got, want := []int{metadata[0].ParentObjNr, metadata[1].ParentObjNr}, []int{3, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata parent object numbers: got %v, want %v", got, want)
	}
}

// TestExtractionSelectsDeterministicFirstError verifies the corresponding behavior.
func TestExtractionSelectsDeterministicFirstError(t *testing.T) {
	tests := []struct {
		name string
		want string
		fn   func() error
	}{
		{
			name: "image object",
			want: "page 1 image obj#2",
			fn: func() error {
				ctx := &model.Context{
					XRefTable: &model.XRefTable{PageCount: 1},
					Optimize: &model.OptimizationContext{
						PageImages: []types.IntSet{{9: true, 2: true}},
					},
				}
				_, err := ExtractPageImages(ctx, 1, true)
				return err
			},
		},
		{
			name: "font object",
			want: "page 1 font obj#3",
			fn: func() error {
				ctx := &model.Context{
					XRefTable: &model.XRefTable{PageCount: 1},
					Optimize: &model.OptimizationContext{
						PageFonts: []types.IntSet{{8: true, 3: true}},
					},
				}
				_, err := ExtractPageFonts(ctx, 1, nil, nil)
				return err
			},
		},
		{
			name: "form font object",
			want: "form font obj#2",
			fn: func() error {
				ctx := &model.Context{
					XRefTable: &model.XRefTable{},
					Optimize: &model.OptimizationContext{
						FormFontObjects: map[int]*model.FontObject{9: nil, 2: nil},
					},
				}
				_, err := ExtractFormFonts(ctx)
				return err
			},
		},
		{
			name: "metadata parent",
			want: "metadata parent obj#3",
			fn: func() error {
				ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
					3: model.NewXRefTableEntryGen0(types.Dict{"Metadata": types.StreamDict{Dict: types.Dict{}}}),
					9: model.NewXRefTableEntryGen0(types.Dict{"Metadata": types.StreamDict{Dict: types.Dict{}}}),
				}}}
				_, err := ExtractMetadata(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				err := tt.fn()
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("iteration %d: expected %q, got %v", i, tt.want, err)
				}
			}
		})
	}
}

// TestUnsupportedExtractionResourcesReturnSentinel verifies the corresponding behavior.
func TestUnsupportedExtractionResourcesReturnSentinel(t *testing.T) {
	unsupportedPipeline := []types.PDFFilter{
		{Name: filter.JBIG2},
		{Name: filter.Flate},
	}

	tests := []struct {
		name       string
		want       string
		wantFilter bool
		fn         func() error
	}{
		{
			name: "image filter",
			want: "image obj#7",
			fn: func() error {
				ctx := &model.Context{XRefTable: &model.XRefTable{}}
				sd := &types.StreamDict{Dict: types.Dict{}, FilterPipeline: []types.PDFFilter{{Name: "Unsupported"}}}
				_, err := ExtractImage(ctx, sd, false, "Im0", 7, false)
				return err
			},
		},
		{
			name: "image rendering colorspace",
			want: "image obj#7 render colorspace [CalGray",
			fn: func() error {
				ctx := &model.Context{XRefTable: &model.XRefTable{}}
				sd := &types.StreamDict{
					Dict: types.Dict{
						"BitsPerComponent": types.Integer(8),
						"ColorSpace":       types.Array{types.Name(model.CalGrayCS), types.Dict{}},
						"Height":           types.Integer(1),
						"Width":            types.Integer(1),
					},
					Raw: []byte{0},
				}
				_, err := ExtractImage(ctx, sd, false, "Im0", 7, false)
				return err
			},
		},
		{
			name: "font type",
			want: "font \"OddFont\" obj#8",
			fn: func() error {
				ctx := &model.Context{XRefTable: &model.XRefTable{}}
				fontObject := model.FontObject{
					FontName: "OddFont",
					FontDict: types.Dict{
						"Subtype":        types.Name("Type1"),
						"FontDescriptor": types.Dict{"FontFile": *types.NewIndirectRef(80, 0)},
					},
				}
				_, err := ExtractFont(ctx, fontObject, 8)
				return err
			},
		},
		{
			name:       "metadata filter",
			want:       "metadata parent obj#9: metadata obj#90",
			wantFilter: true,
			fn: func() error {
				ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
					9: model.NewXRefTableEntryGen0(types.Dict{"Metadata": *types.NewIndirectRef(90, 0)}),
					90: model.NewXRefTableEntryGen0(types.StreamDict{
						Dict:           types.Dict{},
						Raw:            []byte("metadata"),
						FilterPipeline: unsupportedPipeline,
					}),
				}}}
				_, err := ExtractMetadata(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrUnsupportedResource) {
				t.Fatalf("expected %v, got %v", ErrUnsupportedResource, err)
			}
			if tt.wantFilter && !errors.Is(err, filter.ErrUnsupportedFilter) {
				t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestExtractMetadataUnsupportedResourcePolicyControlsContinuation verifies metadata skip and fail extraction behavior.
func TestExtractMetadataUnsupportedResourcePolicyControlsContinuation(t *testing.T) {
	unsupportedPipeline := []types.PDFFilter{{Name: filter.JBIG2}, {Name: filter.Flate}}
	newContext := func(policy model.UnsupportedResourcePolicy) *model.Context {
		conf := model.NewDefaultConfiguration()
		conf.UnsupportedResourcePolicy = policy
		return &model.Context{
			Configuration: conf,
			XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
				3:  model.NewXRefTableEntryGen0(types.Dict{"Metadata": *types.NewIndirectRef(30, 0)}),
				9:  model.NewXRefTableEntryGen0(types.Dict{"Metadata": *types.NewIndirectRef(90, 0)}),
				30: model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("first"), FilterPipeline: unsupportedPipeline}),
				90: model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("second"), FilterPipeline: unsupportedPipeline}),
			}},
		}
	}

	_, err := ExtractMetadata(newContext(model.UnsupportedResourceSkip))
	if !errors.Is(err, ErrUnsupportedResource) || !strings.Contains(err.Error(), "metadata parent obj#9: metadata obj#90") {
		t.Fatalf("skip policy must report all unsupported resources, got %v", err)
	}

	_, err = ExtractMetadata(newContext(model.UnsupportedResourceFail))
	if !errors.Is(err, ErrUnsupportedResource) {
		t.Fatalf("fail policy must preserve %v, got %v", ErrUnsupportedResource, err)
	}
	if strings.Contains(err.Error(), "metadata parent obj#9: metadata obj#90") {
		t.Fatalf("fail policy must stop at the first unsupported resource, got %v", err)
	}
}

func assertUnsupportedResourcePolicyContinuation(
	t *testing.T,
	secondResource string,
	extract func(model.UnsupportedResourcePolicy) error,
) {
	t.Helper()

	err := extract(model.UnsupportedResourceSkip)
	if !errors.Is(err, ErrUnsupportedResource) || !strings.Contains(err.Error(), secondResource) {
		t.Fatalf("skip policy must report all unsupported resources, got %v", err)
	}

	err = extract(model.UnsupportedResourceFail)
	if !errors.Is(err, ErrUnsupportedResource) {
		t.Fatalf("fail policy must preserve %v, got %v", ErrUnsupportedResource, err)
	}
	if strings.Contains(err.Error(), secondResource) {
		t.Fatalf("fail policy must stop at the first unsupported resource, got %v", err)
	}
}

func extractionPolicyConfiguration(policy model.UnsupportedResourcePolicy) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.UnsupportedResourcePolicy = policy
	return conf
}

// TestExtractPageImagesUnsupportedResourcePolicyControlsContinuation verifies image skip and fail extraction behavior.
func TestExtractPageImagesUnsupportedResourcePolicyControlsContinuation(t *testing.T) {
	extract := func(policy model.UnsupportedResourcePolicy) error {
		imageDict := func() *types.StreamDict {
			return &types.StreamDict{
				Dict:           types.Dict{},
				FilterPipeline: []types.PDFFilter{{Name: "Unsupported"}},
			}
		}
		ctx := &model.Context{
			Configuration: extractionPolicyConfiguration(policy),
			XRefTable:     &model.XRefTable{PageCount: 1},
			Optimize: &model.OptimizationContext{
				PageImages: []types.IntSet{{2: true, 9: true}},
				ImageObjects: map[int]*model.ImageObject{
					2: {ResourceNames: map[int]string{0: "Im2"}, ImageDict: imageDict()},
					9: {ResourceNames: map[int]string{0: "Im9"}, ImageDict: imageDict()},
				},
			},
		}
		_, err := ExtractPageImages(ctx, 1, false)
		return err
	}

	assertUnsupportedResourcePolicyContinuation(t, "image obj#9", extract)
}

func unsupportedFontObject(name string, fileObjNr int) *model.FontObject {
	return &model.FontObject{
		FontName: name,
		FontDict: types.Dict{
			"Subtype":        types.Name("Type1"),
			"FontDescriptor": types.Dict{"FontFile": *types.NewIndirectRef(fileObjNr, 0)},
		},
	}
}

// TestExtractPageFontsUnsupportedResourcePolicyControlsContinuation verifies page-font skip and fail extraction behavior.
func TestExtractPageFontsUnsupportedResourcePolicyControlsContinuation(t *testing.T) {
	extract := func(policy model.UnsupportedResourcePolicy) error {
		ctx := &model.Context{
			Configuration: extractionPolicyConfiguration(policy),
			XRefTable:     &model.XRefTable{PageCount: 1},
			Optimize: &model.OptimizationContext{
				PageFonts: []types.IntSet{{2: true, 9: true}},
				FontObjects: map[int]*model.FontObject{
					2: unsupportedFontObject("FirstFont", 20),
					9: unsupportedFontObject("SecondFont", 90),
				},
			},
		}
		_, err := ExtractPageFonts(ctx, 1, nil, nil)
		return err
	}

	assertUnsupportedResourcePolicyContinuation(t, "font \"SecondFont\" obj#9", extract)
}

// TestExtractFormFontsUnsupportedResourcePolicyControlsContinuation verifies form-font skip and fail extraction behavior.
func TestExtractFormFontsUnsupportedResourcePolicyControlsContinuation(t *testing.T) {
	extract := func(policy model.UnsupportedResourcePolicy) error {
		ctx := &model.Context{
			Configuration: extractionPolicyConfiguration(policy),
			XRefTable:     &model.XRefTable{},
			Optimize: &model.OptimizationContext{
				FormFontObjects: map[int]*model.FontObject{
					2: unsupportedFontObject("FirstFormFont", 20),
					9: unsupportedFontObject("SecondFormFont", 90),
				},
			},
		}
		_, err := ExtractFormFonts(ctx)
		return err
	}

	assertUnsupportedResourcePolicyContinuation(t, "font \"SecondFormFont\" obj#9", extract)
}

// TestExtractPageImagesUsesOneBasedResourceNames verifies the corresponding behavior.
func TestExtractPageImagesUsesOneBasedResourceNames(t *testing.T) {
	streamLength := int64(0)
	ctx := &model.Context{
		XRefTable: &model.XRefTable{PageCount: 2},
		Optimize: &model.OptimizationContext{
			PageImages: []types.IntSet{
				{},
				{16: true},
			},
			ImageObjects: map[int]*model.ImageObject{
				16: {
					ResourceNames: map[int]string{1: "Im1"},
					ImageDict: &types.StreamDict{
						Dict: types.Dict{
							"Subtype": types.Name("Image"),
							"Width":   types.Integer(1),
							"Height":  types.Integer(1),
							"Length":  types.Integer(0),
						},
						StreamLength: &streamLength,
					},
				},
			},
		},
	}

	images, err := ExtractPageImages(ctx, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := images[16]; !ok {
		t.Fatalf("expected image obj#16, got %v", images)
	}
}

// TestExtractMetadataRejectsDirectMetadataStream verifies the corresponding behavior.
func TestExtractMetadataRejectsDirectMetadataStream(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	d := types.Dict{"Metadata": types.StreamDict{Dict: types.Dict{}}}

	_, err := extractMetadataFromDict(ctx, d, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "metadata: expected indirect ref"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestExtractPageImagesThumbnailErrorsIncludePageContext verifies the corresponding behavior.
func TestExtractPageImagesThumbnailErrorsIncludePageContext(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			PageCount:  1,
			PageThumbs: map[int]types.IndirectRef{1: *types.NewIndirectRef(7, 0)},
		},
		Optimize: &model.OptimizationContext{},
	}

	_, err := ExtractPageImages(ctx, 1, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "page 1: thumbnail obj#7"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestExtractFormFontsErrorsIncludeObjectContext verifies the corresponding behavior.
func TestExtractFormFontsErrorsIncludeObjectContext(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{},
		Optimize: &model.OptimizationContext{
			FormFontObjects: map[int]*model.FontObject{
				9: {FontDict: types.Dict{"FontDescriptor": types.Integer(1)}},
			},
		},
	}

	_, err := ExtractFormFonts(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "form: font obj#9 descriptor"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestExtractPageContentErrorsIncludePageContext verifies the corresponding behavior.
func TestExtractPageContentErrorsIncludePageContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}

	_, err := ExtractPageContent(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "page 1: page dict"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestExtractMetadataErrorsIncludeParentObjectContext verifies the corresponding behavior.
func TestExtractMetadataErrorsIncludeParentObjectContext(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				12: model.NewXRefTableEntryGen0(types.Dict{"Metadata": types.StreamDict{Dict: types.Dict{}}}),
			},
		},
	}

	_, err := ExtractMetadata(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "metadata parent obj#12"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
	if want := "metadata: expected indirect ref"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestColorSpaceDereferenceErrorsIncludeEntryContext verifies the corresponding behavior.
func TestColorSpaceDereferenceErrorsIncludeEntryContext(t *testing.T) {
	lazyObject := types.NewLazyObjectStreamObject(&types.ObjectStreamDict{}, -1, 0, nil)
	xRefTable := &model.XRefTable{
		Table: map[int]*model.XRefTableEntry{
			7: model.NewXRefTableEntryGen0(lazyObject),
		},
	}
	ctx := &model.Context{XRefTable: xRefTable}
	sd := &types.StreamDict{
		Dict: types.Dict{"ColorSpace": *types.NewIndirectRef(7, 0)},
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "string",
			fn: func() error {
				_, err := ColorSpaceString(ctx, sd)
				return err
			},
		},
		{
			name: "components",
			fn: func() error {
				_, err := ColorSpaceComponents(xRefTable, sd)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if want := "colorspace obj#7: dereference"; !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err.Error())
			}
			if want := "obj#7"; !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err.Error())
			}
		})
	}
}

// TestDecodeImageColorSpaceErrorsIncludeObjectContext verifies the corresponding behavior.
func TestDecodeImageColorSpaceErrorsIncludeObjectContext(t *testing.T) {
	lazyObject := types.NewLazyObjectStreamObject(&types.ObjectStreamDict{}, -1, 0, nil)
	ctx := &model.Context{XRefTable: &model.XRefTable{
		Table: map[int]*model.XRefTableEntry{
			7: model.NewXRefTableEntryGen0(lazyObject),
		},
	}}
	sd := &types.StreamDict{
		Dict: types.Dict{"ColorSpace": *types.NewIndirectRef(7, 0)},
	}

	err := decodeImage(ctx, sd, filter.DCT, filter.DCT, 11)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "image obj#11 colorspace components"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestExtractFontDescriptorErrorsIncludeObjectContext verifies the corresponding behavior.
func TestExtractFontDescriptorErrorsIncludeObjectContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}}
	fontObject := model.FontObject{
		FontDict: types.Dict{"FontDescriptor": *types.NewIndirectRef(7, 0)},
	}

	_, err := ExtractFont(ctx, fontObject, 9)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "font obj#9 descriptor"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}
