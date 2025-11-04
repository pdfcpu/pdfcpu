/*
Copyright 2025 The pdfcpu Authors.

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
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"
	"strings"
)

// CertDir is the location for installed certificates.
var CertDir string

// UserCertPool contains all certificates loaded from CertDir.
var UserCertPool *x509.CertPool

// TODO Do we need locking?
//var UserCertPoolLock = &sync.RWMutex{}

func IsPEM(fname string) bool {
	return strings.HasSuffix(strings.ToLower(fname), ".pem")
}

func IsP7C(fname string) bool {
	return strings.HasSuffix(strings.ToLower(fname), ".p7c")
}

func strSliceString(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	ss1 := []string{}
	ss1 = append(ss1, ss...)
	return strings.Join(ss1, ",")
}

func nameString(subj pkix.Name) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("             org       : %s", strSliceString(subj.Organization)))

	if len(subj.OrganizationalUnit) > 0 {
		sb.WriteString(fmt.Sprintf("\n             unit      : %s", strSliceString(subj.OrganizationalUnit)))
	}

	if len(subj.CommonName) > 0 {
		sb.WriteString(fmt.Sprintf("\n             name      : %s", subj.CommonName))
	}

	if len(subj.StreetAddress) > 0 {
		sb.WriteString(fmt.Sprintf("\n             street    : %s", strSliceString(subj.StreetAddress)))
	}

	if len(subj.Locality) > 0 {
		sb.WriteString(fmt.Sprintf("\n             locality  : %s", strSliceString(subj.Locality)))
	}

	if len(subj.Province) > 0 {
		sb.WriteString(fmt.Sprintf("\n             province  : %s", strSliceString(subj.Province)))
	}

	if len(subj.PostalCode) > 0 {
		sb.WriteString(fmt.Sprintf("\n             postalCode: %s", strSliceString(subj.PostalCode)))
	}

	if len(subj.Country) > 0 {
		sb.WriteString(fmt.Sprintf("\n             country   : %s", strSliceString(subj.Country)))
	}

	return sb.String()
}

func CertString(cert *x509.Certificate) string {

	return fmt.Sprintf(
		"    Subject:\n%s\n"+
			"     Issuer:\n%s\n"+
			"       from: %s\n"+
			"       thru: %s\n"+
			"         CA: %t\n",
		nameString(cert.Subject),
		nameString(cert.Issuer),
		cert.NotBefore.Format("2006-01-02"),
		cert.NotAfter.Format("2006-01-02"),
		cert.IsCA,
	)
}

func ResetCertificates() error {

	// remove certs/*.pem

	path, err := os.UserConfigDir()
	if err != nil {
		path = os.TempDir()
	}
	return EnsureDefaultConfigAt(path, true)
}
