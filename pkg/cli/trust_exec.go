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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hhrutter/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type certificateName struct {
	Organization       []string `json:"organization,omitempty"`
	OrganizationalUnit []string `json:"organizationalUnit,omitempty"`
	CommonName         string   `json:"commonName,omitempty"`
	StreetAddress      []string `json:"streetAddress,omitempty"`
	Locality           []string `json:"locality,omitempty"`
	Province           []string `json:"province,omitempty"`
	PostalCode         []string `json:"postalCode,omitempty"`
	Country            []string `json:"country,omitempty"`
}

type certificateListEntry struct {
	Subject      certificateName `json:"subject"`
	Issuer       certificateName `json:"issuer"`
	SerialNumber string          `json:"serialNumber"`
	NotBefore    string          `json:"notBefore"`
	NotAfter     string          `json:"notAfter"`
	IsCA         bool            `json:"isCA"`
}

type certificateFileEntry struct {
	Name         string                 `json:"name"`
	Certificates []certificateListEntry `json:"certificates,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

func listPEM(fName string, ss *[]string) (int, error) {
	bb, err := os.ReadFile(fName)
	if err != nil {
		return 0, err
	}

	if len(bb) == 0 {
		return 0, errors.New("is empty\n")
	}

	ss1 := []string{}
	for len(bb) > 0 {
		var block *pem.Block
		block, bb = pem.Decode(bb)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		certBytes := block.Bytes
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			ss1 = append(ss1, fmt.Sprintf("%v\n", err))
			continue
		}
		*ss = append(*ss, model.CertString(cert))
	}

	sort.Strings(ss1)
	for i, s := range ss1 {
		*ss = append(*ss, fmt.Sprintf("%03d:\n%s", i+1, s))
	}

	return len(ss1), nil
}

func listP7C(fName string, ss *[]string) (int, error) {
	bb, err := os.ReadFile(fName)
	if err != nil {
		return 0, err
	}

	if len(bb) == 0 {
		return 0, errors.New("is empty\n")
	}

	p7, err := pkcs7.Parse(bb)
	if err != nil {
		return 0, err
	}

	ss1 := []string{}
	for _, cert := range p7.Certificates {
		ss1 = append(ss1, model.CertString(cert))
	}

	sort.Strings(ss1)
	for i, s := range ss1 {
		*ss = append(*ss, fmt.Sprintf("%03d:\n%s", i+1, s))
	}

	return len(ss1), nil
}

// ListCertificatesAll returns information about installed certificates.
func ListCertificatesAll(json bool, conf *model.Configuration) ([]string, error) {
	if json {
		return listCertificatesAllJSON()
	}

	// Process *.pem and *.p7c

	if err := os.MkdirAll(model.TrustedCertDir, 0755); err != nil {
		return nil, err
	}

	count := 0

	var ss []string

	err := filepath.WalkDir(model.TrustedCertDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !model.IsPEM(path) && !model.IsP7C(path) {
			return nil
		}

		ss = append(ss, fmt.Sprintf("%s:\n", strings.TrimPrefix(path, model.TrustedCertDir)))

		if model.IsPEM(path) {
			c, err := listPEM(path, &ss)
			if err != nil {
				ss = append(ss, fmt.Sprintf("%v\n", err))
			}
			count += c
			return nil
		}
		c, err := listP7C(path, &ss)
		if err != nil {
			ss = append(ss, fmt.Sprintf("%v\n", err))
		}
		count += c
		return nil
	})

	ss = append(ss, fmt.Sprintf("trustedCertDir: %s", model.TrustedCertDir))
	ss = append(ss, fmt.Sprintf("total installed certs: %d", count))

	return ss, err
}

func listCertificatesAllJSON() ([]string, error) {
	if err := os.MkdirAll(model.TrustedCertDir, 0755); err != nil {
		return nil, err
	}

	files, count, err := certificateFilesJSON(model.TrustedCertDir)
	if err != nil {
		return nil, err
	}

	s := struct {
		Header              pdfcpu.Header          `json:"header"`
		TrustedCertDir      string                 `json:"trustedCertDir"`
		TotalInstalledCerts int                    `json:"totalInstalledCerts"`
		Files               []certificateFileEntry `json:"files"`
	}{
		Header:              pdfcpu.Header{Version: "pdfcpu " + model.VersionStr, Creation: time.Now().Format("2006-01-02 15:04:05 MST")},
		TrustedCertDir:      model.TrustedCertDir,
		TotalInstalledCerts: count,
		Files:               files,
	}

	bb, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return nil, err
	}

	return []string{string(bb)}, nil
}

func certificateFilesJSON(dir string) ([]certificateFileEntry, int, error) {
	count := 0
	var files []certificateFileEntry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skipCertificateFile(path, d) {
			return nil
		}
		entry := certificateFileJSON(dir, path)
		count += len(entry.Certificates)
		files = append(files, entry)
		return nil
	})

	return files, count, err
}

func skipCertificateFile(path string, d os.DirEntry) bool {
	return d.IsDir() || (!model.IsPEM(path) && !model.IsP7C(path))
}

func certificateFileJSON(dir, path string) certificateFileEntry {
	entry := certificateFileEntry{Name: strings.TrimPrefix(path, dir)}
	certs, err := pdfcpu.LoadCertificatesFile(path)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	sort.Slice(certs, func(i, j int) bool {
		return model.CertString(certs[i]) < model.CertString(certs[j])
	})
	entry.Certificates = certificateListEntries(certs)
	return entry
}

func certificateListEntries(certs []*x509.Certificate) []certificateListEntry {
	entries := make([]certificateListEntry, 0, len(certs))
	for _, cert := range certs {
		entries = append(entries, certificateListEntry{
			Subject:      newCertificateName(cert.Subject),
			Issuer:       newCertificateName(cert.Issuer),
			SerialNumber: cert.SerialNumber.Text(16),
			NotBefore:    cert.NotBefore.Format("2006-01-02"),
			NotAfter:     cert.NotAfter.Format("2006-01-02"),
			IsCA:         cert.IsCA,
		})
	}
	return entries
}

func newCertificateName(name pkix.Name) certificateName {
	return certificateName{
		Organization:       name.Organization,
		OrganizationalUnit: name.OrganizationalUnit,
		CommonName:         name.CommonName,
		StreetAddress:      name.StreetAddress,
		Locality:           name.Locality,
		Province:           name.Province,
		PostalCode:         name.PostalCode,
		Country:            name.Country,
	}
}

// ListCertificates returns installed certificates.
func ListCertificates(cmd *Command) ([]string, error) {
	return ListCertificatesAll(cmd.BoolVal1, cmd.Conf)
}

// ImportCertificates imports certificates.
func ImportCertificates(cmd *Command) ([]string, error) {
	return api.ImportCertificates(cmd.InFiles)
}

// InspectCertificates prints the certificate details.
func InspectCertificates(cmd *Command) ([]string, error) {
	return api.InspectCertificates(cmd.InFiles)
}

// ValidateSignatures validates contained digital signature integrity.
func ValidateSignatures(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}

		f, err := os.CreateTemp("", "pdfcpu-signatures-stdin-*.pdf")
		if err != nil {
			return nil, err
		}
		name := f.Name()
		defer os.Remove(name)

		if _, err := io.Copy(f, rs); err != nil {
			_ = f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}

		return api.ValidateSignaturesFile(name, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
	}

	return api.ValidateSignaturesFile(*cmd.InFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

// RemoveSignatures removes contained digital signatures.
func RemoveSignatures(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveSignaturesFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.RemoveSignatures(rs, w, cmd.Conf)
}
