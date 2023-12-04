/*
Copyright 2018 The pdfcpu Authors.

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

package model

import (
	"fmt"

	"github.com/pkg/errors"
)

// VersionStr is the current pdfcpu version.
var VersionStr = "v0.6.0 dev"

// Version is a type for the internal representation of PDF versions.
type Version int

const (
	V10 Version = iota
	V11
	V12
	V13
	V14
	V15
	V16
	V17
	V20
)

// PDFVersion returns the PDFVersion for a version string.
func PDFVersion(versionStr string) (Version, error) {

	switch versionStr {
	case "1.0":
		return V10, nil
	case "1.1":
		return V11, nil
	case "1.2":
		return V12, nil
	case "1.3":
		return V13, nil
	case "1.4":
		return V14, nil
	case "1.5":
		return V15, nil
	case "1.6":
		return V16, nil
	case "1.7":
		return V17, nil
	case "2.0":
		return V20, nil
	}

	return -1, errors.New(versionStr)
}

// String returns a string representation for a given PDFVersion.
func (v Version) String() string {
	if v == V20 {
		return "2.0"
	}
	return "1." + fmt.Sprintf("%d", v)
}
