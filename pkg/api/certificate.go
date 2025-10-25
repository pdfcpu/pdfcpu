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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// ImportCertificates validates and installs found certificate files to pdfcpu config dir.
func ImportCertificates(inFiles []string) ([]string, error) {
	count := 0
	overwrite := true
	ss := []string{}
	for _, inFile := range inFiles {
		n, ok, err := pdfcpu.ImportCertificate(inFile, overwrite)
		if err != nil {
			return nil, err
		}
		if !ok {
			ss = append(ss, fmt.Sprintf("%s skipped (already imported)", inFile))
			continue
		}
		ss = append(ss, fmt.Sprintf("%s: %d certificates", inFile, n))
		count += n
	}

	ss = append(ss, fmt.Sprintf("imported %d certificates", count))
	return ss, nil
}

// InspectCertificates loads and inspects certs from indFiles.
func InspectCertificates(inFiles []string) ([]string, error) {
	count := 0
	ss := []string{}

	for _, inFile := range inFiles {

		certs, err := pdfcpu.LoadCertificatesFile(inFile)
		if err != nil {
			return nil, err
		}

		ss = append(ss, fmt.Sprintf("%s: %d certificates\n", inFile, len(certs)))

		for i, cert := range certs {
			s, err := pdfcpu.InspectCertificate(cert)
			if err != nil {
				return nil, err
			}
			ss = append(ss, fmt.Sprintf("%d:", i+1))
			ss = append(ss, s)
			count++
		}

	}

	ss = append(ss, fmt.Sprintf("inspected %d certificates", count))
	return ss, nil
}
