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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Encrypt reads a PDF stream from rs and writes the encrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Encrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
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

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return Encrypt(f1, f2, conf)
}

// Decrypt reads a PDF stream from rs and writes the encrypted PDF stream to w.
// A configuration containing at least the current passwords is required.
func Decrypt(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
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

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return Decrypt(f1, f2, conf)
}

// ChangeUserPassword reads a PDF stream from rs, changes the user password and writes the encrypted PDF stream to w.
// A configuration containing the current passwords is required.
func ChangeUserPassword(rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) error {
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

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return ChangeUserPassword(f1, f2, pwOld, pwNew, conf)
}

// ChangeOwnerPassword reads a PDF stream from rs, changes the owner password and writes the encrypted PDF stream to w.
// A configuration containing the current passwords is required.
func ChangeOwnerPassword(rs io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *model.Configuration) error {
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

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return ChangeOwnerPassword(f1, f2, pwOld, pwNew, conf)
}
