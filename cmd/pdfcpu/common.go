/*
Copyright 2025 The pdfcpu Authors.

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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/spf13/cobra"
)

const (
	selectedPagesWarn = "-selectedPages problem"
	pdfcpuErrPrefix   = "pdfcpu: "
)

func wrapHandler(handler func(*model.Configuration, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		conf, err := getConfig()
		if err != nil {
			return commandError(err)
		}
		if conf.Version != model.VersionStr {
			model.CheckConfigVersion(conf.Version)
		}
		return commandError(handler(conf, args))
	}
}

func commandError(err error) error {
	if err == nil {
		return nil
	}
	if !strings.HasPrefix(err.Error(), pdfcpuErrPrefix) {
		return err
	}
	return prefixStrippedError{err: err}
}

type prefixStrippedError struct {
	err error
}

func (e prefixStrippedError) Error() string {
	return strings.TrimPrefix(e.err.Error(), pdfcpuErrPrefix)
}

func (e prefixStrippedError) Unwrap() error {
	return e.err
}

func addPasswordFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&upw, "upw", "", "user password")
	cmd.Flags().StringVar(&opw, "opw", "", "owner password")
}

func addPersistentPasswordFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&upw, "upw", "", "user password")
	cmd.PersistentFlags().StringVar(&opw, "opw", "", "owner password")
}

func addSelectedPagesFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
}

func addPersistentSelectedPagesFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&selectedPages, "pages", "p", "", "pages to process")
}

func addRequiredSelectedPagesFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&selectedPages, "pages", "p", "", "pages to process (required)")
	cmd.MarkFlagRequired("pages")
}

func addUnitFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&unit, "unit", "u", "", "display unit: po(ints) | in(ches) | cm | mm")
}

func modeCompletion(modePrefix string, modes []string) string {
	var modeStr string
	for _, mode := range modes {
		if !strings.HasPrefix(mode, modePrefix) {
			continue
		}
		if len(modeStr) > 0 {
			return ""
		}
		modeStr = mode
	}
	return modeStr
}

func parseSelectedPages() ([]string, error) {
	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", selectedPagesWarn, err)
	}
	return selectedPages, nil
}

func hasPDFExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func ensurePDFExtension(filename string) error {
	if !hasPDFExtension(filename) {
		return fmt.Errorf("%s needs extension \".pdf\".", filename)
	}
	return nil
}

func hasJSONExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".json")
}

func ensureJSONExtension(filename string) error {
	if !hasJSONExtension(filename) {
		return fmt.Errorf("%s needs extension \".json\".", filename)
	}
	return nil
}

func hasCSVExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".csv")
}

func ensureJSONOrCSVExtension(filename string) error {
	if !hasJSONExtension(filename) && !hasCSVExtension(filename) {
		return fmt.Errorf("%s needs extension \".json\" or \".csv\".", filename)
	}
	return nil
}

func ensureOutputFileAvailable(outFile string) error {
	if outFile == "" || outFile == "-" || force {
		return nil
	}

	if _, err := os.Stat(outFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return fmt.Errorf("refusing to overwrite existing file: %s\nUse --force to overwrite.", outFile)
}

func ensureOutputDirEmpty(outDir string) error {
	if outDir == "" || outDir == "-" || force {
		return nil
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	return fmt.Errorf("refusing to write to non-empty directory: %s\nUse --force to write anyway.", outDir)
}

func ensureOutputDirOrFileAvailable(outDir, outFile string) error {
	if outFile == "" {
		return ensureOutputDirEmpty(outDir)
	}
	return ensureOutputFileAvailable(filepath.Join(outDir, outFile))
}

func inputOutputPDFArgs(conf *model.Configuration, args []string) (string, string, error) {
	inFile := args[0]
	if conf.CheckFileNameExt && inFile != "-" {
		if err := ensurePDFExtension(inFile); err != nil {
			return "", "", err
		}
	}

	outFile := inFile
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 2 {
		outFile = args[1]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return "", "", err
			}
		}
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return "", "", err
		}
	}

	return inFile, outFile, nil
}

func optionalOutputPDFArgs(conf *model.Configuration, args []string) (string, string, error) {
	inFile := args[0]
	if conf.CheckFileNameExt && inFile != "-" {
		if err := ensurePDFExtension(inFile); err != nil {
			return "", "", err
		}
	}

	outFile := ""
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 2 {
		outFile = args[1]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return "", "", err
			}
		}
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return "", "", err
		}
	}

	return inFile, outFile, nil
}

func inputPDFArg(conf *model.Configuration, inFile string) error {
	if conf.CheckFileNameExt && inFile != "-" {
		return ensurePDFExtension(inFile)
	}
	return nil
}

func stdoutForStdin(inFile string) string {
	if inFile == "-" {
		return "-"
	}
	return ""
}

func validateNoEmptyArgs(args []string, name string) error {
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}

func configureDisplayUnit(conf *model.Configuration) error {
	if !types.MemberOf(unit, []string{"", "points", "po", "inches", "in", "cm", "mm"}) {
		return errors.New("supported units: (po)ints, (in)ches, cm, mm")
	}

	switch unit {
	case "points", "po":
		conf.Unit = types.POINTS
	case "inches", "in":
		conf.Unit = types.INCHES
	case "cm":
		conf.Unit = types.CENTIMETRES
	case "mm":
		conf.Unit = types.MILLIMETRES
	}
	return nil
}

type commandDispatch func(*cli.Command) ([]string, error)

func runCommandWithOutput(cmd *cli.Command, w io.Writer, dispatch commandDispatch, suppressOutput bool) error {
	if cmd == nil {
		return errors.New("pdfcpu: missing command")
	}
	out, dispatchErr := dispatch(cmd)
	var writeErr error
	if out != nil && !suppressOutput {
		for i, s := range out {
			if _, err := fmt.Fprintln(w, s); err != nil {
				writeErr = fmt.Errorf("write command output line %d: %w", i+1, err)
				break
			}
		}
	}
	return errors.Join(dispatchErr, writeErr)
}

// runCommand dispatches a CLI command and writes command output.
func runCommand(cmd *cli.Command) error {
	return runCommandWithOutput(cmd, os.Stdout, cli.Dispatch, quiet)
}
