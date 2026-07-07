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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/spf13/cobra"
)

func fontsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fonts",
		Short: "Install, list supported fonts, create cheat sheets",
		Long:  usageLongFonts,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List supported fonts",
			Args:  cobra.NoArgs,
			RunE:  wrapHandler(handleListFontsCommand),
		},
		&cobra.Command{
			Use:   "install fontFiles...",
			Short: "Install fonts",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleInstallFontsCommand),
		},
		&cobra.Command{
			Use:   "cheatsheet [fontNames...]",
			Short: "Create font cheat sheets",
			RunE:  wrapHandler(handleCreateCheatSheetFontsCommand),
		},
	)

	return cmd
}

func imagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "List, extract, update images",
		Long:  usageLongImages,
	}
	addPersistentPasswordFlags(cmd)

	list := &cobra.Command{
		Use:   "list inFile...",
		Short: "List images",
		Args:  cobra.MinimumNArgs(1),
		RunE:  wrapHandler(handleListImagesCommand),
	}
	addSelectedPagesFlag(list)

	extract := &cobra.Command{
		Use:   "extract inFile outDir",
		Short: "Extract images",
		Args:  cobra.ExactArgs(2),
		RunE:  wrapHandler(handleExtractImagesCommand),
	}
	addSelectedPagesFlag(extract)

	update := &cobra.Command{
		Use:   "update inFile imageFile [ outFile ] [ objNr | (pageNr Id) ]",
		Short: "Update images",
		Args:  cobra.RangeArgs(2, 5),
		RunE:  wrapHandler(handleUpdateImagesCommand),
	}

	cmd.AddCommand(list, extract, update)

	return cmd
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [description] outFile imageFile...",
		Short: "Import/convert images to PDF",
		Long:  usageLongImportImages,
		Args:  cobra.MinimumNArgs(2),
		RunE:  wrapHandler(handleImportImagesCommand),
	}
	addUnitFlag(cmd)

	return cmd
}

func ensureImageExtension(filename string) error {
	if !model.ImageFileName(filename) {
		return fmt.Errorf("%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff, .webp)", filename)
	}
	return nil
}

func parseArgsForImageFileNames(args []string, startInd int) ([]string, error) {
	imageFileNames := []string{}
	for i := startInd; i < len(args); i++ {
		files, err := imageFileNamesForArg(args[i])
		if err != nil {
			return nil, err
		}
		imageFileNames = append(imageFileNames, files...)
	}
	return imageFileNames, nil
}

func imageFileNamesForArg(arg string) ([]string, error) {
	if strings.Contains(arg, "*") {
		return expandedImageFileNames(arg)
	}
	if arg != "-" {
		if err := ensureImageExtension(arg); err != nil {
			return nil, err
		}
	}
	return []string{arg}, nil
}

func expandedImageFileNames(arg string) ([]string, error) {
	matches, err := filepath.Glob(arg)
	if err != nil {
		return nil, err
	}
	for _, fn := range matches {
		if err := ensureImageExtension(fn); err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func defaultImageImportCommand(conf *model.Configuration, args []string) error {
	imageFileNames, err := parseArgsForImageFileNames(args, 1)
	if err != nil {
		return err
	}
	return runCommand(cli.ImportImagesCommand(imageFileNames, args[0], api.DefaultImportConfig(), conf))
}

func describedImageImportCommand(conf *model.Configuration, args []string) error {
	imp, err := api.Import(args[0], conf.Unit)
	if err != nil {
		return err
	}
	if imp == nil {
		return errors.New("missing import description")
	}

	outFile := args[1]
	if outFile != "-" {
		if err := ensurePDFExtension(outFile); err != nil {
			return err
		}
	}
	imageFileNames, err := parseArgsForImageFileNames(args, 2)
	if err != nil {
		return err
	}
	return runCommand(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
}

func handleImportImagesCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}
	var outFile string
	outFile = args[0]
	if hasPDFExtension(outFile) || outFile == "-" {
		return defaultImageImportCommand(conf, args)
	}
	return describedImageImportCommand(conf, args)
}

func handleListFontsCommand(conf *model.Configuration, args []string) error {
	return runCommand(cli.ListFontsCommand(conf))
}

func handleInstallFontsCommand(conf *model.Configuration, args []string) error {
	return runCommand(cli.InstallFontsCommand(args, conf))
}

func handleCreateCheatSheetFontsCommand(conf *model.Configuration, args []string) error {
	if err := validateNoEmptyArgs(args, "font name"); err != nil {
		return err
	}
	return runCommand(cli.CreateCheatSheetsFontsCommand(args, conf))
}

func handleListImagesCommand(conf *model.Configuration, args []string) error {
	inFiles, err := infoInputFiles(conf, args)
	if err != nil {
		return err
	}
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	return runCommand(cli.ListImagesCommand(inFiles, selectedPages, conf))
}

func handleExtractImagesCommand(conf *model.Configuration, args []string) error {
	// See also handleExtractCommand
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}
	outDir := args[1]
	if err := ensureOutputDirEmpty(outDir); err != nil {
		return err
	}

	pages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	return runCommand(cli.ExtractImagesCommand(inFile, outDir, pages, conf))
}

func handleUpdateImagesCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	imageFile := args[1]
	if err := ensureImageExtension(imageFile); err != nil {
		return err
	}

	outFile, objNrOrPageNr, id, err := updateImageArgs(args)
	if err != nil {
		return err
	}

	return runCommand(cli.UpdateImagesCommand(inFile, imageFile, outFile, objNrOrPageNr, id, conf))
}

func updateImageArgs(args []string) (string, int, string, error) {
	outFile := ""
	objNrOrPageNr := 0
	id := ""

	argCount := len(args)
	if argCount > 2 {
		c := 2
		if hasPDFExtension(args[2]) || args[2] == "-" {
			outFile = args[2]
			if outFile != "-" {
				if err := ensureOutputFileAvailable(outFile); err != nil {
					return "", 0, "", err
				}
			}
			c++
		}
		if argCount > c {
			i, err := strconv.Atoi(args[c])
			if err != nil {
				return "", 0, "", err
			}
			if i <= 0 {
				return "", 0, "", errors.New("objNr & pageNr must be > 0")
			}
			objNrOrPageNr = i
			if argCount == c+2 {
				id = args[c+1]
			}
		}
	}

	return outFile, objNrOrPageNr, id, nil
}
func attachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "List, add, remove, extract embedded file attachments",
		Long:  usageLongAttach,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List attachments",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "add inFile file [ , desc ]...",
			Short: "Add attachments",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleAddAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [ file... ]",
			Short: "Remove attachments",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleRemoveAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "extract inFile outDir [ file... ]",
			Short: "Extract attachments",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleExtractAttachmentsCommand),
		},
	)

	return cmd
}

func portfolioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "List, add, remove, extract portfolio entries",
		Long:  usageLongPortfolio,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List portfolio entries",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "add inFile file [ , desc ]...",
			Short: "Add portfolio entries",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleAddAttachmentsPortfolioCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [ file... ]",
			Short: "Remove portfolio entries",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleRemoveAttachmentsCommand),
		},
		&cobra.Command{
			Use:   "extract inFile outDir [ file... ]",
			Short: "Extract portfolio entries",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleExtractAttachmentsCommand),
		},
	)

	return cmd
}

func validateAttachmentArg(arg string) error {
	fileName := strings.TrimSpace(strings.SplitN(arg, ",", 2)[0])
	if fileName == "" {
		return errors.New("attachment filename must not be empty")
	}
	return nil
}

var errNoAttachmentGlobMatches = errors.New("no matches")

