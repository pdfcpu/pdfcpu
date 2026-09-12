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
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const extractPagesOperation = "extract pages"

func validateExtractionCommand(cmd *Command, operation string) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outDir:    commandStringRequiredNonEmpty,
	})
}

func reportUnsupportedResourceSkips(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var unsupportedErr *api.UnsupportedResourceError
	if !errors.As(err, &unsupportedErr) {
		return err
	}
	if log.CLIEnabled() {
		log.CLI.Println(err)
	}
	return nil
}

func reportExtractionProgress(cmd *Command, resource string) {
	inFile := *cmd.InFile
	if inFile == "-" {
		inFile = "stdin"
	}
	reportCommandProgress(cmd, "extracting %s from %s into %s/ ...\n", resource, inFile, *cmd.OutDir)
}

func writeExtractedPageToStdout(c context.Context, ctx *model.Context, pageNr int, w io.Writer) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	r, err := api.ExtractPage(c, ctx, pageNr)
	if err != nil {
		return fmt.Errorf("%s: extraction: %w", extractPagesOperation, err)
	}

	if _, err := io.Copy(w, contextReader{ctx: c, r: r}); err != nil {
		return fmt.Errorf("%s: stdout copy: %w", extractPagesOperation, err)
	}
	return c.Err()
}

func extractSelectedPageToStdout(c context.Context, rs io.ReadSeeker, w io.Writer, cmd *Command) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	conf := configurationForMode(cmd.Conf, model.EXTRACTPAGES)

	ctx, err := api.ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("%s: read: %w", extractPagesOperation, err)
	}

	pages, err := api.PagesForSelection(ctx.PageCount, cmd.PageSelection, true)
	if err != nil {
		return fmt.Errorf("%s: selection: %w", extractPagesOperation, err)
	}

	pageNr, count := 0, 0
	for i, selected := range pages {
		if err := c.Err(); err != nil {
			return err
		}
		if selected {
			pageNr = i
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("%s: selection: stdout requires exactly one selected page", extractPagesOperation)
	}

	return writeExtractedPageToStdout(c, ctx, pageNr, w)
}

func extractPageToStdout(c context.Context, cmd *Command) error {
	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, "-", extractPagesOperation)
	if err != nil {
		return err
	}
	return finalize(extractSelectedPageToStdout(c, rs, w, cmd))
}

func extractImages(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateExtractionCommand(cmd, "extract images"); err != nil {
		return nil, err
	}
	reportExtractionProgress(cmd, "images")
	if *cmd.InFile == "-" {
		return withStdinReadSeeker(c, "extract images", func(rs io.ReadSeeker) ([]string, error) {
			err := api.ExtractImages(
				c, rs, cmd.PageSelection, api.WriteImageToDisk(c, *cmd.OutDir, "stdin"), cmd.Conf,
			)
			return nil, reportUnsupportedResourceSkips(err)
		})
	}
	return nil, reportUnsupportedResourceSkips(
		api.ExtractImagesFile(c, *cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf),
	)
}

func extractFonts(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateExtractionCommand(cmd, "extract fonts"); err != nil {
		return nil, err
	}
	reportExtractionProgress(cmd, "fonts")
	if *cmd.InFile == "-" {
		return withStdinReadSeeker(c, "extract fonts", func(rs io.ReadSeeker) ([]string, error) {
			err := api.ExtractFonts(
				c, rs, cmd.PageSelection, api.WriteFontToDisk(c, *cmd.OutDir, "stdin"), cmd.Conf,
			)
			return nil, reportUnsupportedResourceSkips(err)
		})
	}
	return nil, reportUnsupportedResourceSkips(
		api.ExtractFontsFile(c, *cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf),
	)
}

func extractPages(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateExtractionCommand(cmd, extractPagesOperation); err != nil {
		return nil, err
	}
	reportExtractionProgress(cmd, "pages")
	if *cmd.OutDir == "-" {
		return nil, extractPageToStdout(c, cmd)
	}

	if *cmd.InFile == "-" {
		return withStdinReadSeeker(c, "extract pages", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.ExtractPages(
				c, rs, cmd.PageSelection, api.WritePageToDisk(c, *cmd.OutDir, "stdin"), cmd.Conf,
			)
		})
	}

	return nil, api.ExtractPagesFile(c, *cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

func extractContent(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateExtractionCommand(cmd, "extract content"); err != nil {
		return nil, err
	}
	reportExtractionProgress(cmd, "content")
	if *cmd.InFile == "-" {
		return withStdinReadSeeker(c, "extract content", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.ExtractContent(
				c, rs, cmd.PageSelection, api.WriteContentToDisk(c, *cmd.OutDir, "stdin"), cmd.Conf,
			)
		})
	}
	return nil, api.ExtractContentFile(c, *cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

func extractMetadata(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateExtractionCommand(cmd, "extract metadata"); err != nil {
		return nil, err
	}
	reportExtractionProgress(cmd, "metadata")
	if *cmd.InFile == "-" {
		return withStdinReadSeeker(c, "extract metadata", func(rs io.ReadSeeker) ([]string, error) {
			err := api.ExtractMetadata(
				c, rs, api.WriteMetadataToDisk(c, *cmd.OutDir, "stdin"), cmd.Conf,
			)
			return nil, reportUnsupportedResourceSkips(err)
		})
	}

	return nil, reportUnsupportedResourceSkips(
		api.ExtractMetadataFile(c, *cmd.InFile, *cmd.OutDir, cmd.Conf),
	)
}
