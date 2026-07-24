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

package api

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
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

type certificateImport struct {
	inFile       string
	outFile      string
	certificates []*x509.Certificate
}

type stagedCertificateImport struct {
	certificateImport
	stageFile   string
	backupFile  string
	hadOriginal bool
	published   bool
}

type certificateImportOperations struct {
	files            fileOperations
	saveCertificates func([]*x509.Certificate, string) error
}

func defaultCertificateImportOperations() certificateImportOperations {
	return certificateImportOperations{
		files:            defaultFileOperations(),
		saveCertificates: pdfcpu.SaveCertificates,
	}
}

func validateCertificateFiles(inFiles []string) error {
	if len(inFiles) == 0 {
		return ErrMissingCertificateInput
	}
	for i, inFile := range inFiles {
		if strings.TrimSpace(inFile) == "" {
			return fmt.Errorf("certificate input %d: %w", i+1, ErrMissingCertificateInput)
		}
	}
	return nil
}

func ensureTrustedCertificateDir() error {
	if err := os.MkdirAll(model.TrustedCertDir, 0755); err != nil {
		return fmt.Errorf("create trusted certificate directory: %w", err)
	}
	return nil
}

func certificateStrings(certs []*x509.Certificate) []string {
	ss := make([]string, 0, len(certs))
	for _, cert := range certs {
		ss = append(ss, model.CertString(cert))
	}
	sort.Strings(ss)
	return ss
}

func appendCertificateFile(path string, ss *[]string) (int, error) {
	certs, err := pdfcpu.LoadCertificatesFile(path)
	if err != nil {
		*ss = append(*ss, fmt.Sprintf("%v\n", err))
		return 0, err
	}
	if model.IsPEM(path) {
		for _, cert := range certs {
			*ss = append(*ss, model.CertString(cert))
		}
		return len(certs), nil
	}
	certStrings := certificateStrings(certs)
	for i, s := range certStrings {
		*ss = append(*ss, fmt.Sprintf("%03d:\n%s", i+1, s))
	}
	return len(certStrings), nil
}

func listCertificatesText() ([]string, error) {
	count := 0
	var ss []string
	var listErr error
	err := filepath.WalkDir(model.TrustedCertDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			listErr = errors.Join(listErr, fmt.Errorf("access %q: %w", path, err))
			return nil
		}
		if skipCertificateFile(path, d) {
			return nil
		}
		ss = append(ss, fmt.Sprintf("%s:\n", strings.TrimPrefix(path, model.TrustedCertDir)))
		n, err := appendCertificateFile(path, &ss)
		count += n
		if err != nil {
			listErr = errors.Join(listErr, fmt.Errorf("certificate file %q: %w", path, err))
		}
		return nil
	})
	ss = append(ss, fmt.Sprintf("trustedCertDir: %s", model.TrustedCertDir))
	ss = append(ss, fmt.Sprintf("total installed certs: %d", count))
	if err != nil {
		listErr = errors.Join(listErr, fmt.Errorf("walk trusted certificate directory: %w", err))
	}
	return ss, listErr
}

func listCertificatesJSON() ([]string, error) {
	files, count, listErr := certificateFilesJSON(model.TrustedCertDir)

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
		return nil, errors.Join(listErr, fmt.Errorf("encode JSON: %w", err))
	}
	return []string{string(bb)}, listErr
}

func certificateFilesJSON(dir string) ([]certificateFileEntry, int, error) {
	count := 0
	var files []certificateFileEntry
	var listErr error
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			listErr = errors.Join(listErr, fmt.Errorf("access %q: %w", path, err))
			return nil
		}
		if skipCertificateFile(path, d) {
			return nil
		}
		entry, err := certificateFileJSON(dir, path)
		count += len(entry.Certificates)
		files = append(files, entry)
		if err != nil {
			listErr = errors.Join(listErr, fmt.Errorf("certificate file %q: %w", path, err))
		}
		return nil
	})
	if err != nil {
		listErr = errors.Join(listErr, fmt.Errorf("walk trusted certificate directory: %w", err))
	}
	return files, count, listErr
}

func skipCertificateFile(path string, d os.DirEntry) bool {
	return d.IsDir() || (!model.IsPEM(path) && !model.IsP7C(path))
}

