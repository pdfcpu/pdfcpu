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
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
)

// ImportImages turns image files into a page sequence and writes the result to outFile.
// In its simplest form this operation converts an image into a PDF.
func ImportImages(cmd *Command) (result []string, err error) {
	if err := validateImportImagesCommand(cmd); err != nil {
		return nil, err
	}
	stdinImage, err := hasStdinImage(cmd.InFiles)
	if err != nil {
		return nil, err
	}
	imp, err := api.PrepareImportConfiguration(cmd.Import)
	if err != nil {
		return nil, err
	}
	conf := cmd.Conf
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	if *cmd.OutFile != "-" && stdinImage {
		if err := api.ValidateImportImagesOutput(cmd.InFiles, *cmd.OutFile); err != nil {
			return nil, err
		}
	}
	if *cmd.OutFile != "-" && !stdinImage {
		return nil, api.ImportImagesFile(cmd.InFiles, *cmd.OutFile, imp, conf)
	}

	readers, closers, err := importImageReaders(cmd.InFiles, conf.Limits.MaxStreamBytes)
	if err != nil {
		return nil, err
	}

	if *cmd.OutFile == "-" {
		defer func() {
			err = errors.Join(err, closeImportImageInputs(closers))
		}()
		log.SetCLILogger(nil)
		return nil, api.ImportImages(nil, os.Stdout, readers, imp, conf)
	}

	return nil, importImagesToFile(*cmd.OutFile, readers, closers, imp, conf)
}

func validateImportImagesCommand(cmd *Command) error {
	const operation = "import images"
	if err := validateCommandRequirements(cmd, commandRequirements{operation: operation}); err != nil {
		return err
	}
	if err := validateCommandInputFiles(cmd.InFiles, 1, 0, api.ErrMissingImageInput); err != nil {
		return commandValidationError(operation, err)
	}
	err := validateCommandString(cmd.OutFile, commandStringRequiredNonEmpty, api.ErrMissingPDFOutput)
	return commandValidationError(operation, err)
}

func hasStdinImage(inFiles []string) (bool, error) {
	stdinImage := false
	for _, fn := range inFiles {
		if fn == "-" {
			if stdinImage {
				return false, errors.New("import images: only one image may read from stdin")
			}
			stdinImage = true
		}
	}
	return stdinImage, nil
}

func readImportImageStdin(r io.Reader, imageIndex int, maxStreamBytes int64) (io.Reader, error) {
	if r == nil {
		return nil, fmt.Errorf("import images: image %d stdin: read: missing reader", imageIndex)
	}
	if maxStreamBytes < 0 {
		return nil, fmt.Errorf(
			"import images: image %d stdin: read: invalid stream byte limit %d",
			imageIndex,
			maxStreamBytes,
		)
	}
	readLimit := maxStreamBytes
	if maxStreamBytes < math.MaxInt64 {
		readLimit++
	}
	bb, err := io.ReadAll(io.LimitReader(r, readLimit))
	if err != nil {
		return nil, fmt.Errorf("import images: image %d stdin: read: %w", imageIndex, err)
	}
	if int64(len(bb)) > maxStreamBytes {
		return nil, fmt.Errorf(
			"import images: image %d stdin: read: input size %d exceeds limit %d",
			imageIndex,
			len(bb),
			maxStreamBytes,
		)
	}
	if len(bb) == 0 {
		return nil, fmt.Errorf("import images: image %d stdin: read: stdin is empty", imageIndex)
	}
	return bytes.NewReader(bb), nil
}

func importImageReader(fn string, imageIndex int, maxStreamBytes int64) (io.Reader, io.Closer, error) {
	if fn != "-" {
		f, err := os.Open(fn)
		if err != nil {
			return nil, nil, fmt.Errorf("import images: image %d %q: open: %w", imageIndex, fn, err)
		}
		return f, f, nil
	}

	r, err := readImportImageStdin(os.Stdin, imageIndex, maxStreamBytes)
	return r, nil, err
}

