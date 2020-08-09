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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// EncryptFile encrypts inFile and writes the result to outFile.
// A configuration containing the current passwords is required.
func EncryptFile(inFile, outFile string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for encryption")
	}
	conf.Cmd = pdfcpu.ENCRYPT
	return OptimizeFile(inFile, outFile, conf)
}

// DecryptFile decrypts inFile and writes the result to outFile.
// A configuration containing the current passwords is required.
func DecryptFile(inFile, outFile string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for decryption")
	}
	conf.Cmd = pdfcpu.DECRYPT
	return OptimizeFile(inFile, outFile, conf)
}

// ChangeUserPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeUserPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change user password")
	}
	conf.Cmd = pdfcpu.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew
	return OptimizeFile(inFile, outFile, conf)
}

// ChangeOwnerPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeOwnerPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change owner password")
	}
	conf.Cmd = pdfcpu.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew
	return OptimizeFile(inFile, outFile, conf)
}
