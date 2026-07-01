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
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Encrypt reads a PDF stream from rs and writes the encrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Encrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: Encrypt: missing rs")
	}

	if conf == nil {
		return errors.New("pdfcpu: missing configuration for encryption")
	}
	conf.Cmd = model.ENCRYPT

	return Optimize(rs, w, conf)
}

// EncryptFile encrypts inFile and writes the result to outFile.
// A configuration containing at least the current passwords is required.
func EncryptFile(inFile, outFile string, conf *model.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for encryption")
	}
	conf.Cmd = model.ENCRYPT

	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = Encrypt(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// Decrypt reads a PDF stream from rs and writes the encrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Decrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: Decrypt: missing rs")
	}

	if conf == nil {
		return errors.New("pdfcpu: missing configuration for decryption")
	}
	conf.Cmd = model.DECRYPT

	return Optimize(rs, w, conf)
}

// DecryptFile decrypts inFile and writes the result to outFile.
// A configuration containing at least the current passwords is required.
func DecryptFile(inFile, outFile string, conf *model.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for decryption")
	}
	conf.Cmd = model.DECRYPT

	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = Decrypt(f1, f2, conf); err != nil {
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
		return errors.New("pdfcpu: ChangeUserPassword: missing rs")
	}

	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change user password")
	}

	conf.Cmd = model.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew

	return Optimize(rs, w, conf)
}

// ChangeUserPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeUserPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change user password")
	}

	conf.Cmd = model.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew

	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = ChangeUserPassword(f1, f2, pwOld, pwNew, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ChangeOwnerPassword reads a PDF stream from rs, changes the owner password and writes the encrypted PDF stream to w.
// A configuration containing the current passwords is required.
func ChangeOwnerPassword(rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ChangeOwnerPassword: missing rs")
	}

	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change owner password")
	}

	conf.Cmd = model.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew

	return Optimize(rs, w, conf)
}

// ChangeOwnerPasswordFile reads inFile, changes the owner password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeOwnerPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *model.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change owner password")
	}
	conf.Cmd = model.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew

	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = ChangeOwnerPassword(f1, f2, pwOld, pwNew, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
