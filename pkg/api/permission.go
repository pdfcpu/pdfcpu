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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Permissions returns user access permissions for rs.
func Permissions(rs io.ReadSeeker, conf *model.Configuration) (int, error) {
	if rs == nil {
		return 0, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPERMISSIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return 0, err
	}

	p := 0
	if ctx.E != nil {
		p = ctx.E.P
	}

	return p, nil
}

// SetPermissions sets user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissions(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
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
		return err
	}

	return WriteContext(ctx, w)
}

// SetPermissionsFile sets inFile's user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissionsFile(inFile, outFile string, conf *model.Configuration) (err error) {
	if conf == nil {
		return ErrMissingConfiguration
	}

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
			os.Remove(tmpFile)
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

	if err = SetPermissions(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// GetPermissions returns the permissions for rs.
func GetPermissions(rs io.ReadSeeker, conf *model.Configuration) (*int16, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	// No cmd available.

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	if ctx.E == nil {
		// Full access - permissions don't apply.
		return nil, nil
	}
	p := int16(ctx.E.P)

	return &p, nil
}

// GetPermissionsFile returns the permissions for inFile.
func GetPermissionsFile(inFile string, conf *model.Configuration) (*int16, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return GetPermissions(f, conf)
}
