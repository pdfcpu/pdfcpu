/*
	Copyright 2019 The pdfcpu Authors.

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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
)

func addAttachmentsOperation(coll bool) string {
	if coll {
		return "add portfolio attachments"
	}
	return "add attachments"
}

func addAttachmentsCommandMode(coll bool) model.CommandMode {
	if coll {
		return model.ADDATTACHMENTSPORTFOLIO
	}
	return model.ADDATTACHMENTS
}

// Attachments returns rs's attachments.
func Attachments(rs io.ReadSeeker, conf *model.Configuration) (aa []model.Attachment, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTATTACHMENTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}

	aa, err = ctx.ListAttachments()
	if err != nil {
		return nil, fmt.Errorf("list attachments: collect attachments: %w", err)
	}
	return aa, nil
}

func addAttachment(ctx *model.Context, spec string, coll bool, op string) (err error) {
	parts := strings.SplitN(spec, ",", 2)
	fileName := parts[0]
	desc := ""
	if len(parts) == 2 {
		desc = parts[1]
	}

	if log.CLIEnabled() {
		log.CLI.Printf("adding %s\n", fileName)
	}
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("%s: open attachment %s: %w", op, fileName, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, fmt.Sprintf("%s: close attachment %s", op, fileName)))
	}()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("%s: stat attachment %s: %w", op, fileName, err)
	}
	mt := fi.ModTime()

	a := model.Attachment{Reader: f, ID: filepath.Base(fileName), Desc: desc, ModTime: &mt}
	if err := ctx.AddAttachment(a, coll); err != nil {
		return fmt.Errorf("%s: embed attachment %s: %w", op, fileName, err)
	}
	return nil
}

// AddAttachments embeds files into a PDF context read from rs and writes the result to w.
// Each file is either a file name or a file name and a description separated by a comma.
func AddAttachments(rs io.ReadSeeker, w io.Writer, files []string, coll bool, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	op := addAttachmentsOperation(coll)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if len(files) == 0 {
		return fmt.Errorf("%s: %w", op, ErrNoAttachmentAdded)
	}

	if err := validateAttachmentFileNames(files); err != nil {
		return fmt.Errorf("%s: validate attachment filenames: %w", op, err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = addAttachmentsCommandMode(coll)

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for _, spec := range files {
		if err := addAttachment(ctx, spec, coll, op); err != nil {
			return err
		}
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("%s: write output: %w", op, err)
	}
	return nil
}

type attachmentMutation func(io.ReadSeeker, io.Writer) error

func mutateAttachmentsFile(inFile, outFile, op string, mutate attachmentMutation) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, op)
	if err != nil {
		return errors.Join(
			fmt.Errorf("%s: create output: %w", op, err),
			closeFile(f1, op+": close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = mutate(f1, f2); err != nil {
		return err
	}

	ok = true
	return nil
}

// AddAttachmentsFile embeds files into a PDF context read from inFile and writes the result to outFile.
func AddAttachmentsFile(inFile, outFile string, files []string, coll bool, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	op := addAttachmentsOperation(coll)
	if len(files) == 0 {
		return fmt.Errorf("%s: %w", op, ErrNoAttachmentAdded)
	}
	if err := validateAttachmentFileNames(files); err != nil {
		return fmt.Errorf("%s: validate attachment filenames: %w", op, err)
	}
	return mutateAttachmentsFile(inFile, outFile, op, func(rs io.ReadSeeker, w io.Writer) error {
		return AddAttachments(rs, w, files, coll, conf)
	})
}

// RemoveAttachments deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveAttachments(rs io.ReadSeeker, w io.Writer, files []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(files, "attachment filename"); err != nil {
		return fmt.Errorf("remove attachments: validate attachment filenames: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEATTACHMENTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove attachments: %w", err)
	}

	var ok bool
	if ok, err = ctx.RemoveAttachments(files); err != nil {
		return fmt.Errorf("remove attachments: remove: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove attachments: %w", ErrNoAttachmentRemoved)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove attachments: write output: %w", err)
	}
	return nil
}

// RemoveAttachmentsFile deletes embedded files from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachmentsFile(inFile, outFile string, files []string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if err := validateNoEmptyStrings(files, "attachment filename"); err != nil {
		return fmt.Errorf("remove attachments: validate attachment filenames: %w", err)
	}

	return mutateAttachmentsFile(inFile, outFile, "remove attachments", func(rs io.ReadSeeker, w io.Writer) error {
		return RemoveAttachments(rs, w, files, conf)
	})
}

// ExtractAttachmentsRaw extracts embedded files from a PDF context read from rs.
// outDir is retained for API compatibility and is otherwise ignored.
func ExtractAttachmentsRaw(rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) (aa []model.Attachment, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if err := validateNoEmptyStrings(fileNames, "attachment filename"); err != nil {
		return nil, fmt.Errorf("extract attachments: validate attachment filenames: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTATTACHMENTS

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("extract attachments: %w", err)
	}

	aa, err = ctx.ExtractAttachments(fileNames)
	if err != nil {
		return nil, fmt.Errorf("extract attachments: collect attachments: %w", err)
	}
	return aa, nil
}

func attachmentOutputPath(outDir string, i int, a model.Attachment) string {
	fn, err := sanitize.Path(a.FileName)
	if err != nil {
		fn = fmt.Sprintf("attachment_%d", i+1)
	}
	return filepath.Clean(filepath.Join(outDir, fn))
}

func writeAttachmentToPath(fileName string, a model.Attachment) error {
	staged, err := openStagedOutput(nil, "", fileName, "extract attachments")
	if err != nil {
		return fmt.Errorf("extract attachments: create output %s: %w", fileName, err)
	}
	f := staged.output.file
	logWritingTo(fileName)

	_, copyErr := io.Copy(f, a)
	if copyErr != nil {
		return staged.cleanup(
			fmt.Errorf("extract attachments: write output %s: %w", fileName, copyErr),
		)
	}
	return staged.commit()
}

func writeAttachment(outDir string, i int, a model.Attachment) error {
	return writeAttachmentToPath(attachmentOutputPath(outDir, i, a), a)
}

func attachmentOutputPaths(outDir string, aa []model.Attachment) []string {
	paths := make([]string, len(aa))
	for i, a := range aa {
		paths[i] = attachmentOutputPath(outDir, i, a)
	}
	return paths
}

type attachmentOutputReservation struct {
	file *os.File
	path string
	id   string
}

func attachmentReservationToken() (string, error) {
	bb := make([]byte, 8)
	if _, err := rand.Read(bb); err != nil {
		return "", fmt.Errorf("extract attachments: create output reservation token: %w", err)
	}
	return hex.EncodeToString(bb), nil
}

func attachmentReservationPath(fileName, token string) string {
	name := "." + filepath.Base(fileName) + ".pdfcpu-reservation-" + token
	return filepath.Join(filepath.Dir(fileName), name)
}

func attachmentReservationConflictID(path string, rr []attachmentOutputReservation) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	for _, r := range rr {
		rfi, err := r.file.Stat()
		if err != nil {
			return "", err
		}
		if os.SameFile(fi, rfi) {
			return r.id, nil
		}
	}
	return "", errors.New("conflicting output reservation not found")
}

func reserveAttachmentOutputs(
	paths []string,
	aa []model.Attachment,
) ([]attachmentOutputReservation, error) {
	token, err := attachmentReservationToken()
	if err != nil {
		return nil, err
	}
	rr := make([]attachmentOutputReservation, 0, len(paths))
	for i, path := range paths {
		reservationPath := attachmentReservationPath(path, token)
		f, err := os.OpenFile(reservationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			id, lookupErr := attachmentReservationConflictID(reservationPath, rr)
			collisionErr := fmt.Errorf(
				"extract attachments: output %s for attachment %q conflicts with attachment %q: %w",
				path,
				aa[i].ID,
				id,
				ErrAttachmentOutputCollision,
			)
			return rr, errors.Join(collisionErr, lookupErr)
		}
		if err != nil {
			return rr, fmt.Errorf(
				"extract attachments: reserve output %s for attachment %q: %w",
				path,
				aa[i].ID,
				err,
			)
		}
		rr = append(rr, attachmentOutputReservation{file: f, path: reservationPath, id: aa[i].ID})
	}
	return rr, nil
}

func releaseAttachmentOutputReservations(rr []attachmentOutputReservation) error {
	var err error
	for _, r := range rr {
		err = errors.Join(
			err,
			closeFile(r.file, "extract attachments: close output reservation "+r.path),
			removeFile(r.path, "extract attachments: remove output reservation "+r.path),
		)
	}
	return err
}

func writeAttachments(outDir string, aa []model.Attachment) (err error) {
	paths := attachmentOutputPaths(outDir, aa)
	rr, err := reserveAttachmentOutputs(paths, aa)
	if err != nil {
		return errors.Join(err, releaseAttachmentOutputReservations(rr))
	}
	defer func() {
		err = errors.Join(err, releaseAttachmentOutputReservations(rr))
	}()
	for i, a := range aa {
		if err := writeAttachmentToPath(paths[i], a); err != nil {
			return err
		}
	}
	return nil
}

// ExtractAttachments extracts embedded files from a PDF context read from rs into outDir.
func ExtractAttachments(rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if outDir == "" {
		return ErrMissingPDFOutput
	}

	aa, err := ExtractAttachmentsRaw(rs, outDir, fileNames, conf)
	if err != nil {
		return err
	}

	return writeAttachments(outDir, aa)
}

// ExtractAttachmentsFile extracts embedded files from a PDF context read from inFile into outDir.
func ExtractAttachmentsFile(inFile, outDir string, files []string, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if outDir == "" {
		return ErrMissingPDFOutput
	}

	if err := validateNoEmptyStrings(files, "attachment filename"); err != nil {
		return fmt.Errorf("extract attachments: validate attachment filenames: %w", err)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract attachments: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "extract attachments: close input"))
	}()

	return ExtractAttachments(f, outDir, files, conf)
}
