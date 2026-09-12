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

// Permissions returns user access permissions for rs and supports cancellation.
func Permissions(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (p int, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	if rs == nil {
		return 0, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTPERMISSIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return 0, fmt.Errorf("list permissions: %w", err)
	}

	if ctx.E != nil {
		p = ctx.E.P
	}

	return p, contextutil.Check(c)
}

// PermissionsList returns formatted user access permissions for rs and supports cancellation.
func PermissionsList(c context.Context, rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	p, err := Permissions(c, rs, conf)
	if err != nil {
		return nil, err
	}
	return pdfcpu.PermissionsList(p), contextutil.Check(c)
}

// SetPermissions sets user access permissions and supports cancellation.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissions(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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
	conf = operationConfiguration(conf, model.SETPERMISSIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	if err := WriteContext(c, ctx, w); err != nil {
		return fmt.Errorf("set permissions: write output: %w", err)
	}
	return nil
}

// SetPermissionsFile sets inFile's user access permissions and supports cancellation.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissionsFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	const op = "set permissions"

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
		return errors.Join(
			fmt.Errorf("%s: create output: %w", op, err),
			closeFile(f1, op+": close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = SetPermissions(c, f1, staged.output.file, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// GetPermissions returns the permissions for rs and supports cancellation.
func GetPermissions(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (p *int16, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	// No cmd available.

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}

	if ctx.E == nil {
		// Full access - permissions don't apply.
		return nil, nil
	}
	permissions := int16(ctx.E.P)

	return &permissions, contextutil.Check(c)
}

// GetPermissionsFile returns the permissions for inFile and supports cancellation.
func GetPermissionsFile(c context.Context, inFile string, conf *model.Configuration) (p *int16, err error) {
	const op = "get permissions"

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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

	return GetPermissions(c, f, conf)
}