type importImageInputCloser struct {
	closer     io.Closer
	imageIndex int
	fileName   string
}

func importImageReaders(inFiles []string, maxStreamBytes int64) ([]io.Reader, []importImageInputCloser, error) {
	readers := make([]io.Reader, 0, len(inFiles))
	closers := make([]importImageInputCloser, 0, len(inFiles))
	for i, fn := range inFiles {
		r, c, err := importImageReader(fn, i+1, maxStreamBytes)
		if err != nil {
			return nil, nil, errors.Join(err, closeImportImageInputs(closers))
		}
		readers = append(readers, r)
		if c != nil {
			closers = append(closers, importImageInputCloser{
				closer:     c,
				imageIndex: i + 1,
				fileName:   fn,
			})
		}
	}
	return readers, closers, nil
}

func closeImportImageInputs(closers []importImageInputCloser) error {
	errs := make([]error, 0, len(closers))
	for _, c := range closers {
		if err := c.closer.Close(); err != nil {
			errs = append(
				errs,
				fmt.Errorf("import images: image %d %q: close: %w", c.imageIndex, c.fileName, err),
			)
		}
	}
	return errors.Join(errs...)
}

func importImagesDestination(outFile string) (*os.File, error) {
	f, err := os.Open(outFile)
	if err == nil {
		return f, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, fmt.Errorf("import images: inspect output %s: %w", outFile, err)
}

func createImportImagesStreamOutput(outFile string, replace bool) (*os.File, string, string, error) {
	if replace {
		return createStreamOutput(outFile)
	}
	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	return f, outFile, "", err
}

func importImagesToFile(
	outFile string,
	readers []io.Reader,
	closers []importImageInputCloser,
	imp *pdfcpu.Import,
	conf *model.Configuration,
) (err error) {
	f1, err := importImagesDestination(outFile)
	if err != nil {
		return errors.Join(err, closeImportImageInputs(closers))
	}
	f2, tmpFile, replaceOut, err := createImportImagesStreamOutput(outFile, f1 != nil)
	if err != nil {
		return errors.Join(
			fmt.Errorf("import images: create output: %w", err),
			closeStreamFile(f1, "import images: close input"),
			closeImportImageInputs(closers),
		)
	}

	finalizer := streamInOutFinalizer{
		input:      f1,
		output:     f2,
		outFile:    tmpFile,
		replaceOut: replaceOut,
	}
	defer func() {
		err = errors.Join(err, closeImportImageInputs(closers))
		err = finalizer.finalize("import images", err)
	}()

	return api.ImportImages(f1, f2, readers, imp, conf)
}

// CreateCheatSheetsFonts creates single page PDF cheat sheets for user fonts in current dir.
func CreateCheatSheetsFonts(cmd *Command) ([]string, error) {
	if err := validateFontsCommand(cmd, model.CHEATSHEETSFONTS); err != nil {
		return nil, err
	}
	return nil, api.CreateCheatSheetsUserFonts(cmd.InFiles)
}

func validateFontsCommand(cmd *Command, expectedMode model.CommandMode) error {
	if cmd == nil {
		return commandValidationError("fonts command", ErrMissingCommand)
	}
	switch expectedMode {
	case model.LISTFONTS, model.INSTALLFONTS, model.CHEATSHEETSFONTS:
	default:
		return fmt.Errorf("fonts command: inappropriate expected mode %d", expectedMode)
	}
	if cmd.Mode != expectedMode {
		return fmt.Errorf("fonts command: mode %d, expected %d", cmd.Mode, expectedMode)
	}
	return nil
}

// ListFonts gathers information about supported fonts and returns the result as []string.
func ListFonts(cmd *Command) ([]string, error) {
	if err := validateFontsCommand(cmd, model.LISTFONTS); err != nil {
		return nil, err
	}
	return api.ListFonts()
}

// InstallFonts installs True Type fonts into the pdfcpu pconfig dir.
func InstallFonts(cmd *Command) ([]string, error) {
	if err := validateFontsCommand(cmd, model.INSTALLFONTS); err != nil {
		return nil, err
	}
	return nil, api.InstallFonts(cmd.InFiles)
}

var (
	openListImagesInput  = os.Open
	closeListImagesInput = (*os.File).Close
)

func listImagesFile(inFile string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if inFile == "-" {
		return withStdinReadSeeker("list images", func(rs io.ReadSeeker) ([]string, error) {
			output, err := api.ListImages(rs, selectedPages, conf)
			if err != nil {
				return nil, fmt.Errorf("stdin: %w", err)
			}
			return output, nil
		})
	}
	f, err := openListImagesInput(inFile)
	if err != nil {
		return nil, fmt.Errorf("list images: open input: %w", err)
	}
	output, opErr := api.ListImages(f, selectedPages, conf)
	if opErr != nil {
		opErr = fmt.Errorf("%s: %w", inFile, opErr)
	}
	closeErr := closeListImagesInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list images: close input %s: %w", inFile, closeErr)
	}
	return output, errors.Join(opErr, closeErr)
}

