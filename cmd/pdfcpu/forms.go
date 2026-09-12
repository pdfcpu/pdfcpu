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
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type formListOptions struct {
	json bool
}
type formMultifillOptions struct {
	mode string
}

func formCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "form",
		Short: "List, remove fields, lock, unlock, reset, export, fill form via JSON or CSV",
		Long:  usageLongForm,
	}

	fill := &cobra.Command{
		Use:   "fill inFile inFileJSON [ outFile ]",
		Short: "Fill form with data via JSON",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapContextHandler(handleFillFormCommand),
	}

	multifillOpts := &formMultifillOptions{mode: "single"}
	multifill := &cobra.Command{
		Use:   "multifill inFile inFileData outDir [ outFile ]",
		Short: "Fill multiple form instances",
		Args:  cobra.RangeArgs(3, 4),
		RunE: wrapContextHandler(func(c context.Context, conf *model.Configuration, args []string) error {
			return handleMultiFillFormCommand(c, conf, args, multifillOpts)
		}),
	}
	multifill.Flags().StringVarP(&multifillOpts.mode, "mode", "m", multifillOpts.mode, "output mode: single|merge")

	listOpts := &formListOptions{json: false}
	list := &cobra.Command{
		Use:   "list inFile...",
		Short: "List form fields",
		Args:  cobra.MinimumNArgs(1),
		RunE: wrapContextHandler(func(c context.Context, conf *model.Configuration, args []string) error {
			return handleListFormFieldsCommand(c, conf, args, listOpts)
		}),
	}
	list.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	cmd.AddCommand(
		list,
		&cobra.Command{
			Use:   "remove inFile [ outFile ] < fieldID | fieldName >...",
			Short: "Remove form fields",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapContextHandler(handleRemoveFormFieldsCommand),
		},
		&cobra.Command{
			Use:   "lock inFile [ outFile ] [ fieldID | fieldName ]...",
			Short: "Lock form fields",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapContextHandler(handleLockFormCommand),
		},
		&cobra.Command{
			Use:   "unlock inFile [ outFile ] [ fieldID | fieldName ]...",
			Short: "Unlock form fields",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapContextHandler(handleUnlockFormCommand),
		},
		&cobra.Command{
			Use:   "reset inFile [ outFile ] [ fieldID | fieldName ]...",
			Short: "Reset form fields",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapContextHandler(handleResetFormCommand),
		},
		&cobra.Command{
			Use:   "export inFile [ outFileJSON ]",
			Short: "Export form data",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapContextHandler(handleExportFormCommand),
		},
		fill,
		multifill,
	)

	return cmd
}

func listFormFiles(c context.Context, conf *model.Configuration, args []string) ([]string, error) {
	if c == nil {
		return nil, cli.ErrMissingContext
	}
	inFiles := []string{}
	for _, arg := range args {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return nil, err
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt && arg != "-" {
			if err := ensurePDFExtension(arg); err != nil {
				return nil, err
			}
		}
		inFiles = append(inFiles, arg)
	}
	return inFiles, nil
}

func handleListFormFieldsCommand(c context.Context, conf *model.Configuration, args []string, opts *formListOptions) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFiles, err := listFormFiles(c, conf, args)
	if err != nil {
		return err
	}
	if opts.json {
		return runCommand(c, cli.ListFormFieldsJSONCommand(inFiles, conf))
	}
	return runCommand(c, cli.ListFormFieldsCommand(inFiles, conf))
}

func formFieldArgs(conf *model.Configuration, args []string, rejectPDFAsOnlyField bool) (string, string, []string, error) {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return "", "", nil, err
	}
	outFile := inFile
	if inFile == "-" {
		outFile = "-"
	}

	fieldIDs := []string{}
	if len(args) > 1 {
		if rejectPDFAsOnlyField && len(args) == 2 && hasPDFExtension(args[1]) {
			return "", "", nil, fmt.Errorf("expecting fieldID, got: %s", args[1])
		}
		if hasPDFExtension(args[1]) || args[1] == "-" {
			outFile = args[1]
			if outFile != "-" {
				if err := ensureOutputFileAvailable(outFile); err != nil {
					return "", "", nil, err
				}
			}
		} else {
			if err := validateNoEmptyArgs([]string{args[1]}, "form field ID or name"); err != nil {
				return "", "", nil, err
			}
			fieldIDs = append(fieldIDs, args[1])
		}
		for i := 2; i < len(args); i++ {
			if err := validateNoEmptyArgs([]string{args[i]}, "form field ID or name"); err != nil {
				return "", "", nil, err
			}
			fieldIDs = append(fieldIDs, args[i])
		}
	}
	return inFile, outFile, fieldIDs, nil
}

func handleRemoveFormFieldsCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile, outFile, fieldIDs, err := formFieldArgs(conf, args, true)
	if err != nil {
		return err
	}
	return runCommand(c, cli.RemoveFormFieldsCommand(inFile, outFile, fieldIDs, conf))
}

func handleLockFormCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile, outFile, fieldIDs, err := formFieldArgs(conf, args, false)
	if err != nil {
		return err
	}
	return runCommand(c, cli.LockFormCommand(inFile, outFile, fieldIDs, conf))
}

func handleUnlockFormCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile, outFile, fieldIDs, err := formFieldArgs(conf, args, false)
	if err != nil {
		return err
	}
	return runCommand(c, cli.UnlockFormCommand(inFile, outFile, fieldIDs, conf))
}

func handleResetFormCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile, outFile, fieldIDs, err := formFieldArgs(conf, args, false)
	if err != nil {
		return err
	}
	return runCommand(c, cli.ResetFormCommand(inFile, outFile, fieldIDs, conf))
}

func handleExportFormCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	outFileJSON := "out.json"
	if len(args) == 2 {
		outFileJSON = args[1]
		if err := ensureJSONExtension(outFileJSON); err != nil {
			return err
		}
	}
	if err := ensureJSONExtension(outFileJSON); err != nil {
		return err
	}
	if err := ensureOutputFileAvailable(outFileJSON); err != nil {
		return err
	}

	return runCommand(c, cli.ExportFormCommand(inFile, outFileJSON, conf))
}

func handleFillFormCommand(c context.Context, conf *model.Configuration, args []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}
	inFileJSON := args[1]
	if err := ensureJSONExtension(inFileJSON); err != nil {
		return err
	}

	outFile := inFile
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 3 {
		outFile = args[2]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return err
			}
			if err := ensureOutputFileAvailable(outFile); err != nil {
				return err
			}
		}
	}

	return runCommand(c, cli.FillFormCommand(inFile, inFileJSON, outFile, conf))
}

func multifillMode(opts *formMultifillOptions) error {
	if opts.mode == "" {
		opts.mode = "single"
	}
	opts.mode = modeCompletion(opts.mode, []string{"single", "merge"})
	if opts.mode == "" {
		return errors.New("mode must be one of: single, merge")
	}
	return nil
}

func multifillArgs(conf *model.Configuration, args []string, opts *formMultifillOptions) (string, string, string, string, error) {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return "", "", "", "", err
	}

	inFileData := args[1]
	if err := ensureJSONOrCSVExtension(inFileData); err != nil {
		return "", "", "", "", err
	}

	outDir := args[2]
	outFile := inFile
	if inFile == "-" {
		outFile = "stdin.pdf"
	}
	if len(args) == 4 {
		outFile = args[3]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return "", "", "", "", err
			}
		}
	}
	if outFile == "-" {
		if opts.mode != "merge" {
			return "", "", "", "", errors.New("form multifill stdout requires -m merge")
		}
	} else if len(args) == 4 {
		if err := ensureOutputDirOrFileAvailable(outDir, outFile); err != nil {
			return "", "", "", "", err
		}
	} else {
		if err := ensureOutputDirOrFileAvailable(outDir, ""); err != nil {
			return "", "", "", "", err
		}
	}
	return inFile, inFileData, outDir, outFile, nil
}

func handleMultiFillFormCommand(c context.Context, conf *model.Configuration, args []string, opts *formMultifillOptions) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := multifillMode(opts); err != nil {
		return err
	}
	inFile, inFileData, outDir, outFile, err := multifillArgs(conf, args, opts)
	if err != nil {
		return err
	}
	return runCommand(c, cli.MultiFillFormCommand(inFile, inFileData, outDir, outFile, opts.mode == "merge", conf))
}
