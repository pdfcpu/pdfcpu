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
	"errors"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const extractPagesOperation = "extract pages"

func reportUnsupportedResourceSkips(err error) error {
	var unsupportedErr *api.UnsupportedResourceError
	if !errors.As(err, &unsupportedErr) {
		return err
	}
	if log.CLIEnabled() {
		log.CLI.Println(err)
	}
	return nil
}

func writeExtractedPageToStdout(ctx *model.Context, pageNr int, w io.Writer) error {
	r, err := api.ExtractPage(ctx, pageNr)
	if err != nil {
		return fmt.Errorf("%s: extraction: %w", extractPagesOperation, err)
	}

	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("%s: stdout copy: %w", extractPagesOperation, err)
	}
	return nil
}

func extractSelectedPageToStdout(rs io.ReadSeeker, w io.Writer, cmd *Command) error {
	conf := cmd.Conf
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTPAGES

	ctx, err := api.ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("%s: read: %w", extractPagesOperation, err)
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, cmd.PageSelection, true, true)
	if err != nil {
		return fmt.Errorf("%s: selection: %w", extractPagesOperation, err)
	}

	pageNr, count := 0, 0
	for i, selected := range pages {
		if selected {
			pageNr = i
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("%s: selection: stdout requires exactly one selected page", extractPagesOperation)
	}

	return writeExtractedPageToStdout(ctx, pageNr, w)
}

func extractPageToStdout(cmd *Command) error {
	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, "-", extractPagesOperation)
	if err != nil {
		return err
	}
	return finalize(extractSelectedPageToStdout(rs, w, cmd))
}

// ExtractImages dumps embedded image resources from inFile into outDir for selected pages.
func ExtractImages(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract images", func(rs io.ReadSeeker) ([]string, error) {
			return nil, reportUnsupportedResourceSkips(api.ExtractImages(rs, cmd.PageSelection, api.WriteImageToDisk(*cmd.OutDir, "stdin"), cmd.Conf))
		})
	}
	return nil, reportUnsupportedResourceSkips(api.ExtractImagesFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf))
}

// ExtractFonts dumps embedded fontfiles from inFile into outDir for selected pages.
func ExtractFonts(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract fonts", func(rs io.ReadSeeker) ([]string, error) {
			return nil, reportUnsupportedResourceSkips(api.ExtractFonts(rs, cmd.PageSelection, api.WriteFontToDisk(*cmd.OutDir, "stdin"), cmd.Conf))
		})
	}
	return nil, reportUnsupportedResourceSkips(api.ExtractFontsFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf))
}

// ExtractPages generates single page PDF files from inFile in outDir for selected pages.
func ExtractPages(cmd *Command) ([]string, error) {
	if *cmd.OutDir == "-" {
		return nil, extractPageToStdout(cmd)
	}

	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract pages", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.ExtractPages(rs, cmd.PageSelection, api.WritePageToDisk(*cmd.OutDir, "stdin"), cmd.Conf)
		})
	}

	return nil, api.ExtractPagesFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractContent dumps "PDF source" files from inFile into outDir for selected pages.
func ExtractContent(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract content", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.ExtractContent(rs, cmd.PageSelection, api.WriteContentToDisk(*cmd.OutDir, "stdin"), cmd.Conf)
		})
	}
	return nil, api.ExtractContentFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractMetadata dumps all metadata dict entries for inFile into outDir.
func ExtractMetadata(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract metadata", func(rs io.ReadSeeker) ([]string, error) {
			return nil, reportUnsupportedResourceSkips(api.ExtractMetadata(rs, api.WriteMetadataToDisk(*cmd.OutDir, "stdin"), cmd.Conf))
		})
	}

	return nil, reportUnsupportedResourceSkips(api.ExtractMetadataFile(*cmd.InFile, *cmd.OutDir, cmd.Conf))
}
