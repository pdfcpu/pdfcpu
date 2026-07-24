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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type certificatesListOptions struct {
	json bool
}

type signaturesValidateOptions struct {
	all  bool
	full bool
}

func certificatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certificates",
		Short: "List, inspect, import, reset certificates",
		Long:  usageLongCertificates,
	}

	listOpts := &certificatesListOptions{json: false}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List certificates",
		Long:  usageLongCertificatesList,
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleListCertificatesCommand(conf, args, listOpts)
		}),
	}
	listCmd.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	cmd.AddCommand(
		listCmd,
		&cobra.Command{
			Use:   "inspect inFile",
			Short: "Inspect certificates",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleInspectCertificatesCommand),
		},
		&cobra.Command{
			Use:   "import inFile...",
			Short: "Import certificates, replacing matching installed files",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleImportCertificatesCommand),
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Reset certificates",
			RunE:  wrapHandler(resetCertificates),
		},
	)

	return cmd
}

func signaturesRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove inFile [ outFile ]",
		Short: "Remove signatures",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleRemoveSignaturesCommand),
	}
	cmd.Flags().BoolVar(&removeEncryption, "rmenc", false, "remove encryption")
	return cmd
}

func signaturesValidateCmd() *cobra.Command {
	opts := &signaturesValidateOptions{all: false, full: false}
	cmd := &cobra.Command{
		Use:   "validate inFile",
		Short: "Validate signature integrity",
		Long:  usageLongSignaturesValidate,
		Args:  cobra.ExactArgs(1),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleValidateSignaturesCommand(conf, args, opts)
		}),
	}
	cmd.Flags().BoolVarP(&opts.all, "all", "a", opts.all, "validate all signatures")
	cmd.Flags().BoolVarP(&opts.full, "full", "f", opts.full, "detailed output")
	return cmd
}

func signaturesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signatures",
		Short: "Remove signatures, validate integrity",
		Long:  usageLongSignatures,
	}
	addPersistentPasswordFlags(cmd)
	cmd.AddCommand(signaturesRemoveCmd(), signaturesValidateCmd())
	return cmd
}

func resetCertificates(conf *model.Configuration, args []string) error {
	fmt.Println("Are you ready to reset your trusted certificates to the build defaults?")
	if confirmed() {
		fmt.Println("resetting..")
		if err := model.ResetCertificates(); err != nil {
			return fmt.Errorf("config problem: %v", err)
		}
		fmt.Println("Finished")
	} else {
		fmt.Println("Operation canceled")
	}
	return nil
}

func handleListCertificatesCommand(conf *model.Configuration, args []string, opts *certificatesListOptions) error {
	if opts.json {
		log.SetCLILogger(nil)
	}

	return runCommand(cli.ListCertificatesCommand(opts.json, conf))
}

func isCertificateFile(fName string) bool {
	for _, ext := range []string{".p7c", ".pem", ".cer", ".crt"} {
		if strings.HasSuffix(strings.ToLower(fName), ext) {
			return true
		}
	}
	return false
}

func certificateFiles(args []string) ([]string, error) {
	var inFiles []string
	for _, arg := range args {
		files, err := certificateFilesForArg(arg)
		if err != nil {
			return nil, err
		}
		inFiles = append(inFiles, files...)
	}
	return inFiles, nil
}

func certificateFilesForArg(arg string) ([]string, error) {
	if strings.Contains(arg, "*") {
		return expandedCertificateFiles(arg)
	}
	if !isCertificateFile(arg) {
		return nil, fmt.Errorf("%s - allowed extensions: .pem, .p7c, .cer, .crt", arg)
	}
	return []string{arg}, nil
}

func expandedCertificateFiles(arg string) ([]string, error) {
	matches, err := filepath.Glob(arg)
	if err != nil {
		return nil, err
	}
	var inFiles []string
	for _, inFile := range matches {
		if !isCertificateFile(inFile) {
			fmt.Fprintf(os.Stderr, "skipping %s - allowed extensions: .pem, .p7c, .cer, .crt\n", inFile)
			continue
		}
		inFiles = append(inFiles, inFile)
	}
	return inFiles, nil
}

func handleInspectCertificatesCommand(conf *model.Configuration, args []string) error {
	inFiles, err := certificateFiles(args)
	if err != nil {
		return err
	}
	return runCommand(cli.InspectCertificatesCommand(inFiles, conf))
}

func handleImportCertificatesCommand(conf *model.Configuration, args []string) error {
	inFiles, err := certificateFiles(args)
	if err != nil {
		return err
	}
	return runCommand(cli.ImportCertificatesCommand(inFiles, conf))
}

func handleValidateSignaturesCommand(conf *model.Configuration, args []string, opts *signaturesValidateOptions) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	return runCommand(cli.ValidateSignaturesCommand(inFile, opts.all, opts.full, conf))
}

func handleRemoveSignaturesCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, err := inputOutputPDFArgs(conf, args)
	if err != nil {
		return err
	}
	return runCommand(cli.RemoveSignaturesCommand(inFile, outFile, conf))
}
