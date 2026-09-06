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
	"bytes"
	stdlog "log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func runPresentationCommand(command *Command) error {
	_, err := Dispatch(command)
	return err
}

func TestOptimizationRendersTypedProgressStages(t *testing.T) {
	var output bytes.Buffer
	log.SetCLILogger(stdlog.New(&output, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "optimized.pdf")
	if err := runPresentationCommand(OptimizeCommand(inFile, outFile, nil)); err != nil {
		t.Fatal(err)
	}

	want := "optimizing...\nwriting " + outFile + "...\n"
	if got := output.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDocumentAndPageCommandsOwnPresentation(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	jsonFile := filepath.Join(dir, "missing.json")
	pb := &model.PageBoundaries{}
	pb.SelectAll()

	tests := []struct {
		name    string
		command *Command
		want    []string
	}{
		{
			name:    "merge",
			command: MergeCreateCommand([]string{missing}, outFile, false, nil),
			want:    []string{"writing " + outFile, missing},
		},
		{
			name:    "merge zip",
			command: MergeCreateZipCommand([]string{missing, missing}, outFile, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "merge append",
			command: MergeAppendCommand([]string{missing}, outFile, false, nil),
			want:    []string{"writing " + outFile, missing},
		},
		{
			name:    "split",
			command: SplitCommand(missing, dir, 1, nil),
			want:    []string{"splitting " + missing + " to " + dir + "/"},
		},
		{
			name:    "split by page number",
			command: SplitByPageNrCommand(missing, dir, []int{2}, nil),
			want:    []string{"splitting " + missing + " to " + dir + "/"},
		},
		{
			name:    "trim",
			command: TrimCommand(missing, outFile, nil, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "collect",
			command: CollectCommand(missing, outFile, nil, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "create",
			command: CreateCommand("", jsonFile, outFile, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "resize",
			command: ResizeCommand(missing, outFile, nil, &model.Resize{}, nil),
			want:    []string{"resizing " + missing, "writing " + outFile},
		},
		{
			name:    "n-up",
			command: NUpCommand([]string{missing}, outFile, nil, &model.NUp{}, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "grid",
			command: GridCommand([]string{missing}, outFile, nil, &model.NUp{}, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "booklet",
			command: BookletCommand([]string{missing}, outFile, nil, &model.NUp{}, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "poster",
			command: PosterCommand(missing, dir, "", nil, &model.Cut{}, nil),
			want:    []string{"creating poster pages from " + missing + " into " + dir + "/"},
		},
		{
			name:    "n-down",
			command: NDownCommand(missing, dir, "", nil, 2, &model.Cut{}, nil),
			want:    []string{"ndown " + missing + " into " + dir + "/"},
		},
		{
			name:    "cut",
			command: CutCommand(missing, dir, "", nil, &model.Cut{}, nil),
			want:    []string{"cutting " + missing + " into " + dir + "/"},
		},
		{
			name:    "zoom",
			command: ZoomCommand(missing, outFile, nil, &model.Zoom{}, nil),
			want:    []string{"zooming " + missing, "writing " + outFile},
		},
		{
			name:    "rotate",
			command: RotateCommand(missing, outFile, 90, nil, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "insert pages",
			command: InsertPagesCommand(missing, outFile, nil, nil, "before", nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "remove pages",
			command: RemovePagesCommand(missing, outFile, nil, nil),
			want:    []string{"writing " + outFile},
		},
		{
			name:    "list boxes",
			command: ListBoxesCommand(missing, nil, pb, nil),
			want:    []string{"listing ", " for " + missing},
		},
		{
			name:    "add boxes",
			command: AddBoxesCommand(missing, outFile, nil, pb, nil),
			want:    []string{"adding ", " for " + missing, "writing " + outFile},
		},
		{
			name:    "remove boxes",
			command: RemoveBoxesCommand(missing, outFile, nil, pb, nil),
			want:    []string{"removing ", " for " + missing, "writing " + outFile},
		},
		{
			name:    "crop",
			command: CropCommand(missing, outFile, nil, &model.Box{}, nil),
			want:    []string{"cropping " + missing, "writing " + outFile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			log.SetCLILogger(stdlog.New(&output, "", 0))
			t.Cleanup(func() { log.SetCLILogger(nil) })

			if err := runPresentationCommand(tt.command); err == nil {
				t.Fatal("expected missing-input or invalid-configuration error")
			}
			for _, want := range tt.want {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("missing %q in %q", want, output.String())
				}
			}
		})
	}
}

func TestResourceAndContentCommandsOwnPresentation(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.pdf")
	missingImage := filepath.Join(dir, "missing.png")
	outFile := filepath.Join(dir, "out.pdf")
	jsonFile := filepath.Join(dir, "form.json")
	csvFile := filepath.Join(dir, "form.csv")
	attachment := filepath.Join(dir, "attachment.txt")
	password := "secret"

	tests := []struct {
		name    string
		command *Command
		want    []string
	}{
		{name: "remove watermarks", command: RemoveWatermarksCommand(missing, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "remove annotations", command: RemoveAnnotationsCommand(missing, outFile, nil, nil, nil, nil), want: []string{"writing " + outFile}},
		{name: "export bookmarks", command: ExportBookmarksCommand(missing, jsonFile, nil), want: []string{"writing " + jsonFile}},
		{name: "import bookmarks", command: ImportBookmarksCommand(missing, jsonFile, outFile, false, nil), want: []string{"writing " + outFile}},
		{name: "remove bookmarks", command: RemoveBookmarksCommand(missing, outFile, nil), want: []string{"writing " + outFile}},
		{name: "import images", command: ImportImagesCommand([]string{missingImage}, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "update images", command: UpdateImagesCommand(missing, missingImage, outFile, 1, "", nil), want: []string{"writing " + outFile}},
		{name: "extract images", command: ExtractImagesCommand(missing, dir, nil, nil), want: []string{"extracting images from " + missing + " into " + dir + "/"}},
		{name: "extract fonts", command: ExtractFontsCommand(missing, dir, nil, nil), want: []string{"extracting fonts from " + missing + " into " + dir + "/"}},
		{name: "extract pages", command: ExtractPagesCommand(missing, dir, nil, nil), want: []string{"extracting pages from " + missing + " into " + dir + "/"}},
		{name: "extract content", command: ExtractContentCommand(missing, dir, nil, nil), want: []string{"extracting content from " + missing + " into " + dir + "/"}},
		{name: "extract metadata", command: ExtractMetadataCommand(missing, dir, nil), want: []string{"extracting metadata from " + missing + " into " + dir + "/"}},
		{name: "add attachments", command: AddAttachmentsCommand(missing, outFile, []string{attachment}, nil), want: []string{"adding " + attachment}},
		{name: "extract attachments", command: ExtractAttachmentsCommand(missing, dir, nil, nil), want: []string{"extracting attachments from " + missing + " into " + dir + "/"}},
		{name: "remove form fields", command: RemoveFormFieldsCommand(missing, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "lock form fields", command: LockFormCommand(missing, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "unlock form fields", command: UnlockFormCommand(missing, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "reset form fields", command: ResetFormCommand(missing, outFile, nil, nil), want: []string{"writing " + outFile}},
		{name: "export form", command: ExportFormCommand(missing, jsonFile, nil), want: []string{"writing " + jsonFile}},
		{name: "fill form", command: FillFormCommand(missing, jsonFile, outFile, nil), want: []string{"filling...", "writing " + outFile}},
		{
			name:    "multi-fill form",
			command: MultiFillFormCommand(missing, csvFile, dir, outFile, true, nil),
			want:    []string{"filling multiple forms via " + missing + " based on CSV data from " + csvFile + " into " + dir + "/out.pdf"},
		},
		{name: "encrypt", command: EncryptCommand(missing, outFile, nil), want: []string{"writing " + outFile}},
		{name: "decrypt", command: DecryptCommand(missing, outFile, nil), want: []string{"writing " + outFile}},
		{name: "change user password", command: ChangeUserPWCommand(missing, outFile, &password, &password, nil), want: []string{"writing " + outFile}},
		{name: "change owner password", command: ChangeOwnerPWCommand(missing, outFile, &password, &password, nil), want: []string{"writing " + outFile}},
		{name: "set permissions", command: SetPermissionsCommand(missing, outFile, nil), want: []string{"writing " + outFile}},
		{name: "remove signatures", command: RemoveSignaturesCommand(missing, outFile, nil), want: []string{"writing " + outFile}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			log.SetCLILogger(stdlog.New(&output, "", 0))
			t.Cleanup(func() { log.SetCLILogger(nil) })

			if err := runPresentationCommand(tt.command); err == nil {
				t.Fatal("expected missing-input error")
			}
			for _, want := range tt.want {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("missing %q in %q", want, output.String())
				}
			}
		})
	}
}

func TestOrdinaryResourceOutputUsesCommandResult(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "attachment.pdf")
	attachment := "presentation_test.go"
	log.SetCLILogger(nil)
	if err := runPresentationCommand(AddAttachmentsCommand(inFile, outFile, []string{attachment}, nil)); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetCLILogger(stdlog.New(&output, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })
	result, err := Dispatch(ListAttachmentsCommand(outFile, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != attachment {
		t.Fatalf("got %#v, want [%q]", result, attachment)
	}
	if output.Len() != 0 {
		t.Fatalf("ordinary result leaked to CLI logger: %q", output.String())
	}
}

func TestOrdinaryPresentationIsQuiet(t *testing.T) {
	log.SetCLILogger(nil)
	var errorOutput bytes.Buffer
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	cmd := TrimCommand(inFile, outFile, nil, nil)
	cmd.ErrorOutput = &errorOutput

	if err := runPresentationCommand(cmd); err == nil {
		t.Fatal("expected missing-input error")
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("quiet command wrote presentation: %q", errorOutput.String())
	}
}

func TestResourcePresentationIsQuiet(t *testing.T) {
	log.SetCLILogger(nil)
	var errorOutput bytes.Buffer
	dir := t.TempDir()
	cmd := ExtractAttachmentsCommand(filepath.Join(dir, "missing.pdf"), dir, nil, nil)
	cmd.ErrorOutput = &errorOutput

	if err := runPresentationCommand(cmd); err == nil {
		t.Fatal("expected missing-input error")
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("quiet command wrote presentation: %q", errorOutput.String())
	}
}

func TestPDFStdoutSuppressesPresentation(t *testing.T) {
	var output bytes.Buffer
	log.SetCLILogger(stdlog.New(&output, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	cmd := ResizeCommand(inFile, "-", nil, &model.Resize{}, nil)
	if err := runPresentationCommand(cmd); err == nil {
		t.Fatal("expected missing-input error")
	}
	if output.Len() != 0 {
		t.Fatalf("stdout PDF command wrote presentation: %q", output.String())
	}

	implicitStdout := ResizeCommand("-", "", nil, &model.Resize{}, nil)
	reportCommandProgress(implicitStdout, "must not be rendered\n")
	reportCommandOutputPath(implicitStdout)
	if output.Len() != 0 {
		t.Fatalf("implicit stdout PDF command wrote presentation: %q", output.String())
	}
}

func TestResourcePDFStdoutSuppressesPresentation(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.pdf")
	missingImage := filepath.Join(dir, "missing.png")
	missingJSON := filepath.Join(dir, "missing.json")
	attachment := filepath.Join(dir, "attachment.txt")

	tests := []struct {
		name    string
		command *Command
	}{
		{name: "remove watermarks", command: RemoveWatermarksCommand(missing, "-", nil, nil)},
		{name: "import images", command: ImportImagesCommand([]string{missingImage}, "-", nil, nil)},
		{name: "update images", command: UpdateImagesCommand(missing, missingImage, "-", 1, "", nil)},
		{name: "add attachments", command: AddAttachmentsCommand(missing, "-", []string{attachment}, nil)},
		{name: "fill form", command: FillFormCommand(missing, missingJSON, "-", nil)},
		{name: "encrypt", command: EncryptCommand(missing, "-", nil)},
		{name: "remove signatures", command: RemoveSignaturesCommand(missing, "-", nil)},
		{name: "extract pages", command: ExtractPagesCommand(missing, "-", nil, nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			log.SetCLILogger(stdlog.New(&output, "", 0))
			t.Cleanup(func() { log.SetCLILogger(nil) })

			if err := runPresentationCommand(tt.command); err == nil {
				t.Fatal("expected missing-input error")
			}
			if output.Len() != 0 {
				t.Fatalf("stdout PDF command wrote presentation: %q", output.String())
			}
		})
	}
}
