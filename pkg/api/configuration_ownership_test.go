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

package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const callerMode = model.CommandMode(-1)

type configurationOperation struct {
	name string
	run  func(*model.Configuration) error
}

type configurationReuseState struct {
	cmd                             model.CommandMode
	validationMode                  int
	optimizeDuplicateContentStreams bool
	createBookmarks                 bool
	userPW                          string
	ownerPW                         string
	userPWNew                       *string
	userPWNewValue                  string
	ownerPWNew                      *string
	ownerPWNewValue                 string
	allowedRevocationHosts          int
	allowedRevocationHost0          string
	allowedRevocationHost1          string
	allowedRevocationHost0Address   *string
}

func reusableConfiguration() *model.Configuration {
	userPWNew := "configured-new-user"
	ownerPWNew := "configured-new-owner"
	return &model.Configuration{
		Cmd:                             callerMode,
		ValidationMode:                  model.ValidationStrict,
		OptimizeDuplicateContentStreams: true,
		CreateBookmarks:                 true,
		UserPW:                          "configured-user",
		OwnerPW:                         "configured-owner",
		UserPWNew:                       &userPWNew,
		OwnerPWNew:                      &ownerPWNew,
		AllowedRevocationHosts:          []string{"ocsp.example.corp", "crl.example.corp"},
	}
}

func reusableConfigurationState(conf *model.Configuration) configurationReuseState {
	state := configurationReuseState{
		cmd:                             conf.Cmd,
		validationMode:                  conf.ValidationMode,
		optimizeDuplicateContentStreams: conf.OptimizeDuplicateContentStreams,
		createBookmarks:                 conf.CreateBookmarks,
		userPW:                          conf.UserPW,
		ownerPW:                         conf.OwnerPW,
		userPWNew:                       conf.UserPWNew,
		ownerPWNew:                      conf.OwnerPWNew,
		allowedRevocationHosts:          len(conf.AllowedRevocationHosts),
	}
	if conf.UserPWNew != nil {
		state.userPWNewValue = *conf.UserPWNew
	}
	if conf.OwnerPWNew != nil {
		state.ownerPWNewValue = *conf.OwnerPWNew
	}
	if len(conf.AllowedRevocationHosts) > 0 {
		state.allowedRevocationHost0 = conf.AllowedRevocationHosts[0]
		state.allowedRevocationHost0Address = &conf.AllowedRevocationHosts[0]
	}
	if len(conf.AllowedRevocationHosts) > 1 {
		state.allowedRevocationHost1 = conf.AllowedRevocationHosts[1]
	}
	return state
}

func configurationReuseOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"validate", func(conf *model.Configuration) error {
			return Validate(testContext, bytes.NewReader(nil), conf, nil)
		}},
		{"add keywords", func(conf *model.Configuration) error {
			return AddKeywords(testContext, bytes.NewReader(nil), io.Discard, []string{"keyword"}, conf)
		}},
		{"merge", func(conf *model.Configuration) error {
			return MergeRaw(testContext, []io.ReadSeeker{bytes.NewReader(nil)}, io.Discard, false, conf)
		}},
		{"add watermarks", func(conf *model.Configuration) error {
			return AddWatermarks(testContext, bytes.NewReader(nil), io.Discard, nil, &model.Watermark{}, conf)
		}},
		{"extract metadata", func(conf *model.Configuration) error {
			return ExtractMetadata(testContext, bytes.NewReader(nil), func(pdfcpu.Metadata) error { return nil }, conf)
		}},
		{"encrypt", func(conf *model.Configuration) error {
			return Encrypt(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
		{"change user password", func(conf *model.Configuration) error {
			return ChangeUserPassword(testContext, bytes.NewReader(nil), io.Discard, "operation-old", "operation-new", conf)
		}},
		{"change owner password", func(conf *model.Configuration) error {
			return ChangeOwnerPassword(testContext, bytes.NewReader(nil), io.Discard, "operation-old", "operation-new", conf)
		}},
		{"add annotations increment", func(conf *model.Configuration) error {
			rws := nopReadWriteSeeker{bytes.NewReader(nil)}
			return AddAnnotationsAsIncrement(testContext, rws, nil, testAnnotationRenderer(), conf)
		}},
		{"multi-fill form", func(conf *model.Configuration) error {
			return MultiFillForm(
				testContext,
				"missing.pdf",
				bytes.NewReader([]byte(`{`)),
				"",
				"out.pdf",
				form.JSON,
				false,
				conf,
			)
		}},
		{"cut", func(conf *model.Configuration) error {
			return Cut(testContext, bytes.NewReader(nil), "", "", nil, &model.Cut{Hor: []float64{0.5}}, conf)
		}},
	}
}

func TestConfigurationSequentialReadOnlyReuse(t *testing.T) {
	conf := reusableConfiguration()
	want := reusableConfigurationState(conf)

	for pass := 1; pass <= 2; pass++ {
		for _, tt := range configurationReuseOperations(t.Context()) {
			if err := tt.run(conf); err == nil {
				t.Fatalf("pass %d %s: expected operation error", pass, tt.name)
			}
			if got := reusableConfigurationState(conf); got != want {
				t.Fatalf("pass %d %s changed caller configuration:\ngot:  %+v\nwant: %+v", pass, tt.name, got, want)
			}
		}
	}
}

func TestConfigurationConcurrentReadOnlyReuse(t *testing.T) {
	conf := reusableConfiguration()
	want := reusableConfigurationState(conf)
	operations := configurationReuseOperations(t.Context())
	const repetitions = 4

	failures := make(chan string, len(operations)*repetitions)
	var wg sync.WaitGroup
	for i := 0; i < repetitions; i++ {
		for _, tt := range operations {
			tt := tt
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := tt.run(conf); err == nil {
					failures <- tt.name
				}
			}()
		}
	}
	wg.Wait()
	close(failures)

	for name := range failures {
		t.Errorf("%s: expected operation error", name)
	}
	if got := reusableConfigurationState(conf); got != want {
		t.Fatalf("concurrent operations changed caller configuration:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func bookmarkConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"bookmarks", func(conf *model.Configuration) error {
			_, err := Bookmarks(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"list bookmarks", func(conf *model.Configuration) error {
			_, err := ListBookmarks(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"export bookmarks", func(conf *model.Configuration) error {
			return ExportBookmarksJSON(testContext, bytes.NewReader(nil), io.Discard, "in.pdf", conf)
		}},
		{"import bookmarks", func(conf *model.Configuration) error {
			return ImportBookmarks(
				testContext, bytes.NewReader(nil), bytes.NewReader([]byte("{}")), io.Discard, false, conf,
			)
		}},
		{"add bookmarks", func(conf *model.Configuration) error {
			return AddBookmarks(testContext, bytes.NewReader(nil), io.Discard, nil, false, conf)
		}},
		{"remove bookmarks", func(conf *model.Configuration) error {
			return RemoveBookmarks(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
	}
}

func documentDisplayConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"page layout", func(conf *model.Configuration) error {
			_, err := PageLayout(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"list page layout", func(conf *model.Configuration) error {
			_, err := ListPageLayout(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"set page layout", func(conf *model.Configuration) error {
			return SetPageLayout(testContext, bytes.NewReader(nil), io.Discard, model.PageLayoutSinglePage, conf)
		}},
		{"reset page layout", func(conf *model.Configuration) error {
			return ResetPageLayout(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
		{"page mode", func(conf *model.Configuration) error {
			_, err := PageMode(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"list page mode", func(conf *model.Configuration) error {
			_, err := ListPageMode(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"set page mode", func(conf *model.Configuration) error {
			return SetPageMode(testContext, bytes.NewReader(nil), io.Discard, model.PageModeUseNone, conf)
		}},
		{"reset page mode", func(conf *model.Configuration) error {
			return ResetPageMode(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
		{"viewer preferences", func(conf *model.Configuration) error {
			_, _, err := ViewerPreferences(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"list viewer preferences", func(conf *model.Configuration) error {
			_, err := ListViewerPreferences(testContext, bytes.NewReader(nil), false, conf)
			return err
		}},
		{"set viewer preferences", func(conf *model.Configuration) error {
			return SetViewerPreferences(testContext, bytes.NewReader(nil), io.Discard, model.ViewerPreferences{}, conf)
		}},
		{"reset viewer preferences", func(conf *model.Configuration) error {
			return ResetViewerPreferences(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
	}
}

func infoPropertyMergeConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"PDF info", func(conf *model.Configuration) error {
			_, err := PDFInfo(testContext, bytes.NewReader(nil), "in.pdf", nil, false, conf)
			return err
		}},
		{"properties", func(conf *model.Configuration) error {
			_, err := Properties(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"add properties", func(conf *model.Configuration) error {
			return AddProperties(testContext, bytes.NewReader(nil), io.Discard, map[string]string{"Key": "Value"}, conf)
		}},
		{"remove properties", func(conf *model.Configuration) error {
			return RemoveProperties(testContext, bytes.NewReader(nil), io.Discard, []string{"Key"}, conf)
		}},
		{"merge create", func(conf *model.Configuration) error {
			return Merge(testContext, "", []string{"missing.pdf"}, io.Discard, conf, false)
		}},
		{"merge append", func(conf *model.Configuration) error {
			return Merge(testContext, "missing.pdf", nil, io.Discard, conf, false)
		}},
		{"merge zip", func(conf *model.Configuration) error {
			return MergeCreateZip(testContext, bytes.NewReader(nil), bytes.NewReader(nil), io.Discard, conf)
		}},
	}
}

func pageTransformationConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"insert pages before", func(conf *model.Configuration) error {
			return InsertPages(testContext, bytes.NewReader(nil), io.Discard, nil, true, nil, conf)
		}},
		{"insert pages after", func(conf *model.Configuration) error {
			return InsertPages(testContext, bytes.NewReader(nil), io.Discard, nil, false, nil, conf)
		}},
		{"remove pages", func(conf *model.Configuration) error {
			return RemovePages(testContext, bytes.NewReader(nil), io.Discard, nil, conf)
		}},
		{"rotate", func(conf *model.Configuration) error {
			return Rotate(testContext, bytes.NewReader(nil), io.Discard, 90, nil, conf)
		}},
		{"collect", func(conf *model.Configuration) error {
			return Collect(testContext, bytes.NewReader(nil), io.Discard, nil, conf)
		}},
		{"trim", func(conf *model.Configuration) error {
			return Trim(testContext, bytes.NewReader(nil), io.Discard, nil, conf)
		}},
		{"resize", func(conf *model.Configuration) error {
			return Resize(testContext, bytes.NewReader(nil), io.Discard, nil, &model.Resize{Scale: 0.5}, conf)
		}},
		{"zoom", func(conf *model.Configuration) error {
			return Zoom(testContext, bytes.NewReader(nil), io.Discard, nil, &model.Zoom{Factor: 0.5}, conf)
		}},
	}
}

func boxImageConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"boxes", func(conf *model.Configuration) error {
			_, err := Boxes(testContext, bytes.NewReader(nil), nil, conf)
			return err
		}},
		{"list boxes", func(conf *model.Configuration) error {
			_, err := ListBoxes(testContext, bytes.NewReader(nil), nil, nil, conf)
			return err
		}},
		{"add boxes", func(conf *model.Configuration) error {
			pb := &model.PageBoundaries{Crop: &model.Box{}}
			return AddBoxes(testContext, bytes.NewReader(nil), io.Discard, nil, pb, conf)
		}},
		{"remove boxes", func(conf *model.Configuration) error {
			pb := &model.PageBoundaries{Crop: &model.Box{}}
			return RemoveBoxes(testContext, bytes.NewReader(nil), io.Discard, nil, pb, conf)
		}},
		{"crop", func(conf *model.Configuration) error {
			return Crop(testContext, bytes.NewReader(nil), io.Discard, nil, &model.Box{}, conf)
		}},
		{"images", func(conf *model.Configuration) error {
			_, err := Images(testContext, bytes.NewReader(nil), nil, conf)
			return err
		}},
		{"list images", func(conf *model.Configuration) error {
			_, err := ListImages(testContext, bytes.NewReader(nil), nil, conf)
			return err
		}},
		{"update images", func(conf *model.Configuration) error {
			return UpdateImages(testContext, bytes.NewReader(nil), bytes.NewReader(nil), io.Discard, 1, 0, "", conf)
		}},
	}
}

func attachmentPermissionConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"attachments", func(conf *model.Configuration) error {
			_, err := Attachments(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"add attachments", func(conf *model.Configuration) error {
			return AddAttachments(
				testContext, bytes.NewReader(nil), io.Discard, []string{"attachment.txt"}, false, conf,
			)
		}},
		{"add attachment portfolio", func(conf *model.Configuration) error {
			return AddAttachments(
				testContext, bytes.NewReader(nil), io.Discard, []string{"attachment.txt"}, true, conf,
			)
		}},
		{"remove attachments", func(conf *model.Configuration) error {
			return RemoveAttachments(
				testContext, bytes.NewReader(nil), io.Discard, []string{"attachment.txt"}, conf,
			)
		}},
		{"extract attachments raw", func(conf *model.Configuration) error {
			_, err := ExtractAttachmentsRaw(testContext, bytes.NewReader(nil), "out", nil, conf)
			return err
		}},
		{"extract attachments", func(conf *model.Configuration) error {
			return ExtractAttachments(testContext, bytes.NewReader(nil), "out", nil, conf)
		}},
		{"permissions", func(conf *model.Configuration) error {
			_, err := Permissions(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"permissions list", func(conf *model.Configuration) error {
			_, err := PermissionsList(testContext, bytes.NewReader(nil), conf)
			return err
		}},
	}
}

func coreDocumentConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"validate", func(conf *model.Configuration) error {
			return Validate(testContext, bytes.NewReader(nil), conf, nil)
		}},
		{"optimize", func(conf *model.Configuration) error {
			return Optimize(testContext, bytes.NewReader(nil), io.Discard, conf, nil)
		}},
		{"create", func(conf *model.Configuration) error {
			return Create(testContext, bytes.NewReader(nil), bytes.NewReader([]byte("{}")), io.Discard, conf)
		}},
		{"split raw", func(conf *model.Configuration) error {
			_, err := SplitRaw(testContext, bytes.NewReader(nil), 1, conf)
			return err
		}},
		{"split", func(conf *model.Configuration) error {
			return Split(testContext, bytes.NewReader(nil), "out", "in.pdf", 1, conf)
		}},
		{"split by page number", func(conf *model.Configuration) error {
			return SplitByPageNr(testContext, bytes.NewReader(nil), "out", "in.pdf", []int{2}, conf)
		}},
	}
}

func extractionImportConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"import images", func(conf *model.Configuration) error {
			return ImportImages(
				testContext,
				bytes.NewReader(nil),
				io.Discard,
				[]io.Reader{bytes.NewReader(nil)},
				nil,
				conf,
			)
		}},
		{"extract images raw", func(conf *model.Configuration) error {
			_, err := ExtractImagesRaw(testContext, bytes.NewReader(nil), nil, conf)
			return err
		}},
		{"extract images", func(conf *model.Configuration) error {
			return ExtractImages(testContext, bytes.NewReader(nil), nil, func(model.Image, bool, int) error { return nil }, conf)
		}},
		{"extract fonts", func(conf *model.Configuration) error {
			return ExtractFonts(testContext, bytes.NewReader(nil), nil, func(pdfcpu.Font) error { return nil }, conf)
		}},
		{"extract pages", func(conf *model.Configuration) error {
			return ExtractPages(testContext, bytes.NewReader(nil), nil, func(io.Reader, int) error { return nil }, conf)
		}},
		{"extract content", func(conf *model.Configuration) error {
			return ExtractContent(testContext, bytes.NewReader(nil), nil, func(io.Reader, int) error { return nil }, conf)
		}},
		{"extract metadata", func(conf *model.Configuration) error {
			return ExtractMetadata(testContext, bytes.NewReader(nil), func(pdfcpu.Metadata) error { return nil }, conf)
		}},
	}
}

func securitySignatureConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"encrypt", func(conf *model.Configuration) error {
			return Encrypt(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
		{"decrypt", func(conf *model.Configuration) error {
			return Decrypt(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
		{"validate signatures raw", func(conf *model.Configuration) error {
			_, err := ValidateSignaturesRaw(testContext, bytes.NewReader(nil), false, conf)
			return err
		}},
		{"remove signatures", func(conf *model.Configuration) error {
			return RemoveSignatures(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
	}
}

func cutConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"poster", func(conf *model.Configuration) error {
			cut := &model.Cut{
				Scale:   1,
				PageDim: &types.Dim{Width: 100, Height: 100},
				UserDim: true,
			}
			return Poster(testContext, bytes.NewReader(nil), "", "", nil, cut, conf)
		}},
		{"n-down", func(conf *model.Configuration) error {
			return NDown(testContext, bytes.NewReader(nil), "", "", nil, 2, &model.Cut{}, conf)
		}},
		{"cut", func(conf *model.Configuration) error {
			return Cut(testContext, bytes.NewReader(nil), "", "", nil, &model.Cut{Hor: []float64{0.5}}, conf)
		}},
	}
}

func nUpConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"n-up PDF", func(conf *model.Configuration) error {
			nup, err := PDFNUpConfig(4, "", nil)
			if err != nil {
				return err
			}
			return NUp(testContext, bytes.NewReader(nil), io.Discard, nil, nil, nup, conf)
		}},
		{"n-up image", func(conf *model.Configuration) error {
			nup, err := ImageNUpConfig(4, "", nil)
			if err != nil {
				return err
			}
			return NUp(testContext, nil, io.Discard, []string{"missing.png"}, nil, nup, conf)
		}},
		{"n-up from image", func(conf *model.Configuration) error {
			nup, err := ImageNUpConfig(4, "", nil)
			if err != nil {
				return err
			}
			_, err = NUpFromImage(testContext, conf, []string{"missing.png"}, nup)
			return err
		}},
	}
}

func gridConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"grid PDF", func(conf *model.Configuration) error {
			nup, err := PDFGridConfig(2, 2, "", nil)
			if err != nil {
				return err
			}
			return Grid(testContext, bytes.NewReader(nil), io.Discard, nil, nil, nup, conf)
		}},
		{"grid image", func(conf *model.Configuration) error {
			nup, err := ImageGridConfig(2, 2, "", nil)
			if err != nil {
				return err
			}
			return Grid(testContext, nil, io.Discard, []string{"missing.png"}, nil, nup, conf)
		}},
		{"grid from image", func(conf *model.Configuration) error {
			nup, err := ImageGridConfig(2, 2, "", nil)
			if err != nil {
				return err
			}
			_, err = GridFromImage(testContext, conf, []string{"missing.png"}, nup)
			return err
		}},
	}
}

func bookletConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"booklet PDF", func(conf *model.Configuration) error {
			nup, err := PDFBookletConfig(2, "", nil)
			if err != nil {
				return err
			}
			return Booklet(testContext, bytes.NewReader(nil), io.Discard, nil, nil, nup, conf)
		}},
		{"booklet image", func(conf *model.Configuration) error {
			nup, err := ImageBookletConfig(2, "", nil)
			if err != nil {
				return err
			}
			return Booklet(testContext, nil, io.Discard, []string{"missing.png"}, nil, nup, conf)
		}},
		{"booklet from images", func(conf *model.Configuration) error {
			nup, err := ImageBookletConfig(2, "", nil)
			if err != nil {
				return err
			}
			_, err = BookletFromImages(testContext, conf, []string{"missing.png"}, nup)
			return err
		}},
	}
}

func watermarkConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"add watermark map", func(conf *model.Configuration) error {
			wm := &model.Watermark{}
			return AddWatermarksMap(testContext, bytes.NewReader(nil), io.Discard, map[int]*model.Watermark{1: wm}, conf)
		}},
		{"add watermark slice map", func(conf *model.Configuration) error {
			wm := &model.Watermark{}
			return AddWatermarksSliceMap(
				testContext,
				bytes.NewReader(nil),
				io.Discard,
				map[int][]*model.Watermark{1: {wm}},
				conf,
			)
		}},
		{"remove watermarks", func(conf *model.Configuration) error {
			return RemoveWatermarks(testContext, bytes.NewReader(nil), io.Discard, nil, conf)
		}},
	}
}

func annotationConfigurationOperations(testContext context.Context) []configurationOperation {
	ann := testAnnotationRenderer()
	annMap := map[int][]model.AnnotationRenderer{1: {ann}}

	return []configurationOperation{
		{"list annotations", func(conf *model.Configuration) error {
			_, err := Annotations(testContext, bytes.NewReader(nil), nil, conf)
			return err
		}},
		{"add annotations", func(conf *model.Configuration) error {
			return AddAnnotations(testContext, bytes.NewReader(nil), io.Discard, nil, ann, conf)
		}},
		{"add annotations increment", func(conf *model.Configuration) error {
			rws := nopReadWriteSeeker{bytes.NewReader(nil)}
			return AddAnnotationsAsIncrement(testContext, rws, nil, ann, conf)
		}},
		{"add annotation map", func(conf *model.Configuration) error {
			return AddAnnotationsMap(testContext, bytes.NewReader(nil), io.Discard, annMap, conf)
		}},
		{"add annotation map increment", func(conf *model.Configuration) error {
			rws := nopReadWriteSeeker{bytes.NewReader(nil)}
			return AddAnnotationsMapAsIncrement(testContext, rws, annMap, conf)
		}},
		{"remove annotations", func(conf *model.Configuration) error {
			return RemoveAnnotations(testContext, bytes.NewReader(nil), io.Discard, nil, nil, nil, conf)
		}},
		{"remove annotations increment", func(conf *model.Configuration) error {
			rws := nopReadWriteSeeker{bytes.NewReader(nil)}
			return RemoveAnnotationsAsIncrement(testContext, rws, nil, nil, nil, conf)
		}},
	}
}

func basicFormConfigurationOperations(testContext context.Context) []configurationOperation {
	fieldNames := []string{"field"}

	return []configurationOperation{
		{"form fields", func(conf *model.Configuration) error {
			_, err := FormFields(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"list form fields", func(conf *model.Configuration) error {
			_, err := ListFormFields(testContext, bytes.NewReader(nil), conf)
			return err
		}},
		{"remove form fields", func(conf *model.Configuration) error {
			return RemoveFormFields(testContext, bytes.NewReader(nil), io.Discard, fieldNames, conf)
		}},
		{"lock form fields", func(conf *model.Configuration) error {
			return LockFormFields(testContext, bytes.NewReader(nil), io.Discard, fieldNames, conf)
		}},
		{"unlock form fields", func(conf *model.Configuration) error {
			return UnlockFormFields(testContext, bytes.NewReader(nil), io.Discard, fieldNames, conf)
		}},
		{"reset form fields", func(conf *model.Configuration) error {
			return ResetFormFields(testContext, bytes.NewReader(nil), io.Discard, fieldNames, conf)
		}},
	}
}

func exportFormConfigurationOperations(testContext context.Context) []configurationOperation {
	return []configurationOperation{
		{"export form", func(conf *model.Configuration) error {
			_, err := ExportForm(testContext, bytes.NewReader(nil), "in.pdf", conf)
			return err
		}},
		{"export form JSON", func(conf *model.Configuration) error {
			return ExportFormJSON(testContext, bytes.NewReader(nil), io.Discard, "in.pdf", conf)
		}},
	}
}

func TestKeywordsPreservesCallerConfiguration(t *testing.T) {
	conf := &model.Configuration{
		Cmd:            callerMode,
		ValidationMode: model.ValidationStrict,
	}

	if _, err := Keywords(t.Context(), bytes.NewReader(nil), conf); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
}

func testConfigurationOperationsPreserveCaller(t *testing.T, operations []configurationOperation) {
	t.Helper()

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			conf := &model.Configuration{
				Cmd:            callerMode,
				ValidationMode: model.ValidationStrict,
			}

			if err := tt.run(conf); err == nil {
				t.Fatal("expected operation error")
			}
			if conf.Cmd != callerMode {
				t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
			}
			if conf.ValidationMode != model.ValidationStrict {
				t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
			}
		})
	}
}

func TestOperationsPreserveCallerConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		operations []configurationOperation
	}{
		{"bookmarks", bookmarkConfigurationOperations(t.Context())},
		{"document display", documentDisplayConfigurationOperations(t.Context())},
		{"info properties and merge", infoPropertyMergeConfigurationOperations(t.Context())},
		{"page transformations", pageTransformationConfigurationOperations(t.Context())},
		{"boxes and images", boxImageConfigurationOperations(t.Context())},
		{"attachments and permissions", attachmentPermissionConfigurationOperations(t.Context())},
		{"core document", coreDocumentConfigurationOperations(t.Context())},
		{"extraction and import", extractionImportConfigurationOperations(t.Context())},
		{"security and signatures", securitySignatureConfigurationOperations(t.Context())},
		{"cut", cutConfigurationOperations(t.Context())},
		{"n-up", nUpConfigurationOperations(t.Context())},
		{"grid", gridConfigurationOperations(t.Context())},
		{"booklet", bookletConfigurationOperations(t.Context())},
		{"watermarks", watermarkConfigurationOperations(t.Context())},
		{"annotations", annotationConfigurationOperations(t.Context())},
		{"basic forms", basicFormConfigurationOperations(t.Context())},
		{"form export", exportFormConfigurationOperations(t.Context())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfigurationOperationsPreserveCaller(t, tt.operations)
		})
	}
}