func validateListImagesInputs(inFiles []string) error {
	if err := validateCommandInputFiles(inFiles, 1, 0, nil); err != nil {
		return commandValidationError("list images", err)
	}
	stdinCount := 0
	for _, inFile := range inFiles {
		if inFile == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		return fmt.Errorf("list images: only one input may read from stdin")
	}
	return nil
}

// ListImagesFile returns a formatted list of embedded images of inFile.
func ListImagesFile(inFiles []string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if err := validateListImagesInputs(inFiles); err != nil {
		return nil, err
	}
	if len(selectedPages) == 0 {
		log.CLI.Printf("pages: all\n")
	}

	log.SetCLILogger(nil)

	ss := []string{}
	var errs []error

	for _, fn := range inFiles {
		output, err := listImagesFile(fn, selectedPages, conf)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, err)
				continue
			}
			return nil, err
		}
		label := fn
		if label == "-" {
			label = "stdin"
		}
		ss = append(ss, "\n"+label+":")
		ss = append(ss, output...)
	}

	return ss, errors.Join(errs...)
}

func validateListImagesCommand(cmd *Command) error {
	if err := validateCommandRequirements(cmd, commandRequirements{operation: "list images"}); err != nil {
		return err
	}
	return validateListImagesInputs(cmd.InFiles)
}

// ListImages returns inFiles embedded images.
func ListImages(cmd *Command) ([]string, error) {
	if err := validateListImagesCommand(cmd); err != nil {
		return nil, err
	}
	return ListImagesFile(cmd.InFiles, cmd.PageSelection, cmd.Conf)
}

func updateImageParams(cmd *Command) (objNr, pageNr int, id string) {
	if cmd.StringVal != "" {
		return 0, cmd.IntVal, cmd.StringVal
	}
	return cmd.IntVal, 0, ""
}

func validateUpdateImagesCommand(cmd *Command) error {
	if err := validateCommandRequirements(cmd, commandRequirements{operation: "update images"}); err != nil {
		return err
	}
	if len(cmd.InFiles) == 0 || cmd.InFiles[0] == "" {
		return commandValidationError("update images", api.ErrMissingPDFInput)
	}
	if len(cmd.InFiles) < 2 || cmd.InFiles[1] == "" {
		return commandValidationError("update images", api.ErrMissingImageInput)
	}
	if len(cmd.InFiles) != 2 {
		return fmt.Errorf(
			"update images: expected exactly two inputs, got %d: %w",
			len(cmd.InFiles),
			ErrInvalidCommandArguments,
		)
	}
	return commandValidationError(
		"update images",
		validateCommandString(cmd.OutFile, commandStringRequired, api.ErrMissingPDFOutput),
	)
}

