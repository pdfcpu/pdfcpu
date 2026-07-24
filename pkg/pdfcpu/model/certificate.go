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
	"path/filepath"
	"strings"
	"sync/atomic"
)

// TrustedCertDir is the location for installed trusted certificates.
var TrustedCertDir string

// UserCertPool contains the last successfully loaded trusted certificates.
// Deprecated: Certificate trust pools are managed by package pdfcpu.
var UserCertPool *x509.CertPool

var certificateStoreRevision atomic.Uint64

// CertificateStoreRevision returns the current trusted-certificate store revision.
func CertificateStoreRevision() uint64 {
	return certificateStoreRevision.Load()
}

// MarkCertificateStoreChanged advances the trusted-certificate store revision.
func MarkCertificateStoreChanged() {
	certificateStoreRevision.Add(1)
}

// IsPEM Do we need locking?

func IsPEM(fname string) bool {
	return strings.HasSuffix(strings.ToLower(fname), ".pem")
}

// IsP7C reports whether fname is p7c.
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

// CertString returns a string representation for cert.
func CertString(cert *x509.Certificate) string {
	return fmt.Sprintf(
		"    Subject:\n%s\n"+
			"     Issuer:\n%s\n"+
			"    Serial#: %s\n"+
			"       from: %s\n"+
			"       thru: %s\n"+
			"         CA: %t\n",
		nameString(cert.Subject),
		nameString(cert.Issuer),
		cert.SerialNumber.Text(16),
		cert.NotBefore.Format("2006-01-02"),
		cert.NotAfter.Format("2006-01-02"),
		cert.IsCA,
	)
}

// ResetCertificates resets installed trusted certificates to the build defaults.
func ResetCertificates() error {
	if TrustedCertDir == "" {
		path, err := os.UserConfigDir()
		if err != nil {
			path = os.TempDir()
		}
		if err := EnsureDefaultConfigAt(path, false); err != nil {
			return err
		}
	}
	if err := resetCertificatesDir(); err != nil {
		return err
	}
	MarkCertificateStoreChanged()
	return installDefaultCertificates()
}

func resetCertificatesDir() error {
	files, err := os.ReadDir(TrustedCertDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := os.RemoveAll(filepath.Join(TrustedCertDir, file.Name())); err != nil {
			return err
		}
	}
	return nil
}
