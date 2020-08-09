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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// ListPermissions returns a list of user access permissions.
func ListPermissions(rs io.ReadSeeker, conf *pdfcpu.Configuration) ([]string, error) {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTPERMISSIONS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return nil, err
	}

	fromList := time.Now()
	list := pdfcpu.Permissions(ctx)

	durList := time.Since(fromList).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdfcpu.TimingStats("list permissions", durRead, durVal, durOpt, durList, durTotal)

	return list, nil
}

// ListPermissionsFile returns a list of user access permissions for inFile.
func ListPermissionsFile(inFile string, conf *pdfcpu.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}

	defer func() {
		f.Close()
	}()

	return ListPermissions(f, conf)
}

// SetPermissions sets user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissions(rs io.ReadSeeker, w io.Writer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for setting permissions")
	}
	conf.Cmd = pdfcpu.SETPERMISSIONS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// SetPermissionsFile sets inFile's user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissionsFile(inFile, outFile string, conf *pdfcpu.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for setting permissions")
	}

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
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return SetPermissions(f1, f2, conf)
}

// GetPermissions returns the permissions for rs.
func GetPermissions(rs io.ReadSeeker, conf *pdfcpu.Configuration) (*int16, error) {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
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
func GetPermissionsFile(inFile string, conf *pdfcpu.Configuration) (*int16, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return GetPermissions(f, conf)
}
