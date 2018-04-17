package types

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	// PDFCPUVersion returns the current pdfcpu version.
	PDFCPUVersion = "0.1.10"

	// PDFCPULongVersion returns pdfcpu's signature.
	PDFCPULongVersion = "golang pdfcpu v" + PDFCPUVersion
)

// PDFVersion is a type for the internal representation of PDF versions.
type PDFVersion int

// Constants for all PDF versions up to v1.7
const (
	V10 PDFVersion = iota
	V11
	V12
	V13
	V14
	V15
	V16
	V17
)

// Version returns the PDFVersion for a version string.
func Version(versionStr string) (PDFVersion, error) {

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
	}

	return -1, errors.New(versionStr)
}

// VersionString returns a string representation for a given PDFVersion.
func VersionString(version PDFVersion) string {
	return "1." + fmt.Sprintf("%d", version)
}
