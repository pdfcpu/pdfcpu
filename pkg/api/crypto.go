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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Encrypt reads a PDF stream from rs and writes the encrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Encrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		return ErrMissingConfiguration
	}
	conf.Cmd = model.ENCRYPT

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	return nil
}

// EncryptFile encrypts inFile and writes the result to outFile.
// A configuration containing at least the current passwords is required.
func EncryptFile(inFile, outFile string, conf *model.Configuration) error {
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	conf.Cmd = model.ENCRYPT
	return processSecurityFile(inFile, outFile, conf, "encrypt", Encrypt)
}

// Decrypt reads an encrypted PDF stream from rs and writes the decrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Decrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		return ErrMissingConfiguration
	}
	conf.Cmd = model.DECRYPT

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}
	return nil
}

// DecryptFile decrypts inFile and writes the result to outFile.
// A configuration containing at least the current passwords is required.
func DecryptFile(inFile, outFile string, conf *model.Configuration) error {
	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	conf.Cmd = model.DECRYPT
	return processSecurityFile(inFile, outFile, conf, "decrypt", Decrypt)
}

type securityOperation func(io.ReadSeeker, io.Writer, *model.Configuration) error

func processSecurityFile(
	inFile, outFile string,
	conf *model.Configuration,
	op string,
	process securityOperation,
) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
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

	if err = process(f1, f2, conf); err != nil {
		return err
	}

	ok = true
	return nil
}

// ChangeUserPassword reads a PDF stream from rs, changes the user password and writes the encrypted PDF stream to w.
// A configuration containing the current passwords is required.
func ChangeUserPassword(rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		return ErrMissingConfiguration
	}

	conf.Cmd = model.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("change user password: %w", err)
	}
	return nil
}

// ChangeUserPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeUserPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) error {
	const op = "change user password"

	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	return processSecurityFile(inFile, outFile, conf, op, func(
		rs io.ReadSeeker,
		w io.Writer,
		conf *model.Configuration,
	) error {
		return ChangeUserPassword(rs, w, pwOld, pwNew, conf)
	})
}

// ChangeOwnerPassword reads a PDF stream from rs, changes the owner password and writes the encrypted PDF stream to w.
// A configuration containing the current passwords is required.
func ChangeOwnerPassword(rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

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

	conf.Cmd = model.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("change owner password: %w", err)
	}
	return nil
}

// ChangeOwnerPasswordFile reads inFile, changes the owner password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeOwnerPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) error {
	const op = "change owner password"

	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if pwNew == "" {
		return fmt.Errorf("%s: new owner password must not be empty: %w", op, pdfcpu.ErrOwnerPasswordRequired)
	}

	return processSecurityFile(inFile, outFile, conf, op, func(
		rs io.ReadSeeker,
		w io.Writer,
		conf *model.Configuration,
	) error {
		return ChangeOwnerPassword(rs, w, pwOld, pwNew, conf)
	})
}