func updateImagesInOut(cmd *Command, objNr, pageNr int, id string) ([]string, error) {
	if cmd.InFiles[0] != "-" && *cmd.OutFile != "-" {
		return nil, api.UpdateImagesFile(cmd.InFiles[0], cmd.InFiles[1], *cmd.OutFile, objNr, pageNr, id, cmd.Conf)
	}

	imageFile := cmd.InFiles[1]
	f, err := os.Open(imageFile)
	if err != nil {
		return nil, fmt.Errorf("update images: open image %s: %w", imageFile, err)
	}
	closeImage := func() error {
		return closeStreamFile(f, fmt.Sprintf("update images: close image %s", imageFile))
	}
	if *cmd.OutFile != "-" {
		if err := api.ValidateUpdateImagesOutput(imageFile, *cmd.OutFile); err != nil {
			return nil, errors.Join(err, closeImage())
		}
	}
	rs, w, finalize, err := streamInOutForOperation(cmd.InFiles[0], *cmd.OutFile, "update images")
	if err != nil {
		return nil, errors.Join(err, closeImage())
	}
	opErr := api.UpdateImages(rs, f, w, objNr, pageNr, id, cmd.Conf)
	return nil, finalize(errors.Join(opErr, closeImage()))
}

// UpdateImages replaces image objects.
func UpdateImages(cmd *Command) ([]string, error) {
	if err := validateUpdateImagesCommand(cmd); err != nil {
		return nil, err
	}
	objNr, pageNr, id := updateImageParams(cmd)
	return updateImagesInOut(cmd, objNr, pageNr, id)
}
func listAttachments(rs io.ReadSeeker, conf *model.Configuration, withDesc, sorted bool) ([]string, error) {
	aa, err := api.Attachments(rs, conf)
	if err != nil {
		return nil, err
	}

	var ss []string
	for _, a := range aa {
		s := a.FileName
		if withDesc && a.Desc != "" {
			s = fmt.Sprintf("%s (%s)", s, a.Desc)
		}
		ss = append(ss, s)
	}
	if sorted {
		sort.Strings(ss)
	}

	return ss, nil
}

var closeListAttachmentsInput = (*os.File).Close

// ListAttachmentsFile returns a list of embedded file attachments of inFile with optional description.
func ListAttachmentsFile(inFile string, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list attachments: open input %s: %w", inFile, err)
	}

	ss, opErr := listAttachments(f, conf, true, true)
	closeErr := closeListAttachmentsInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list attachments: close input %s: %w", inFile, closeErr)
	}
	return ss, errors.Join(opErr, closeErr)
}

// ListAttachmentsCompactFile returns a list of embedded file attachments of inFile w/o optional description.
func ListAttachmentsCompactFile(inFile string, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list attachments: open input %s: %w", inFile, err)
	}

	ss, opErr := listAttachments(f, conf, false, false)
	closeErr := closeListAttachmentsInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list attachments: close input %s: %w", inFile, closeErr)
	}
	return ss, errors.Join(opErr, closeErr)
}

func validateListAttachmentsCommand(cmd *Command) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: "list attachments",
		inFile:    commandStringRequiredNonEmpty,
	})
}

func validateMutateAttachmentsCommand(cmd *Command, operation string) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	})
}

func validateExtractAttachmentsCommand(cmd *Command) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: "extract attachments",
		inFile:    commandStringRequiredNonEmpty,
		outDir:    commandStringRequiredNonEmpty,
	})
}

// ListAttachments returns a list of embedded file attachments for inFile.
func ListAttachments(cmd *Command) ([]string, error) {
	if err := validateListAttachmentsCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("list attachments", func(rs io.ReadSeeker) ([]string, error) {
			return listAttachments(rs, cmd.Conf, true, true)
		})
	}

	return ListAttachmentsFile(*cmd.InFile, cmd.Conf)
}