func hasAttachmentGlobMeta(s string) bool {
	meta := "*?["
	if os.PathSeparator != '\\' {
		meta += `\`
	}
	return strings.ContainsAny(s, meta)
}

func attachmentFiles(args []string, expandGlobs bool, op string) ([]string, error) {
	fileNames := []string{}
	for _, arg := range args {
		if err := validateAttachmentArg(arg); err != nil {
			return nil, fmt.Errorf("%s: validate attachment filename: %w", op, err)
		}
		parts := strings.SplitN(arg, ",", 2)
		if expandGlobs && hasAttachmentGlobMeta(parts[0]) {
			matches, err := filepath.Glob(parts[0])
			if err != nil {
				return nil, fmt.Errorf("%s: expand attachment glob %q: %w", op, parts[0], err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("%s: expand attachment glob %q: %w", op, parts[0], errNoAttachmentGlobMatches)
			}
			for _, match := range matches {
				if len(parts) == 2 {
					match += "," + parts[1]
				}
				fileNames = append(fileNames, match)
			}
			continue
		}
		fileNames = append(fileNames, arg)
	}
	return fileNames, nil
}

func handleListAttachmentsCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("list attachments: validate input: %w", err)
	}
	return runCommand(cli.ListAttachmentsCommand(inFile, conf))
}

func handleAddAttachmentsCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}
	if len(args) < 2 {
		return fmt.Errorf("add attachments: %w", api.ErrNoAttachmentAdded)
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("add attachments: validate input: %w", err)
	}
	fileNames, err := attachmentFiles(args[1:], true, "add attachments")
	if err != nil {
		return err
	}
	return runCommand(cli.AddAttachmentsCommand(inFile, stdoutForStdin(inFile), fileNames, conf))
}

func handleAddAttachmentsPortfolioCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}
	if len(args) < 2 {
		return fmt.Errorf("add portfolio attachments: %w", api.ErrNoAttachmentAdded)
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("add portfolio attachments: validate input: %w", err)
	}
	fileNames, err := attachmentFiles(args[1:], true, "add portfolio attachments")
	if err != nil {
		return err
	}
	return runCommand(cli.AddAttachmentsPortfolioCommand(inFile, stdoutForStdin(inFile), fileNames, conf))
}

func handleRemoveAttachmentsCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("remove attachments: validate input: %w", err)
	}
	if err := validateNoEmptyArgs(args[1:], "attachment filename"); err != nil {
		return fmt.Errorf("remove attachments: validate attachment filenames: %w", err)
	}
	return runCommand(cli.RemoveAttachmentsCommand(inFile, stdoutForStdin(inFile), args[1:], conf))
}

func handleExtractAttachmentsCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}
	if len(args) < 2 {
		return api.ErrMissingPDFOutput
	}
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("extract attachments: validate input: %w", err)
	}
	outDir := args[1]
	if err := validateNoEmptyArgs(args[2:], "attachment filename"); err != nil {
		return fmt.Errorf("extract attachments: validate attachment filenames: %w", err)
	}
	if err := ensureOutputDirEmpty(outDir); err != nil {
		return fmt.Errorf("extract attachments: prepare output directory: %w", err)
	}
	return runCommand(cli.ExtractAttachmentsCommand(inFile, outDir, args[2:], conf))
}
func keywordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "List, add, remove keywords",
		Long:  usageLongKeywords,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List keywords",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListKeywordsCommand),
		},
		&cobra.Command{
			Use:   "add inFile [ outFile ] keyword...",
			Short: "Add keywords",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleAddKeywordsCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [ outFile ] [ keyword... ]",
			Short: "Remove keywords",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleRemoveKeywordsCommand),
		},
	)

	return cmd
}

func propertiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "properties",
		Short: "List, add, remove document properties",
		Long:  usageLongProperties,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List properties",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListPropertiesCommand),
		},
		&cobra.Command{
			Use:   "add inFile [ outFile ] nameValuePair...",
			Short: "Add properties",
			Args:  cobra.MinimumNArgs(2),
			RunE:  wrapHandler(handleAddPropertiesCommand),
		},
		&cobra.Command{
			Use:   "remove inFile [ outFile ] [ name... ]",
			Short: "Remove properties",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleRemovePropertiesCommand),
		},
	)

	return cmd
}

func metadataArgs(conf *model.Configuration, args []string) (string, string, []string, error) {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return "", "", nil, err
	}

	outFile := ""
	start := 1
	if len(args) > 1 && (hasPDFExtension(args[1]) || args[1] == "-") {
		outFile = args[1]
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return "", "", nil, err
		}
		start = 2
	}

	return inFile, outFile, args[start:], nil
}

func handleListKeywordsCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	return runCommand(cli.ListKeywordsCommand(inFile, conf))
}

func handleAddKeywordsCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, keywords, err := metadataArgs(conf, args)
	if err != nil {
		return err
	}
	if err := validateNoEmptyArgs(keywords, "keyword"); err != nil {
		return err
	}
	return runCommand(cli.AddKeywordsCommand(inFile, outFile, keywords, conf))
}

func handleRemoveKeywordsCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, keywords, err := metadataArgs(conf, args)
	if err != nil {
		return err
	}
	if err := validateNoEmptyArgs(keywords, "keyword"); err != nil {
		return err
	}
	return runCommand(cli.RemoveKeywordsCommand(inFile, outFile, keywords, conf))
}

func handleListPropertiesCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	return runCommand(cli.ListPropertiesCommand(inFile, conf))
}

func parsePropertyAssignment(arg string) (string, string, error) {
	ss := strings.SplitN(arg, "=", 2)
	if len(ss) != 2 {
		return "", "", errors.New("keyValuePair = 'key = value'")
	}
	k := strings.TrimSpace(ss[0])
	if k == "" {
		return "", "", errors.New("property name must not be empty")
	}
	if !validate.DocumentProperty(k) {
		return "", "", fmt.Errorf("property name \"%s\" not allowed", k)
	}
	v := strings.TrimSpace(ss[1])
	if v == "" {
		return "", "", errors.New("property value must not be empty")
	}
	return k, v, nil
}

func properties(args []string) (map[string]string, error) {
	properties := map[string]string{}
	for _, arg := range args {
		k, v, err := parsePropertyAssignment(arg)
		if err != nil {
			return nil, err
		}
		properties[k] = v
	}
	return properties, nil
}

func handleAddPropertiesCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, propertyArgs, err := metadataArgs(conf, args)
	if err != nil {
		return err
	}
	properties, err := properties(propertyArgs)
	if err != nil {
		return err
	}
	return runCommand(cli.AddPropertiesCommand(inFile, outFile, properties, conf))
}

func propertyKeys(args []string) ([]string, error) {
	keys := []string{}
	for _, arg := range args {
		k := strings.TrimSpace(arg)
		if k == "" {
			return nil, errors.New("property name must not be empty")
		}
		if !validate.DocumentProperty(k) {
			return nil, fmt.Errorf("property name \"%s\" not allowed", k)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func handleRemovePropertiesCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, keyArgs, err := metadataArgs(conf, args)
	if err != nil {
		return err
	}
	keys, err := propertyKeys(keyArgs)
	if err != nil {
		return err
	}
	return runCommand(cli.RemovePropertiesCommand(inFile, outFile, keys, conf))
}
