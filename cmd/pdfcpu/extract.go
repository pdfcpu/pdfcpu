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

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type extractOptions struct {
	mode string
}

func extractCmd() *cobra.Command {
	opts := &extractOptions{
		mode: "",
	}

	cmd := &cobra.Command{
		Use:   "extract inFile outDir",
		Short: "Extract images, fonts, content, pages or metadata",
		Long:  usageLongExtract,
		Args:  cobra.ExactArgs(2),
		RunE: wrapContextHandler(func(c context.Context, conf *model.Configuration, args []string) error {
			return handleExtractCommand(c, conf, args, opts)
		}),
	}

	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "extraction mode: i(mage)|f(ont)|c(ontent)|p(age)|m(eta)")
	addSelectedPagesFlag(cmd)
	cmd.MarkFlagRequired("mode")
	addPasswordFlags(cmd)

	return cmd
}

func extractMode(opts *extractOptions) error {
	opts.mode = modeCompletion(opts.mode, []string{"image", "font", "page", "content", "meta"})
	if opts.mode == "" {
		return errors.New("mode must be one of: image, font, page, content, meta")
	}
	return nil
}

func extractInputOutput(conf *model.Configuration, args []string) (string, string, error) {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return "", "", err
	}
	outDir := args[1]
	if err := ensureOutputDirEmpty(outDir); err != nil {
		return "", "", err
	}
	return inFile, outDir, nil
}

func extractCommandForMode(mode, inFile, outDir string, pages []string, conf *model.Configuration) (*cli.Command, error) {
	switch mode {
	case "image":
		return cli.ExtractImagesCommand(inFile, outDir, pages, conf), nil
	case "font":
		return cli.ExtractFontsCommand(inFile, outDir, pages, conf), nil
	case "page":
		return cli.ExtractPagesCommand(inFile, outDir, pages, conf), nil
	case "content":
		return cli.ExtractContentCommand(inFile, outDir, pages, conf), nil
	case "meta":
		return cli.ExtractMetadataCommand(inFile, outDir, conf), nil
	}
	return nil, fmt.Errorf("unknown extract mode: %s", mode)
}

func handleExtractCommand(c context.Context, conf *model.Configuration, args []string, opts *extractOptions) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := extractMode(opts); err != nil {
		return err
	}
	inFile, outDir, err := extractInputOutput(conf, args)
	if err != nil {
		return err
	}
	pages, err := parseSelectedPages()
	if err != nil {
		return err
	}
	cmd, err := extractCommandForMode(opts.mode, inFile, outDir, pages, conf)
	if err != nil {
		return err
	}
	return runCommand(c, cmd)
}
