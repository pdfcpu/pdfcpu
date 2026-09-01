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

import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

// EncryptCommand creates a new command to encrypt a file.
func EncryptCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.ENCRYPT
	}
	return &Command{
		Mode:    model.ENCRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// DecryptCommand creates a new command to decrypt a file.
func DecryptCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.DECRYPT
	}
	return &Command{
		Mode:    model.DECRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ChangeUserPWCommand creates a new command to change the user password.
func ChangeUserPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.CHANGEUPW
	}
	return &Command{
		Mode:    model.CHANGEUPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ChangeOwnerPWCommand creates a new command to change the owner password.
func ChangeOwnerPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.CHANGEOPW
	}
	return &Command{
		Mode:    model.CHANGEOPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ListPermissionsCommand creates a new command to list permissions.
func ListPermissionsCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTPERMISSIONS
	}
	return &Command{
		Mode:    model.LISTPERMISSIONS,
		InFiles: inFiles,
		Conf:    conf}
}

// SetPermissionsCommand creates a new command to add permissions.
func SetPermissionsCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SETPERMISSIONS
	}
	return &Command{
		Mode:    model.SETPERMISSIONS,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}