// AddAttachments embeds inFiles into a PDF context read from inFile and writes the result to outFile.
func AddAttachments(cmd *Command) ([]string, error) {
	if err := validateMutateAttachmentsCommand(cmd, "add attachments"); err != nil {
		return nil, err
	}
	op := "add attachments"
	if cmd.Mode == model.ADDATTACHMENTSPORTFOLIO {
		op = "add portfolio attachments"
	}
	if *cmd.InFile == "-" || *cmd.OutFile == "-" {
		rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, op)
		if err != nil {
			return nil, err
		}
		opErr := api.AddAttachments(rs, w, cmd.InFiles, cmd.Mode == model.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
		return nil, finalize(opErr)
	}

	return nil, api.AddAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Mode == model.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
}

// RemoveAttachments deletes inFiles from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachments(cmd *Command) ([]string, error) {
	if err := validateMutateAttachmentsCommand(cmd, "remove attachments"); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" || *cmd.OutFile == "-" {
		rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "remove attachments")
		if err != nil {
			return nil, err
		}
		opErr := api.RemoveAttachments(rs, w, cmd.InFiles, cmd.Conf)
		return nil, finalize(opErr)
	}

	return nil, api.RemoveAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// ExtractAttachments extracts inFiles from a PDF context read from inFile and writes the result to outFile.
func ExtractAttachments(cmd *Command) ([]string, error) {
	if err := validateExtractAttachmentsCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("extract attachments", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.ExtractAttachments(rs, *cmd.OutDir, cmd.InFiles, cmd.Conf)
		})
	}

	return nil, api.ExtractAttachmentsFile(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Conf)
}

var closeListKeywordsInput = (*os.File).Close

// ListKeywordsFile returns the keyword list of inFile.
func ListKeywordsFile(inFile string, conf *model.Configuration) ([]string, error) {
	if inFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list keywords: open input %s: %w", inFile, err)
	}

	keywords, opErr := api.Keywords(f, conf)
	closeErr := closeListKeywordsInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list keywords: close input %s: %w", inFile, closeErr)
	}
	return keywords, errors.Join(opErr, closeErr)
}

// ListKeywords returns a list of keywords for inFile.
func ListKeywords(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list keywords")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker("list keywords", func(rs io.ReadSeeker) ([]string, error) {
			return api.Keywords(rs, cmd.Conf)
		})
	}

	return ListKeywordsFile(inFile, cmd.Conf)
}

func runKeywordStreamOperation(inFile, outFile, op string, fn func(io.ReadSeeker, io.Writer) error) error {
	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, op)
	if err != nil {
		return err
	}
	return finalize(fn(rs, w))
}

func validateKeywordValues(keywords []string, op string) error {
	for _, keyword := range keywords {
		if strings.TrimSpace(keyword) == "" {
			return fmt.Errorf("%s: validate keywords: keyword must not be empty", op)
		}
	}
	return nil
}