func TestSetPermissionsPreservesCallerConfiguration(t *testing.T) {
	conf := &model.Configuration{Cmd: callerMode}

	if err := SetPermissions(t.Context(), bytes.NewReader(nil), io.Discard, conf); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
}

func TestSecuritySignatureFileOperationsPreserveCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []configurationOperation{
		{"encrypt file", func(conf *model.Configuration) error {
			return EncryptFile(t.Context(), inFile, filepath.Join(dir, "encrypted.pdf"), conf)
		}},
		{"decrypt file", func(conf *model.Configuration) error {
			return DecryptFile(t.Context(), inFile, filepath.Join(dir, "decrypted.pdf"), conf)
		}},
		{"validate signatures", func(conf *model.Configuration) error {
			_, err := ValidateSignatures(t.Context(), inFile, false, conf)
			return err
		}},
		{"validate signatures file", func(conf *model.Configuration) error {
			_, err := ValidateSignaturesFile(t.Context(), inFile, false, false, conf)
			return err
		}},
		{"remove signatures file", func(conf *model.Configuration) error {
			return RemoveSignaturesFile(t.Context(), inFile, filepath.Join(dir, "unsigned.pdf"), conf)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &model.Configuration{Cmd: callerMode}
			if err := tt.run(conf); err == nil {
				t.Fatal("expected malformed PDF error")
			}
			if conf.Cmd != callerMode {
				t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
			}
		})
	}
}

func TestNUpFilePreservesCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	nup, err := PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	conf := &model.Configuration{Cmd: callerMode}

	if err := NUpFile(t.Context(), []string{inFile}, filepath.Join(dir, "out.pdf"), nil, nup, conf); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
}

func TestGridFilePreservesCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	nup, err := PDFGridConfig(2, 2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	conf := &model.Configuration{Cmd: callerMode}

	if err := GridFile(t.Context(), []string{inFile}, filepath.Join(dir, "out.pdf"), nil, nup, conf); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
}

func TestBookletFilePreservesCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	nup, err := PDFBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	conf := &model.Configuration{Cmd: callerMode}

	if err := BookletFile(t.Context(), []string{inFile}, filepath.Join(dir, "out.pdf"), nil, nup, conf); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
}

func TestFillFormPreservesCallerConfiguration(t *testing.T) {
	conf := &model.Configuration{
		Cmd:            callerMode,
		ValidationMode: model.ValidationStrict,
	}

	err := FillForm(t.Context(), bytes.NewReader(nil), bytes.NewReader([]byte(`{"forms":[]}`)), io.Discard, conf)
	if err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
}

func TestMultiFillFormPreservesCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	conf := &model.Configuration{
		Cmd:            callerMode,
		ValidationMode: model.ValidationStrict,
	}

	err := MultiFillForm(
		t.Context(),
		filepath.Join(dir, "missing.pdf"),
		bytes.NewReader([]byte(`{"forms":[{}]}`)),
		dir,
		"out.pdf",
		form.JSON,
		false,
		conf,
	)
	if err == nil {
		t.Fatal("expected operation error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
}

func TestOptimizeFilePreservesCallerConfiguration(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	conf := &model.Configuration{Cmd: callerMode}

	if err := OptimizeFile(t.Context(), inFile, filepath.Join(dir, "out.pdf"), conf, nil); err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
}

func TestKeywordMutationsPreserveCallerConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(context.Context, io.ReadSeeker, io.Writer, []string, *model.Configuration) error
	}{
		{"add", AddKeywords},
		{"remove", RemoveKeywords},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &model.Configuration{
				Cmd:            callerMode,
				ValidationMode: model.ValidationStrict,
			}

			err := tt.mutate(t.Context(), bytes.NewReader(nil), io.Discard, []string{"keyword"}, conf)
			if err == nil {
				t.Fatal("expected malformed PDF error")
			}
			if conf.Cmd != callerMode {
				t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
			}
			if conf.ValidationMode != model.ValidationStrict {
				t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
			}
		})
	}
}