func certificateFileJSON(dir, path string) (certificateFileEntry, error) {
	entry := certificateFileEntry{Name: strings.TrimPrefix(path, dir)}
	certs, err := pdfcpu.LoadCertificatesFile(path)
	if err != nil {
		entry.Error = err.Error()
		return entry, err
	}
	sort.Slice(certs, func(i, j int) bool {
		return model.CertString(certs[i]) < model.CertString(certs[j])
	})
	entry.Certificates = certificateListEntries(certs)
	return entry, nil
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
func ListCertificates(json bool) (ss []string, err error) {
	defer fault.Catch(&err)
	if err := ensureTrustedCertificateDir(); err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	if json {
		ss, err = listCertificatesJSON()
	} else {
		ss, err = listCertificatesText()
	}
	if err != nil {
		return ss, fmt.Errorf("list certificates: %w", err)
	}
	return ss, nil
}

// ImportCertificates validates and installs certificate files into the pdfcpu configuration directory.
// Existing destinations with the same derived name are replaced transactionally.
func ImportCertificates(inFiles []string) (ss []string, err error) {
	defer fault.Catch(&err)
	return importCertificates(inFiles, defaultCertificateImportOperations())
}

func importCertificates(inFiles []string, ops certificateImportOperations) ([]string, error) {
	if err := validateCertificateFiles(inFiles); err != nil {
		return nil, fmt.Errorf("import certificates: %w", err)
	}

	imports, err := prepareCertificateImports(inFiles)
	if err != nil {
		return nil, fmt.Errorf("import certificates: %w", err)
	}
	if err := ensureTrustedCertificateDir(); err != nil {
		return nil, fmt.Errorf("import certificates: %w", err)
	}
	if err := publishCertificateImports(imports, ops); err != nil {
		return nil, fmt.Errorf("import certificates: publish batch: %w", err)
	}
	model.MarkCertificateStoreChanged()
	return certificateImportSummary(imports), nil
}

func certificateImportDestination(inFile string) string {
	base := filepath.Base(inFile)
	ext := filepath.Ext(base)
	return filepath.Join(model.TrustedCertDir, strings.TrimSuffix(base, ext)+".p7c")
}

func prepareCertificateImports(inFiles []string) ([]certificateImport, error) {
	imports := make([]certificateImport, 0, len(inFiles))
	destinations := map[string]int{}
	var preflightErr error
	for i, inFile := range inFiles {
		outFile := certificateImportDestination(inFile)
		key := strings.ToLower(filepath.Clean(outFile))
		if previous, found := destinations[key]; found {
			preflightErr = errors.Join(
				preflightErr,
				fmt.Errorf(
					"inputs %d %q and %d %q target %q: %w",
					previous+1,
					inFiles[previous],
					i+1,
					inFile,
					outFile,
					ErrDuplicateCertificateDestination,
				),
			)
		} else {
			destinations[key] = i
		}

		certs, err := pdfcpu.LoadCertificatesFile(inFile)
		if err != nil {
			preflightErr = errors.Join(
				preflightErr,
				fmt.Errorf("input %d %q: load certificates: %w", i+1, inFile, err),
			)
		}
		imports = append(imports, certificateImport{
			inFile:       inFile,
			outFile:      outFile,
			certificates: certs,
		})
	}
	return imports, preflightErr
}

func createCertificateTransactionFile(outFile, kind string, ops certificateImportOperations) (string, error) {
	pattern := fmt.Sprintf(".%s.%s-*", filepath.Base(outFile), kind)
	f, err := ops.files.createTempFn(filepath.Dir(outFile), pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := ops.files.closeFile(f, "close "+kind+" file"); err != nil {
		return "", errors.Join(err, ops.files.removeFile(name, "remove "+kind+" file"))
	}
	return name, nil
}

func cleanupCertificateImports(staged []stagedCertificateImport, removeBackups bool, ops certificateImportOperations) error {
	var err error
	for i := range staged {
		file := &staged[i]
		err = errors.Join(err, ops.files.removeFile(file.stageFile, "remove certificate staging file"))
		if file.backupFile != "" && (removeBackups || !file.hadOriginal) {
			err = errors.Join(err, ops.files.removeFile(file.backupFile, "remove certificate backup file"))
		}
	}
	return err
}

func stageCertificateImports(imports []certificateImport, ops certificateImportOperations) ([]stagedCertificateImport, error) {
	staged := make([]stagedCertificateImport, 0, len(imports))
	for i, imp := range imports {
		stageFile, err := createCertificateTransactionFile(imp.outFile, "stage", ops)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("input %d %q: create staging file: %w", i+1, imp.inFile, err),
				cleanupCertificateImports(staged, true, ops),
			)
		}
		staged = append(staged, stagedCertificateImport{certificateImport: imp, stageFile: stageFile})
		if err := ops.saveCertificates(imp.certificates, stageFile); err != nil {
			return nil, errors.Join(
				fmt.Errorf("input %d %q: encode staging file: %w", i+1, imp.inFile, err),
				cleanupCertificateImports(staged, true, ops),
			)
		}
	}
	return staged, nil
}

