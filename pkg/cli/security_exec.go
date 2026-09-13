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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateCryptoCommand(cmd *Command, operation string) error {
	requirements := commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if cmd.Conf == nil {
		return commandValidationError(operation, api.ErrMissingConfiguration)
	}
	return nil
}

func validatePasswordChangeCommand(cmd *Command, operation string) error {
	if err := validateCryptoCommand(cmd, operation); err != nil {
		return err
	}
	if cmd.PWOld == nil {
		return commandValidationError(operation, errors.New("missing old password"))
	}
	if cmd.PWNew == nil {
		return commandValidationError(operation, errors.New("missing new password"))
	}
	return nil
}

func validatePermissionInputs(inFiles []string, conf *model.Configuration) error {
	if err := validateCommandInputFiles(inFiles, 1, 0, nil); err != nil {
		return commandValidationError("list permissions", err)
	}
	if conf == nil {
		return commandValidationError("list permissions", api.ErrMissingConfiguration)
	}
	return nil
}

func encrypt(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCryptoCommand(cmd, "encrypt"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.EncryptFile(c, *cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "encrypt")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Encrypt(c, rs, w, cmd.Conf))
}

func decrypt(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCryptoCommand(cmd, "decrypt"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.DecryptFile(c, *cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "decrypt")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Decrypt(c, rs, w, cmd.Conf))
}

func changeUserPassword(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePasswordChangeCommand(cmd, "change user password"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeUserPasswordFile(
			c, *cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(
		c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "change user password",
	)
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.ChangeUserPassword(c, rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf))
}

func changeOwnerPassword(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePasswordChangeCommand(cmd, "change owner password"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeOwnerPasswordFile(
			c, *cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(
		c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "change owner password",
	)
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.ChangeOwnerPassword(c, rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf))
}

func listPermissionsForReader(c context.Context, rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	return api.PermissionsList(c, rs, conf)
}

var closeListPermissionsInput = (*os.File).Close

func readPermissionsFile(c context.Context, inFile string, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list permissions: open input: %w", err)
	}

	permissions, opErr := listPermissionsForReader(c, f, conf)
	closeErr := closeListPermissionsInput(f)
	if closeErr != nil {
		closeErr = fmt.Errorf("list permissions: close input: %w", closeErr)
	}
	return permissions, errors.Join(opErr, closeErr)
}

// ListPermissionsFile returns user access permissions for inFiles and supports cancellation.
func ListPermissionsFile(c context.Context, inFiles []string, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePermissionInputs(inFiles, conf); err != nil {
		return nil, err
	}
	log.SetCLILogger(nil)

	var ss []string
	var errs []error

	for i, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			ss = append(ss, "")
		}
		ssx, err := readPermissionsFile(c, fn, conf)
		if err != nil {
			if contextErr := c.Err(); contextErr != nil {
				return nil, contextErr
			}
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

	if err := c.Err(); err != nil {
		return nil, err
	}
	return ss, errors.Join(errs...)
}

func permissionInputsContainStdin(c context.Context, inFiles []string) (bool, error) {
	for _, fn := range inFiles {
		if err := c.Err(); err != nil {
			return false, err
		}
		if fn == "-" {
			return true, nil
		}
	}
	return false, nil
}

func readPermissionCommandInput(c context.Context, fileName string, conf *model.Configuration) (string, []string, error) {
	if fileName != "-" {
		ss, err := readPermissionsFile(c, fileName, conf)
		return fileName, ss, err
	}
	ss, err := withStdinReadSeeker(c, conf, "list permissions", func(rs io.ReadSeeker) ([]string, error) {
		return listPermissionsForReader(c, rs, conf)
	})
	return "stdin", ss, err
}

func listPermissionCommandInputs(c context.Context, inFiles []string, conf *model.Configuration) ([]string, error) {
	var ss []string
	var errs []error
	for i, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			ss = append(ss, "")
		}
		label, ssx, err := readPermissionCommandInput(c, fn, conf)
		if err != nil {
			if contextErr := c.Err(); contextErr != nil {
				return nil, contextErr
			}
			err = fmt.Errorf("%s: %w", label, err)
			if len(inFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
			continue
		}
		ss = append(ss, label+":")
		ss = append(ss, ssx...)
	}
	if err := c.Err(); err != nil {
		return nil, err
	}
	return ss, errors.Join(errs...)
}

func listPermissionsCommand(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{operation: "list permissions"}); err != nil {
		return nil, err
	}
	if err := validatePermissionInputs(cmd.InFiles, cmd.Conf); err != nil {
		return nil, err
	}

	stdin, err := permissionInputsContainStdin(c, cmd.InFiles)
	if err != nil {
		return nil, err
	}
	if !stdin {
		return ListPermissionsFile(c, cmd.InFiles, cmd.Conf)
	}

	log.SetCLILogger(nil)
	return listPermissionCommandInputs(c, cmd.InFiles, cmd.Conf)
}

func setPermissions(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCryptoCommand(cmd, "set permissions"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.SetPermissionsFile(c, *cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "set permissions")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.SetPermissions(c, rs, w, cmd.Conf))
}