func TestAddWatermarksPreservesCallerConfiguration(t *testing.T) {
	conf := &model.Configuration{
		Cmd:                             callerMode,
		OptimizeDuplicateContentStreams: true,
	}

	err := AddWatermarks(t.Context(), bytes.NewReader(nil), io.Discard, nil, nil, conf)
	if !errors.Is(err, ErrMissingWatermarkConfiguration) {
		t.Fatalf("expected missing watermark configuration, got %v", err)
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if !conf.OptimizeDuplicateContentStreams {
		t.Error("operation disabled duplicate content stream optimization on caller configuration")
	}
}

func TestMergeRawPreservesCallerConfiguration(t *testing.T) {
	conf := &model.Configuration{
		Cmd:             callerMode,
		ValidationMode:  model.ValidationStrict,
		CreateBookmarks: true,
	}

	err := MergeRaw(t.Context(), []io.ReadSeeker{bytes.NewReader(nil)}, io.Discard, false, conf)
	if err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Errorf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
	if !conf.CreateBookmarks {
		t.Error("merge disabled bookmark creation on caller configuration")
	}
}

func TestChangeUserPasswordPreservesCallerConfiguration(t *testing.T) {
	userPWNew := "configured-new-user"
	conf := &model.Configuration{
		Cmd:       callerMode,
		UserPW:    "configured-user",
		UserPWNew: &userPWNew,
	}

	err := ChangeUserPassword(t.Context(), bytes.NewReader(nil), io.Discard, "operation-old", "operation-new", conf)
	if err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.UserPW != "configured-user" {
		t.Errorf("caller user password: got %q, want configured-user", conf.UserPW)
	}
	if conf.UserPWNew != &userPWNew {
		t.Error("operation replaced caller new user password")
	}
	if conf.UserPWNew == nil {
		t.Error("operation cleared caller new user password")
	} else if *conf.UserPWNew != "configured-new-user" {
		t.Errorf("caller new user password: got %q, want configured-new-user", *conf.UserPWNew)
	}
}

func TestChangeOwnerPasswordPreservesCallerConfiguration(t *testing.T) {
	ownerPWNew := "configured-new-owner"
	conf := &model.Configuration{
		Cmd:        callerMode,
		OwnerPW:    "configured-owner",
		OwnerPWNew: &ownerPWNew,
	}

	err := ChangeOwnerPassword(t.Context(), bytes.NewReader(nil), io.Discard, "operation-old", "operation-new", conf)
	if err == nil {
		t.Fatal("expected malformed PDF error")
	}
	if conf.Cmd != callerMode {
		t.Errorf("caller command mode: got %d, want %d", conf.Cmd, callerMode)
	}
	if conf.OwnerPW != "configured-owner" {
		t.Errorf("caller owner password: got %q, want configured-owner", conf.OwnerPW)
	}
	if conf.OwnerPWNew != &ownerPWNew {
		t.Error("operation replaced caller new owner password")
	}
	if conf.OwnerPWNew == nil {
		t.Error("operation cleared caller new owner password")
	} else if *conf.OwnerPWNew != "configured-new-owner" {
		t.Errorf("caller new owner password: got %q, want configured-new-owner", *conf.OwnerPWNew)
	}
}

func nilConfigurationOperations(t *testing.T) []configurationOperation {
	dir := t.TempDir()
	operations := []configurationOperation{
		{"keywords", func(conf *model.Configuration) error {
			_, err := Keywords(t.Context(), bytes.NewReader(nil), conf)
			return err
		}},
		{"add keywords", func(conf *model.Configuration) error {
			return AddKeywords(t.Context(), bytes.NewReader(nil), io.Discard, []string{"keyword"}, conf)
		}},
		{"remove keywords", func(conf *model.Configuration) error {
			return RemoveKeywords(t.Context(), bytes.NewReader(nil), io.Discard, []string{"keyword"}, conf)
		}},
		{"fill form", func(conf *model.Configuration) error {
			return FillForm(t.Context(), bytes.NewReader(nil), bytes.NewReader([]byte(`{"forms":[]}`)), io.Discard, conf)
		}},
		{"multi-fill form", func(conf *model.Configuration) error {
			return MultiFillForm(
				t.Context(),
				filepath.Join(dir, "missing.pdf"),
				bytes.NewReader([]byte(`{"forms":[{}]}`)),
				dir,
				"out.pdf",
				form.JSON,
				false,
				conf,
			)
		}},
		{"merge", func(conf *model.Configuration) error {
			return MergeRaw(t.Context(), []io.ReadSeeker{bytes.NewReader(nil)}, io.Discard, false, conf)
		}},
	}

	for _, group := range [][]configurationOperation{
		bookmarkConfigurationOperations(t.Context()),
		documentDisplayConfigurationOperations(t.Context()),
		infoPropertyMergeConfigurationOperations(t.Context()),
		pageTransformationConfigurationOperations(t.Context()),
		boxImageConfigurationOperations(t.Context()),
		attachmentPermissionConfigurationOperations(t.Context()),
		coreDocumentConfigurationOperations(t.Context()),
		extractionImportConfigurationOperations(t.Context()),
		securitySignatureConfigurationOperations(t.Context())[2:],
		cutConfigurationOperations(t.Context()),
		nUpConfigurationOperations(t.Context()),
		gridConfigurationOperations(t.Context()),
		bookletConfigurationOperations(t.Context()),
		watermarkConfigurationOperations(t.Context()),
		annotationConfigurationOperations(t.Context()),
		basicFormConfigurationOperations(t.Context()),
		exportFormConfigurationOperations(t.Context()),
	} {
		operations = append(operations, group...)
	}
	return operations
}

func requiredConfigurationOperations(testContext context.Context) []configurationOperation {
	operations := []configurationOperation{
		{"user password", func(conf *model.Configuration) error {
			return ChangeUserPassword(testContext, bytes.NewReader(nil), io.Discard, "old", "new", conf)
		}},
		{"owner password", func(conf *model.Configuration) error {
			return ChangeOwnerPassword(testContext, bytes.NewReader(nil), io.Discard, "old", "new", conf)
		}},
		{"set permissions", func(conf *model.Configuration) error {
			return SetPermissions(testContext, bytes.NewReader(nil), io.Discard, conf)
		}},
	}
	return append(operations, securitySignatureConfigurationOperations(testContext)[:2]...)
}

func testDefaultConfigurationOperations(t *testing.T, operations []configurationOperation) {
	t.Helper()

	for _, tt := range operations {
		t.Run(tt.name+" default configuration", func(t *testing.T) {
			err := tt.run(nil)
			if err == nil || errors.Is(err, ErrMissingConfiguration) {
				t.Fatalf("expected operation error after default configuration, got %v", err)
			}
		})
	}
}

func testRequiredConfigurationOperations(t *testing.T, operations []configurationOperation) {
	t.Helper()

	for _, tt := range operations {
		t.Run(tt.name+" requires configuration", func(t *testing.T) {
			if err := tt.run(nil); !errors.Is(err, ErrMissingConfiguration) {
				t.Fatalf("expected missing configuration, got %v", err)
			}
		})
	}
}

func testMissingWatermarkConfiguration(t *testing.T) {
	t.Helper()

	t.Run("watermark default configuration", func(t *testing.T) {
		err := AddWatermarks(t.Context(), bytes.NewReader(nil), io.Discard, nil, nil, nil)
		if !errors.Is(err, ErrMissingWatermarkConfiguration) {
			t.Fatalf("expected missing watermark configuration, got %v", err)
		}
	})
}

func TestNilConfigurationCompatibility(t *testing.T) {
	testDefaultConfigurationOperations(t, nilConfigurationOperations(t))
	testMissingWatermarkConfiguration(t)
	testRequiredConfigurationOperations(t, requiredConfigurationOperations(t.Context()))
}