func backupCertificateDestinations(staged []stagedCertificateImport, ops certificateImportOperations) error {
	for i := range staged {
		file := &staged[i]
		if _, err := ops.files.statFn(file.outFile); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("input %d %q: inspect destination %q: %w", i+1, file.inFile, file.outFile, err)
		}
		backupFile, err := createCertificateTransactionFile(file.outFile, "backup", ops)
		if err != nil {
			return fmt.Errorf("input %d %q: create backup file: %w", i+1, file.inFile, err)
		}
		file.backupFile = backupFile
		if err := ops.files.removeFile(backupFile, "remove certificate backup placeholder"); err != nil {
			return fmt.Errorf("input %d %q: %w", i+1, file.inFile, err)
		}
		if err := ops.files.replaceFile(file.outFile, backupFile, "backup certificate destination"); err != nil {
			return fmt.Errorf("input %d %q: %w", i+1, file.inFile, err)
		}
		file.hadOriginal = true
	}
	return nil
}

func rollbackCertificateImports(staged []stagedCertificateImport, ops certificateImportOperations) error {
	var err error
	for i := len(staged) - 1; i >= 0; i-- {
		file := &staged[i]
		if file.published {
			err = errors.Join(err, ops.files.removeFile(file.outFile, "remove published certificate"))
			file.published = false
		}
		if file.hadOriginal {
			if restoreErr := ops.files.replaceFile(file.backupFile, file.outFile, "restore certificate destination"); restoreErr != nil {
				err = errors.Join(err, restoreErr, fmt.Errorf("certificate backup retained at %s", file.backupFile))
			} else {
				file.hadOriginal = false
			}
		}
	}
	return errors.Join(err, cleanupCertificateImports(staged, false, ops))
}

func publishCertificateImports(imports []certificateImport, ops certificateImportOperations) error {
	staged, err := stageCertificateImports(imports, ops)
	if err != nil {
		return err
	}
	if err := backupCertificateDestinations(staged, ops); err != nil {
		return errors.Join(err, rollbackCertificateImports(staged, ops))
	}
	for i := range staged {
		file := &staged[i]
		if err := ops.files.replaceFile(file.stageFile, file.outFile, "publish certificate"); err != nil {
			return errors.Join(
				fmt.Errorf("input %d %q: %w", i+1, file.inFile, err),
				rollbackCertificateImports(staged, ops),
			)
		}
		file.published = true
	}
	return cleanupCertificateImports(staged, true, ops)
}

func certificateImportSummary(imports []certificateImport) []string {
	ss := make([]string, 0, len(imports)+1)
	count := 0
	for _, imp := range imports {
		n := len(imp.certificates)
		ss = append(ss, fmt.Sprintf("%s: %d certificates", imp.inFile, n))
		count += n
	}
	return append(ss, fmt.Sprintf("imported %d certificates", count))
}

func inspectCertificate(inFile string, inputIndex, certificateIndex int, cert *x509.Certificate) (string, error) {
	s, err := pdfcpu.InspectCertificate(cert)
	if err != nil {
		return "", fmt.Errorf(
			"inspect certificates: input %d %q: inspect certificate %d: %w",
			inputIndex+1,
			inFile,
			certificateIndex+1,
			err,
		)
	}
	return s, nil
}

// InspectCertificates loads and inspects certificates from inFiles.
func InspectCertificates(inFiles []string) (ss []string, err error) {
	defer fault.Catch(&err)
	if err := validateCertificateFiles(inFiles); err != nil {
		return nil, fmt.Errorf("inspect certificates: %w", err)
	}

	count := 0
	for inputIndex, inFile := range inFiles {
		certs, err := pdfcpu.LoadCertificatesFile(inFile)
		if err != nil {
			return nil, fmt.Errorf("inspect certificates: input %d %q: load certificates: %w", inputIndex+1, inFile, err)
		}

		ss = append(ss, fmt.Sprintf("%s: %d certificates\n", inFile, len(certs)))
		for certificateIndex, cert := range certs {
			s, err := inspectCertificate(inFile, inputIndex, certificateIndex, cert)
			if err != nil {
				return nil, err
			}
			ss = append(ss, fmt.Sprintf("%d:", certificateIndex+1))
			ss = append(ss, s)
			count++
		}
	}

	ss = append(ss, fmt.Sprintf("inspected %d certificates", count))
	return ss, nil
}
