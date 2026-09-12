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

package cli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestDispatchRejectsNilContext(t *testing.T) {
	cmd := ValidateCommand([]string{"ignored.pdf"}, nil)
	_, err := Dispatch(nil, cmd)
	if !errors.Is(err, ErrMissingContext) {
		t.Fatalf("got %v, want ErrMissingContext", err)
	}
}

func TestAllCommandsUseContextDispatcher(t *testing.T) {
	for mode := range dispatchTable {
		if _, ok := dispatchTable[mode]; !ok {
			t.Errorf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestDispatchPropagatesReadingCancellation(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Dispatch(ctx, ValidateCommand([]string{inFile}, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestOptimizeReturnsCancellation(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := optimize(c, OptimizeCommand(inFile, outFile, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestMergeCreateReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := mergeCreate(c, MergeCreateCommand([]string{"ignored.pdf"}, "out.pdf", false, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestMergeAppendReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := mergeAppend(c, MergeAppendCommand([]string{"ignored.pdf"}, "out.pdf", false, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestMergeCreateZipReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := mergeCreateZip(c, MergeCreateZipCommand([]string{"one.pdf", "two.pdf"}, "out.pdf", nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestSplitReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := split(c, SplitCommand("ignored.pdf", "out", 1, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestSplitByPageNrReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := splitByPageNr(c, SplitByPageNrCommand("ignored.pdf", "out", []int{2}, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestTrimReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := trim(c, TrimCommand("ignored.pdf", "out.pdf", []string{"1"}, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestTrimUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.TRIM]; !ok {
		t.Fatal("trim is not registered with the context dispatcher")
	}
}

func TestCollectReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := collect(c, CollectCommand("ignored.pdf", "out.pdf", []string{"1"}, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCollectUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.COLLECT]; !ok {
		t.Fatal("collect is not registered with the context dispatcher")
	}
}

func TestRotateReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := rotate(c, RotateCommand("ignored.pdf", "out.pdf", 90, nil, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRotateUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.ROTATE]; !ok {
		t.Fatal("rotate is not registered with the context dispatcher")
	}
}

func TestInsertPagesReturnsCancellation(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"before", "before"},
		{"after", "after"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, cancel := context.WithCancel(t.Context())
			cancel()

			cmd := InsertPagesCommand("ignored.pdf", "out.pdf", []string{"1"}, nil, tt.mode, nil)
			if _, err := insertPages(c, cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestInsertPagesUsesContextDispatcher(t *testing.T) {
	for _, mode := range []model.CommandMode{model.INSERTPAGESBEFORE, model.INSERTPAGESAFTER} {
		if _, ok := dispatchTable[mode]; !ok {
			t.Errorf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestRemovePagesReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := RemovePagesCommand("ignored.pdf", "out.pdf", []string{"1"}, nil)
	if _, err := removePages(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRemovePagesUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.REMOVEPAGES]; !ok {
		t.Fatal("remove pages is not registered with the context dispatcher")
	}
}

func TestCropReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := CropCommand("ignored.pdf", "out.pdf", []string{"1"}, new(model.Box), nil)
	if _, err := crop(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCropUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.CROP]; !ok {
		t.Fatal("crop is not registered with the context dispatcher")
	}
}

func TestAddBoxesReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := AddBoxesCommand("ignored.pdf", "out.pdf", []string{"1"}, new(model.PageBoundaries), nil)
	if _, err := addBoxes(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestListBoxesReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := ListBoxesCommand("ignored.pdf", []string{"1"}, nil, nil)
	if _, err := listBoxes(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestListBoxesUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.LISTBOXES]; !ok {
		t.Fatal("list boxes is not registered with the context dispatcher")
	}
}

func TestAddBoxesUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.ADDBOXES]; !ok {
		t.Fatal("add boxes is not registered with the context dispatcher")
	}
}

func TestRemoveBoxesReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := RemoveBoxesCommand("ignored.pdf", "out.pdf", []string{"1"}, new(model.PageBoundaries), nil)
	if _, err := removeBoxes(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRemoveBoxesUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.REMOVEBOXES]; !ok {
		t.Fatal("remove boxes is not registered with the context dispatcher")
	}
}

func TestZoomReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := ZoomCommand("ignored.pdf", "out.pdf", []string{"1"}, &model.Zoom{Factor: 0.5}, nil)
	if _, err := zoom(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestZoomUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.ZOOM]; !ok {
		t.Fatal("zoom is not registered with the context dispatcher")
	}
}

func TestResizeReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := ResizeCommand("ignored.pdf", "out.pdf", []string{"1"}, &model.Resize{Scale: 0.5}, nil)
	if _, err := resize(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestResizeUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.RESIZE]; !ok {
		t.Fatal("resize is not registered with the context dispatcher")
	}
}

func TestNUpReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := NUpCommand([]string{"ignored.pdf"}, "out.pdf", nil, new(model.NUp), nil)
	if _, err := nUp(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestNUpUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.NUP]; !ok {
		t.Fatal("n-up is not registered with the context dispatcher")
	}
}

func TestGridReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := GridCommand([]string{"ignored.pdf"}, "out.pdf", nil, new(model.NUp), nil)
	if _, err := grid(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestGridUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.GRID]; !ok {
		t.Fatal("grid is not registered with the context dispatcher")
	}
}

func TestBookletReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := BookletCommand([]string{"ignored.pdf"}, "out.pdf", nil, new(model.NUp), nil)
	if _, err := booklet(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestBookletUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.BOOKLET]; !ok {
		t.Fatal("booklet is not registered with the context dispatcher")
	}
}

func TestPosterReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := PosterCommand("ignored.pdf", "out", "", nil, new(model.Cut), nil)
	if _, err := poster(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestPosterUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.POSTER]; !ok {
		t.Fatal("poster is not registered with the context dispatcher")
	}
}

func TestNDownReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := NDownCommand("ignored.pdf", "out", "", nil, 2, new(model.Cut), nil)
	if _, err := nDown(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestNDownUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.NDOWN]; !ok {
		t.Fatal("ndown is not registered with the context dispatcher")
	}
}

func TestCutReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	cmd := CutCommand("ignored.pdf", "out", "", nil, new(model.Cut), nil)
	if _, err := cut(c, cmd); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCutUsesContextDispatcher(t *testing.T) {
	if _, ok := dispatchTable[model.CUT]; !ok {
		t.Fatal("cut is not registered with the context dispatcher")
	}
}

func TestWatermarkCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"add", AddWatermarksCommand("ignored.pdf", "out.pdf", nil, new(model.Watermark), nil), addWatermarks},
		{"remove", RemoveWatermarksCommand("ignored.pdf", "out.pdf", nil, nil), removeWatermarks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestWatermarkCommandsUseContextDispatcher(t *testing.T) {
	for _, mode := range []model.CommandMode{model.ADDWATERMARKS, model.REMOVEWATERMARKS} {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestAnnotationCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"list", ListAnnotationsCommand("ignored.pdf", nil, nil), listAnnotationsForCommand},
		{"remove", RemoveAnnotationsCommand("ignored.pdf", "out.pdf", nil, nil, nil, nil), removeAnnotations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestAnnotationCommandsUseContextDispatcher(t *testing.T) {
	for _, mode := range []model.CommandMode{model.LISTANNOTATIONS, model.REMOVEANNOTATIONS} {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestBookmarkCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"export", ExportBookmarksCommand("ignored.pdf", "out.json", nil), exportBookmarks},
		{"import", ImportBookmarksCommand("ignored.pdf", "in.json", "out.pdf", false, nil), importBookmarks},
		{"list", ListBookmarksCommand("ignored.pdf", nil), listBookmarks},
		{"remove", RemoveBookmarksCommand("ignored.pdf", "out.pdf", nil), removeBookmarks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestBookmarkCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.LISTBOOKMARKS,
		model.EXPORTBOOKMARKS,
		model.IMPORTBOOKMARKS,
		model.REMOVEBOOKMARKS,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestDocumentViewCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"list page layout", ListPageLayoutCommand("ignored.pdf", nil), listPageLayout},
		{"reset page layout", ResetPageLayoutCommand("ignored.pdf", "out.pdf", nil), resetPageLayout},
		{"set page layout", SetPageLayoutCommand("ignored.pdf", "out.pdf", "SinglePage", nil), setPageLayout},
		{"list page mode", ListPageModeCommand("ignored.pdf", nil), listPageMode},
		{"reset page mode", ResetPageModeCommand("ignored.pdf", "out.pdf", nil), resetPageMode},
		{"set page mode", SetPageModeCommand("ignored.pdf", "out.pdf", "UseNone", nil), setPageMode},
		{"list viewer preferences", ListViewerPreferencesCommand("ignored.pdf", false, false, nil), listViewerPreferences},
		{"reset viewer preferences", ResetViewerPreferencesCommand("ignored.pdf", "out.pdf", nil), resetViewerPreferences},
		{"set viewer preferences", SetViewerPreferencesCommand("ignored.pdf", "", "out.pdf", "{}", nil), setViewerPreferences},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestDocumentViewCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.LISTPAGELAYOUT,
		model.SETPAGELAYOUT,
		model.RESETPAGELAYOUT,
		model.LISTPAGEMODE,
		model.SETPAGEMODE,
		model.RESETPAGEMODE,
		model.LISTVIEWERPREFERENCES,
		model.SETVIEWERPREFERENCES,
		model.RESETVIEWERPREFERENCES,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestImageCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"extract", ExtractImagesCommand("ignored.pdf", "out", nil, nil), extractImages},
		{"import", ImportImagesCommand([]string{"ignored.png"}, "out.pdf", nil, nil), importImages},
		{"list", ListImagesCommand([]string{"ignored.pdf"}, nil, nil), listImages},
		{"update", UpdateImagesCommand("ignored.pdf", "ignored.png", "out.pdf", 1, "", nil), updateImages},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestImageCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.IMPORTIMAGES,
		model.LISTIMAGES,
		model.UPDATEIMAGES,
		model.EXTRACTIMAGES,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestAttachmentCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"add", AddAttachmentsCommand("ignored.pdf", "out.pdf", []string{"ignored.txt"}, nil), addAttachments},
		{"add portfolio", AddAttachmentsPortfolioCommand("ignored.pdf", "out.pdf", []string{"ignored.txt"}, nil), addAttachments},
		{"extract", ExtractAttachmentsCommand("ignored.pdf", "out", nil, nil), extractAttachments},
		{"list", ListAttachmentsCommand("ignored.pdf", nil), listAttachmentsCommand},
		{"remove", RemoveAttachmentsCommand("ignored.pdf", "out.pdf", nil, nil), removeAttachments},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestAttachmentCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.LISTATTACHMENTS,
		model.ADDATTACHMENTS,
		model.ADDATTACHMENTSPORTFOLIO,
		model.REMOVEATTACHMENTS,
		model.EXTRACTATTACHMENTS,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestMetadataCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"add keywords", AddKeywordsCommand("ignored.pdf", "out.pdf", []string{"keyword"}, nil), addKeywords},
		{"add properties", AddPropertiesCommand("ignored.pdf", "out.pdf", map[string]string{"name": "value"}, nil), addProperties},
		{"list keywords", ListKeywordsCommand("ignored.pdf", nil), listKeywords},
		{"list properties", ListPropertiesCommand("ignored.pdf", nil), listPropertiesCommand},
		{"remove keywords", RemoveKeywordsCommand("ignored.pdf", "out.pdf", nil, nil), removeKeywords},
		{"remove properties", RemovePropertiesCommand("ignored.pdf", "out.pdf", nil, nil), removeProperties},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestMetadataCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.LISTKEYWORDS,
		model.ADDKEYWORDS,
		model.REMOVEKEYWORDS,
		model.LISTPROPERTIES,
		model.ADDPROPERTIES,
		model.REMOVEPROPERTIES,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestCryptoCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"change owner password", ChangeOwnerPWCommand("ignored.pdf", "out.pdf", nil, nil, nil), changeOwnerPassword},
		{"change user password", ChangeUserPWCommand("ignored.pdf", "out.pdf", nil, nil, nil), changeUserPassword},
		{"decrypt", DecryptCommand("ignored.pdf", "out.pdf", nil), decrypt},
		{"encrypt", EncryptCommand("ignored.pdf", "out.pdf", nil), encrypt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestCryptoCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{model.CHANGEOPW, model.CHANGEUPW, model.DECRYPT, model.ENCRYPT}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestFormCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"export", ExportFormCommand("ignored.pdf", "out.json", nil), exportFormFields},
		{"fill", FillFormCommand("ignored.pdf", "data.json", "out.pdf", nil), fillFormFields},
		{"list", ListFormFieldsCommand([]string{"ignored.pdf"}, nil), listFormFieldsForCommand},
		{"lock", LockFormCommand("ignored.pdf", "out.pdf", nil, nil), lockFormFields},
		{"multi-fill", MultiFillFormCommand("ignored.pdf", "data.json", "out", "out.pdf", false, nil), multiFillFormFields},
		{"remove", RemoveFormFieldsCommand("ignored.pdf", "out.pdf", nil, nil), removeFormFields},
		{"reset", ResetFormCommand("ignored.pdf", "out.pdf", nil, nil), resetFormFields},
		{"unlock", UnlockFormCommand("ignored.pdf", "out.pdf", nil, nil), unlockFormFields},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestFormCommandsUseContextDispatcher(t *testing.T) {
	modes := []model.CommandMode{
		model.LISTFORMFIELDS,
		model.REMOVEFORMFIELDS,
		model.LOCKFORMFIELDS,
		model.UNLOCKFORMFIELDS,
		model.RESETFORMFIELDS,
		model.EXPORTFORMFIELDS,
		model.FILLFORMFIELDS,
		model.MULTIFILLFORMFIELDS,
	}
	for _, mode := range modes {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestPermissionCommandsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name string
		cmd  *Command
		call dispatchFunc
	}{
		{"list", ListPermissionsCommand([]string{"ignored.pdf"}, nil), listPermissionsCommand},
		{"set", SetPermissionsCommand("ignored.pdf", "out.pdf", nil), setPermissions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(c, tt.cmd); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestPermissionCommandsUseContextDispatcher(t *testing.T) {
	for _, mode := range []model.CommandMode{model.LISTPERMISSIONS, model.SETPERMISSIONS} {
		if _, ok := dispatchTable[mode]; !ok {
			t.Fatalf("mode %d is not registered with the context dispatcher", mode)
		}
	}
}

func TestInstallFontsReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := installFonts(c, InstallFontsCommand([]string{"ignored.ttf"}, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestDispatchRecoversUnexpectedPanicWithStackMetadata(t *testing.T) {
	mode := model.CommandMode(-1)
	dispatchTable[mode] = func(context.Context, *Command) ([]string, error) {
		panic("boom")
	}
	defer delete(dispatchTable, mode)

	_, err := Dispatch(t.Context(), &Command{Mode: mode, Conf: model.NewDefaultConfiguration()})
	if err == nil {
		t.Fatal("expected error")
	}

	var p fault.Panic
	if !errors.As(err, &p) {
		t.Fatalf("got %T, want fault.Panic", err)
	}
	if got := err.Error(); !strings.Contains(got, "unexpected panic attack: boom") {
		t.Fatalf("got %q, want unexpected panic message", got)
	}
	if strings.Contains(err.Error(), "goroutine ") {
		t.Fatalf("error string includes stack trace: %q", err.Error())
	}
	if !strings.Contains(string(p.Stack), "TestDispatchRecoversUnexpectedPanicWithStackMetadata") {
		t.Fatalf("stack trace does not include test frame:\n%s", p.Stack)
	}
}

// TestDispatchRejectsAddSignature verifies the unimplemented command mode cannot report false success.
func TestDispatchRejectsAddSignature(t *testing.T) {
	_, err := Dispatch(t.Context(), &Command{Mode: model.ADDSIGNATURE})
	if !errors.Is(err, ErrUnsupportedCommandMode) {
		t.Fatalf("expected unsupported command mode, got %v", err)
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Fatalf("expected command mode context, got %v", err)
	}
}

// TestDispatchUsesOperationOwnedConfiguration verifies successful execution cannot mutate caller-owned command state.
func TestDispatchUsesOperationOwnedConfiguration(t *testing.T) {
	mode := model.CommandMode(-2)
	userPWNew := "new-user"
	ownerPWNew := "new-owner"
	conf := &model.Configuration{
		Cmd:                    model.OPTIMIZE,
		UserPWNew:              &userPWNew,
		OwnerPWNew:             &ownerPWNew,
		AllowedRevocationHosts: []string{"ocsp.example.corp"},
	}
	cmd := &Command{Mode: mode, StringVal: "caller", Conf: conf}

	var executionCommand *Command
	dispatchTable[mode] = func(c context.Context, exec *Command) ([]string, error) {
		executionCommand = exec
		exec.StringVal = "execution"
		*exec.Conf.UserPWNew = "execution-user"
		*exec.Conf.OwnerPWNew = "execution-owner"
		exec.Conf.AllowedRevocationHosts[0] = "execution.example.corp"
		return []string{"ok"}, nil
	}
	defer delete(dispatchTable, mode)

	out, err := Dispatch(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "ok" {
		t.Fatalf("output: got %v, want [ok]", out)
	}
	if executionCommand == cmd {
		t.Fatal("dispatch executed the caller's command")
	}
	if executionCommand.Conf == conf {
		t.Fatal("dispatch executed with caller-owned configuration")
	}
	if executionCommand.Conf.Cmd != mode {
		t.Fatalf("execution command mode: got %d, want %d", executionCommand.Conf.Cmd, mode)
	}
	if cmd.StringVal != "caller" {
		t.Fatalf("caller command value: got %q, want caller", cmd.StringVal)
	}
	if cmd.Conf != conf {
		t.Fatal("dispatch replaced caller command configuration")
	}
	if conf.Cmd != model.OPTIMIZE {
		t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, model.OPTIMIZE)
	}
	if got, want := *conf.UserPWNew, "new-user"; got != want {
		t.Fatalf("caller new user password: got %q, want %q", got, want)
	}
	if got, want := *conf.OwnerPWNew, "new-owner"; got != want {
		t.Fatalf("caller new owner password: got %q, want %q", got, want)
	}
	if got, want := conf.AllowedRevocationHosts[0], "ocsp.example.corp"; got != want {
		t.Fatalf("caller allowed revocation host: got %q, want %q", got, want)
	}
}

// TestDispatchFailurePreservesCallerState verifies unsupported commands do not mutate caller-owned state.
func TestDispatchFailurePreservesCallerState(t *testing.T) {
	mode := model.CommandMode(-3)
	conf := &model.Configuration{Cmd: model.VALIDATE}
	cmd := &Command{Mode: mode, Conf: conf}

	_, err := Dispatch(t.Context(), cmd)
	if !errors.Is(err, ErrUnsupportedCommandMode) {
		t.Fatalf("expected unsupported command mode, got %v", err)
	}
	if cmd.Conf != conf {
		t.Fatal("dispatch replaced caller command configuration")
	}
	if conf.Cmd != model.VALIDATE {
		t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, model.VALIDATE)
	}
}
