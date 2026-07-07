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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateCryptoCommand(cmd *Command) error {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return api.ErrMissingPDFOutput
	}
	if cmd.Conf == nil {
		return api.ErrMissingConfiguration
	}
	return nil
}

func validatePasswordChangeCommand(cmd *Command) error {
	if err := validateCryptoCommand(cmd); err != nil {
		return err
	}
	if cmd.PWOld == nil {
		return errors.New("missing old password")
	}
	if cmd.PWNew == nil {
		return errors.New("missing new password")
	}
	return nil
}

func validatePermissionInputs(inFiles []string, conf *model.Configuration) error {
	if len(inFiles) == 0 {
		return api.ErrMissingPDFInput
	}
	for i, inFile := range inFiles {
		if inFile == "" {
			return fmt.Errorf("input %d: %w", i+1, api.ErrMissingPDFInput)
		}
	}
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	return nil
}

// Encrypt inFile and write result to outFile.
func Encrypt(cmd *Command) ([]string, error) {
	if err := validateCryptoCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.EncryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}
	return nil, runContentStreamOperation(*cmd.InFile, *cmd.OutFile, "encrypt", func(rs io.ReadSeeker, w io.Writer) error {
		return api.Encrypt(rs, w, cmd.Conf)
	})
}

// Decrypt inFile and write result to outFile.
func Decrypt(cmd *Command) ([]string, error) {
	if err := validateCryptoCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.DecryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	return nil, runContentStreamOperation(*cmd.InFile, *cmd.OutFile, "decrypt", func(rs io.ReadSeeker, w io.Writer) error {
		return api.Decrypt(rs, w, cmd.Conf)
	})
}

// ChangeUserPassword of inFile and write result to outFile.
func ChangeUserPassword(cmd *Command) ([]string, error) {
	if err := validatePasswordChangeCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeUserPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	}

	return nil, runContentStreamOperation(
		*cmd.InFile,
		*cmd.OutFile,
		"change user password",
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.ChangeUserPassword(rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
		},
	)
}

// ChangeOwnerPassword of inFile and write result to outFile.
func ChangeOwnerPassword(cmd *Command) ([]string, error) {
	if err := validatePasswordChangeCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeOwnerPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	}

	return nil, runContentStreamOperation(
		*cmd.InFile,
		*cmd.OutFile,
		"change owner password",
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.ChangeOwnerPassword(rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
		},
	)
}

func listPermissions(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	return api.PermissionsList(rs, conf)
}

var closeListPermissionsInput = (*os.File).Close

func readPermissionsFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list permissions: open input: %w", err)
	}

	permissions, opErr := listPermissions(f, conf)
	closeErr := closeListPermissionsInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list permissions: close input: %w", closeErr)
	}
	return permissions, errors.Join(opErr, closeErr)
}

// ListPermissionsFile returns a list of user access permissions for inFile.
func ListPermissionsFile(inFiles []string, conf *model.Configuration) ([]string, error) {
	if err := validatePermissionInputs(inFiles, conf); err != nil {
		return nil, err
	}
	log.SetCLILogger(nil)

	var ss []string
	var errs []error

	for i, fn := range inFiles {
		if i > 0 {
			ss = append(ss, "")
		}
		ssx, err := readPermissionsFile(fn, conf)
		if err != nil {
			err = fmt.Errorf("%s: %w", fn, err)
			if len(inFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
			continue
		}
		ss = append(ss, fn+":")
		ss = append(ss, ssx...)
	}

	return ss, errors.Join(errs...)
}

// ListPermissions of inFile.
func ListPermissions(cmd *Command) ([]string, error) {
	if cmd == nil {
		return nil, api.ErrMissingPDFInput
	}
	if err := validatePermissionInputs(cmd.InFiles, cmd.Conf); err != nil {
		return nil, err
	}

	stdin := false
	for _, fn := range cmd.InFiles {
		if fn == "-" {
			stdin = true
			break
		}
	}
	if !stdin {
		return ListPermissionsFile(cmd.InFiles, cmd.Conf)
	}

	log.SetCLILogger(nil)
	var ss []string
	var errs []error
	for i, fn := range cmd.InFiles {
		if i > 0 {
			ss = append(ss, "")
		}

		var ssx []string
		var err error
		label := fn
		if fn == "-" {
			label = "stdin"
			ssx, err = withStdinReadSeeker("list permissions", func(rs io.ReadSeeker) ([]string, error) {
				return listPermissions(rs, cmd.Conf)
			})
		} else {
			ssx, err = readPermissionsFile(fn, cmd.Conf)
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", label, err)
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
			continue
		}
		ss = append(ss, label+":")
		ss = append(ss, ssx...)
	}

	return ss, errors.Join(errs...)
}

// SetPermissions of inFile.
func SetPermissions(cmd *Command) ([]string, error) {
	if err := validateCryptoCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.SetPermissionsFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	return nil, runContentStreamOperation(
		*cmd.InFile,
		*cmd.OutFile,
		"set permissions",
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.SetPermissions(rs, w, cmd.Conf)
		},
	)
}
