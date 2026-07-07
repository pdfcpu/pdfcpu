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

// Permissions returns user access permissions for rs.
func Permissions(rs io.ReadSeeker, conf *model.Configuration) (p int, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return 0, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPERMISSIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return 0, fmt.Errorf("list permissions: %w", err)
	}

	if ctx.E != nil {
		p = ctx.E.P
	}

	return p, nil
}

// PermissionsList returns formatted user access permissions for rs.
func PermissionsList(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	p, err := Permissions(rs, conf)
	if err != nil {
		return nil, err
	}
	return pdfcpu.PermissionsList(p), nil
}

// SetPermissions sets user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissions(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.SETPERMISSIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	if err := WriteContext(ctx, w); err != nil {
		return fmt.Errorf("set permissions: write output: %w", err)
	}
	return nil
}

// SetPermissionsFile sets inFile's user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissionsFile(inFile, outFile string, conf *model.Configuration) error {
	const op = "set permissions"

	if conf == nil {
		return ErrMissingConfiguration
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	return processSecurityFile(inFile, outFile, conf, op, SetPermissions)
}

// GetPermissions returns the permissions for rs.
func GetPermissions(rs io.ReadSeeker, conf *model.Configuration) (p *int16, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	// No cmd available.

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}

	if ctx.E == nil {
		// Full access - permissions don't apply.
		return nil, nil
	}
	permissions := int16(ctx.E.P)

	return &permissions, nil
}

// GetPermissionsFile returns the permissions for inFile.
func GetPermissionsFile(inFile string, conf *model.Configuration) (p *int16, err error) {
	const op = "get permissions"

	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("%s: close input: %w", op, closeErr))
		}
	}()

	return GetPermissions(f, conf)
}
