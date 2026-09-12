/*
	Copyright 2020 The pdfcpu Authors.

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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Encrypt reads a PDF stream from rs, writes the encrypted PDF stream to w and supports cancellation.
// A configuration containing at least the current passwords is required.
func Encrypt(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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

	if conf == nil {
		return ErrMissingConfiguration
	}
	conf = operationConfiguration(conf, model.ENCRYPT)

	if err := optimize(c, rs, w, conf, ProgressOptions{}); err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	return nil
}

// EncryptFile encrypts inFile, writes the result to outFile and supports cancellation.
// A configuration containing at least the current passwords is required.
func EncryptFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("encrypt: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "encrypt")
	if err != nil {
		return errors.Join(fmt.Errorf("encrypt: create output: %w", err), closeFile(f1, "encrypt: close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Encrypt(c, f1, staged.output.file, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// Decrypt reads an encrypted PDF stream from rs, writes the decrypted PDF stream to w and supports cancellation.
// A configuration containing at least the current passwords is required.
func Decrypt(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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

	if conf == nil {
		return ErrMissingConfiguration
	}
	conf = operationConfiguration(conf, model.DECRYPT)

	if err := optimize(c, rs, w, conf, ProgressOptions{}); err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}
	return nil
}

// DecryptFile decrypts inFile, writes the result to outFile and supports cancellation.
// A configuration containing at least the current passwords is required.
func DecryptFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("decrypt: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "decrypt")
	if err != nil {
		return errors.Join(fmt.Errorf("decrypt: create output: %w", err), closeFile(f1, "decrypt: close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Decrypt(c, f1, staged.output.file, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// ChangeUserPassword reads a PDF stream from rs, changes the user password,
// writes the encrypted PDF stream to w and supports cancellation.
// A configuration containing the current passwords is required.
func ChangeUserPassword(c context.Context, rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) (err error) {
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

	if conf == nil {
		return ErrMissingConfiguration
	}

	conf = operationConfiguration(conf, model.CHANGEUPW)
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew

	if err := optimize(c, rs, w, conf, ProgressOptions{}); err != nil {
		return fmt.Errorf("change user password: %w", err)
	}
	return nil
}

// ChangeUserPasswordFile reads inFile, changes the user password,
// writes the result to outFile and supports cancellation.
// A configuration containing the current passwords is required.
func ChangeUserPasswordFile(c context.Context, inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) (err error) {
	const op = "change user password"

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, op)
	if err != nil {
		return errors.Join(fmt.Errorf("%s: create output: %w", op, err), closeFile(f1, op+": close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ChangeUserPassword(c, f1, staged.output.file, pwOld, pwNew, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// ChangeOwnerPassword reads a PDF stream from rs, changes the owner password,
// writes the encrypted PDF stream to w and supports cancellation.
// A configuration containing the current passwords is required.
func ChangeOwnerPassword(c context.Context, rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) (err error) {
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

	if conf == nil {
		return ErrMissingConfiguration
	}
	if pwNew == "" {
		return fmt.Errorf("change owner password: new owner password must not be empty: %w", pdfcpu.ErrOwnerPasswordRequired)
	}

	conf = operationConfiguration(conf, model.CHANGEOPW)
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew

	if err := optimize(c, rs, w, conf, ProgressOptions{}); err != nil {
		return fmt.Errorf("change owner password: %w", err)
	}
	return nil
}

// ChangeOwnerPasswordFile reads inFile, changes the owner password,
// writes the result to outFile and supports cancellation.
// A configuration containing the current passwords is required.
func ChangeOwnerPasswordFile(c context.Context, inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) (err error) {
	const op = "change owner password"

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if pwNew == "" {
		return fmt.Errorf("%s: new owner password must not be empty: %w", op, pdfcpu.ErrOwnerPasswordRequired)
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, op)
	if err != nil {
		return errors.Join(fmt.Errorf("%s: create output: %w", op, err), closeFile(f1, op+": close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ChangeOwnerPassword(c, f1, staged.output.file, pwOld, pwNew, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}
