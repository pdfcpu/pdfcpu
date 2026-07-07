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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Encrypt inFile and write result to outFile.
func Encrypt(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.EncryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	var rs io.ReadSeeker
	var err error
	if *cmd.InFile == "-" {
		rs, err = readSeekerFromStdin()
	} else {
		rs, err = os.Open(*cmd.InFile)
	}
	if err != nil {
		return nil, err
	}
	if f, ok := rs.(*os.File); ok {
		defer f.Close()
	}

	w := io.Writer(os.Stdout)
	if *cmd.OutFile == "-" {
		log.SetCLILogger(nil)
	} else {
		f, err := os.Create(*cmd.OutFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		w = f
	}

	return nil, api.Encrypt(rs, w, cmd.Conf)
}

// Decrypt inFile and write result to outFile.
func Decrypt(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.DecryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	var rs io.ReadSeeker
	var err error
	if *cmd.InFile == "-" {
		rs, err = readSeekerFromStdin()
	} else {
		rs, err = os.Open(*cmd.InFile)
	}
	if err != nil {
		return nil, err
	}
	if f, ok := rs.(*os.File); ok {
		defer f.Close()
	}

	w := io.Writer(os.Stdout)
	if *cmd.OutFile == "-" {
		log.SetCLILogger(nil)
	} else {
		f, err := os.Create(*cmd.OutFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		w = f
	}

	return nil, api.Decrypt(rs, w, cmd.Conf)
}

// ChangeUserPassword of inFile and write result to outFile.
func ChangeUserPassword(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeUserPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.ChangeUserPassword(rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
}

// ChangeOwnerPassword of inFile and write result to outFile.
func ChangeOwnerPassword(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ChangeOwnerPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.ChangeOwnerPassword(rs, w, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
}

func listPermissions(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listPermissions: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPERMISSIONS

	ctx, err := api.ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	return pdfcpu.Permissions(ctx), nil
}

// ListPermissionsFile returns a list of user access permissions for inFile.
func ListPermissionsFile(inFiles []string, conf *model.Configuration) ([]string, error) {
	log.SetCLILogger(nil)

	var ss []string
	var errs []error

	for i, fn := range inFiles {
		if i > 0 {
			ss = append(ss, "")
		}
		f, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		defer func() {
			f.Close()
		}()
		ssx, err := listPermissions(f, conf)
		if err != nil {
			if len(inFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}
		ss = append(ss, fn+":")
		ss = append(ss, ssx...)
	}

	return ss, errors.Join(errs...)
}

// ListPermissions of inFile.
func ListPermissions(cmd *Command) ([]string, error) {
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

		var rs io.ReadSeeker
		var err error
		if fn == "-" {
			rs, err = readSeekerFromStdin()
		} else {
			rs, err = os.Open(fn)
		}
		if err != nil {
			return nil, err
		}
		if f, ok := rs.(*os.File); ok {
			defer f.Close()
		}

		ssx, err := listPermissions(rs, cmd.Conf)
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}
		label := fn
		if label == "-" {
			label = "stdin"
		}
		ss = append(ss, label+":")
		ss = append(ss, ssx...)
	}

	return ss, errors.Join(errs...)
}

// SetPermissions of inFile.
func SetPermissions(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.SetPermissionsFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.SetPermissions(rs, w, cmd.Conf)
}
