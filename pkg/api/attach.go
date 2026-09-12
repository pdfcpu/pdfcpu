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
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
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

// Attachments returns rs's attachments and supports cancellation.
func Attachments(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (aa []model.Attachment, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTATTACHMENTS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}

	aa, err = ctx.ListAttachments(c)
	if err != nil {
		return nil, fmt.Errorf("list attachments: collect attachments: %w", err)
	}
	return aa, nil
}

func addAttachment(c context.Context, ctx *model.Context, spec string, coll bool, op string) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	parts := strings.SplitN(spec, ",", 2)
	fileName := parts[0]
	desc := ""
	if len(parts) == 2 {
		desc = parts[1]
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
	if err := ctx.AddAttachment(c, a, coll); err != nil {
		return fmt.Errorf("%s: embed attachment %s: %w", op, fileName, err)
	}
	return nil
}

// AddAttachments embeds files into a PDF context read from rs, writes the result to w and supports cancellation.
// Each file is either a filename or a filename and description separated by a comma.
func AddAttachments(c context.Context, rs io.ReadSeeker, w io.Writer, files []string, coll bool, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	conf = operationConfiguration(conf, addAttachmentsCommandMode(coll))

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for _, spec := range files {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := addAttachment(c, ctx, spec, coll, op); err != nil {
			return err
		}
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("%s: write output: %w", op, err)
	}
	return nil
}

// AddAttachmentsFile embeds files into a PDF context read from inFile,
// writes the result to outFile and supports cancellation.
func AddAttachmentsFile(c context.Context, inFile, outFile string, files []string, coll bool, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = AddAttachments(c, f1, f2, files, coll, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true
	return nil
}

// RemoveAttachments deletes embedded files from a PDF context read from rs,
// writes the result to w and supports cancellation.
func RemoveAttachments(c context.Context, rs io.ReadSeeker, w io.Writer, files []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(files, "attachment filename"); err != nil {
		return fmt.Errorf("remove attachments: validate attachment filenames: %w", err)
	}

	conf = operationConfiguration(conf, model.REMOVEATTACHMENTS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove attachments: %w", err)
	}

	var ok bool
	if ok, err = ctx.RemoveAttachments(c, files); err != nil {
		return fmt.Errorf("remove attachments: remove: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove attachments: %w", ErrNoAttachmentRemoved)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove attachments: write output: %w", err)
	}
	return nil
}

// RemoveAttachmentsFile deletes embedded files from a PDF context read from inFile,
// writes the result to outFile and supports cancellation.
func RemoveAttachmentsFile(c context.Context, inFile, outFile string, files []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if err := validateNoEmptyStrings(files, "attachment filename"); err != nil {
		return fmt.Errorf("remove attachments: validate attachment filenames: %w", err)
	}

	op := "remove attachments"
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

	if err = RemoveAttachments(c, f1, f2, files, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true
	return nil
}

// ExtractAttachmentsRaw extracts embedded files from a PDF context read from rs and supports cancellation.
// outDir is retained for API compatibility and is otherwise ignored.
func ExtractAttachmentsRaw(c context.Context, rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) (aa []model.Attachment, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if err := validateNoEmptyStrings(fileNames, "attachment filename"); err != nil {
		return nil, fmt.Errorf("extract attachments: validate attachment filenames: %w", err)
	}

	conf = operationConfiguration(conf, model.EXTRACTATTACHMENTS)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, fmt.Errorf("extract attachments: %w", err)
	}

	aa, err = ctx.ExtractAttachments(c, fileNames)
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

func writeAttachmentToPath(c context.Context, fileName string, a model.Attachment) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	staged, err := openStagedOutput(nil, "", fileName, "extract attachments")
	if err != nil {
		return fmt.Errorf("extract attachments: create output %s: %w", fileName, err)
	}
	f := staged.output.file

	copyErr := copyStream(c, f, a)
	if copyErr != nil {
		return staged.cleanup(
			fmt.Errorf("extract attachments: write output %s: %w", fileName, copyErr),
		)
	}
	if err := contextutil.Check(c); err != nil {
		return staged.cleanup(err)
	}
	return staged.commit()
}

func writeAttachment(c context.Context, outDir string, i int, a model.Attachment) error {
	return writeAttachmentToPath(c, attachmentOutputPath(outDir, i, a), a)
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

func reserveAttachmentOutputs(c context.Context, paths []string, aa []model.Attachment) ([]attachmentOutputReservation, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	token, err := attachmentReservationToken()
	if err != nil {
		return nil, err
	}
	rr := make([]attachmentOutputReservation, 0, len(paths))
	for i, path := range paths {
		if err := contextutil.Check(c); err != nil {
			return rr, err
		}
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

func writeAttachments(c context.Context, outDir string, aa []model.Attachment) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	paths := attachmentOutputPaths(outDir, aa)
	rr, err := reserveAttachmentOutputs(c, paths, aa)
	if err != nil {
		return errors.Join(err, releaseAttachmentOutputReservations(rr))
	}
	defer func() {
		err = errors.Join(err, releaseAttachmentOutputReservations(rr))
	}()
	for i, a := range aa {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := writeAttachmentToPath(c, paths[i], a); err != nil {
			return err
		}
	}
	return nil
}

// ExtractAttachments extracts embedded files from a PDF context read from rs into outDir and supports cancellation.
func ExtractAttachments(c context.Context, rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if outDir == "" {
		return ErrMissingPDFOutput
	}

	aa, err := ExtractAttachmentsRaw(c, rs, outDir, fileNames, conf)
	if err != nil {
		return err
	}

	return writeAttachments(c, outDir, aa)
}

// ExtractAttachmentsFile extracts embedded files from a PDF context read from inFile into outDir
// and supports cancellation.
func ExtractAttachmentsFile(c context.Context, inFile, outDir string, files []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	return ExtractAttachments(c, f, outDir, files, conf)
}