// AddKeywords adds keywords to inFile's document info dict and writes the result to outFile.
func AddKeywords(cmd *Command) ([]string, error) {
	requirements := commandRequirements{
		operation: "add keywords",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	if err := validateKeywordValues(cmd.StringVals, "add keywords"); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	err := runKeywordStreamOperation(*cmd.InFile, *cmd.OutFile, "add keywords", func(rs io.ReadSeeker, w io.Writer) error {
		return api.AddKeywords(rs, w, cmd.StringVals, cmd.Conf)
	})
	return nil, err
}

// RemoveKeywords deletes keywords from inFile's document info dict and writes the result to outFile.
func RemoveKeywords(cmd *Command) ([]string, error) {
	requirements := commandRequirements{
		operation: "remove keywords",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	if err := validateKeywordValues(cmd.StringVals, "remove keywords"); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	err := runKeywordStreamOperation(*cmd.InFile, *cmd.OutFile, "remove keywords", func(rs io.ReadSeeker, w io.Writer) error {
		return api.RemoveKeywords(rs, w, cmd.StringVals, cmd.Conf)
	})
	return nil, err
}

func renderProperties(properties map[string]string) []string {
	ss := make([]string, 0, len(properties))
	for k, v := range properties {
		ss = append(ss, fmt.Sprintf("%s = %s", k, v))
	}
	sort.Strings(ss)
	return ss
}

func listProperties(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	properties, err := api.Properties(rs, conf)
	if err != nil {
		return nil, err
	}
	return renderProperties(properties), nil
}

var closeListPropertiesInput = (*os.File).Close

// ListPropertiesFile returns the property list of inFile.
func ListPropertiesFile(inFile string, conf *model.Configuration) ([]string, error) {
	if inFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list properties: open input %s: %w", inFile, err)
	}

	properties, opErr := listProperties(f, conf)
	closeErr := closeListPropertiesInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list properties: close input %s: %w", inFile, closeErr)
	}
	return properties, errors.Join(opErr, closeErr)
}

// ListProperties returns inFile's properties.
func ListProperties(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list properties")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker("list properties", func(rs io.ReadSeeker) ([]string, error) {
			return listProperties(rs, cmd.Conf)
		})
	}

	return ListPropertiesFile(inFile, cmd.Conf)
}

func validatePropertyMap(properties map[string]string) error {
	for k, v := range properties {
		if strings.TrimSpace(k) == "" {
			return errors.New("property name must not be empty")
		}
		if !validate.DocumentProperty(k) {
			return fmt.Errorf("property name %q not allowed", k)
		}
		if strings.TrimSpace(v) == "" {
			return errors.New("property value must not be empty")
		}
	}
	return nil
}

func validatePropertyNames(properties []string) error {
	for _, property := range properties {
		if strings.TrimSpace(property) == "" {
			return errors.New("property name must not be empty")
		}
		if !validate.DocumentProperty(property) {
			return fmt.Errorf("property name %q not allowed", property)
		}
	}
	return nil
}

func runPropertyStreamOperation(inFile, outFile, op string, fn func(io.ReadSeeker, io.Writer) error) error {
	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, op)
	if err != nil {
		return err
	}
	return finalize(fn(rs, w))
}

func validatePropertyCommand(cmd *Command, operation string) error {
	requirements := commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if *cmd.InFile == "-" && *cmd.OutFile == "" {
		return commandValidationError(operation, api.ErrMissingPDFOutput)
	}
	return nil
}

// AddProperties adds properties to inFile's document info dict and writes the result to outFile.
func AddProperties(cmd *Command) ([]string, error) {
	if err := validatePropertyCommand(cmd, "add properties"); err != nil {
		return nil, err
	}
	if err := validatePropertyMap(cmd.StringMap); err != nil {
		return nil, fmt.Errorf("add properties: validate properties: %w", err)
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddPropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringMap, cmd.Conf)
	}

	err := runPropertyStreamOperation(*cmd.InFile, *cmd.OutFile, "add properties", func(rs io.ReadSeeker, w io.Writer) error {
		return api.AddProperties(rs, w, cmd.StringMap, cmd.Conf)
	})
	return nil, err
}

// RemoveProperties deletes properties from inFile's document info dict and writes the result to outFile.
func RemoveProperties(cmd *Command) ([]string, error) {
	if err := validatePropertyCommand(cmd, "remove properties"); err != nil {
		return nil, err
	}
	if err := validatePropertyNames(cmd.StringVals); err != nil {
		return nil, fmt.Errorf("remove properties: validate properties: %w", err)
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemovePropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	err := runPropertyStreamOperation(*cmd.InFile, *cmd.OutFile, "remove properties", func(rs io.ReadSeeker, w io.Writer) error {
		return api.RemoveProperties(rs, w, cmd.StringVals, cmd.Conf)
	})
	return nil, err
}
