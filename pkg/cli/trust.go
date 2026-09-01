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

import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

// ListCertificatesCommand creates a new command to list installed certificates.
func ListCertificatesCommand(json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTCERTIFICATES
	}
	return &Command{
		Mode:     model.LISTCERTIFICATES,
		BoolVal1: json,
		Conf:     conf}
}

// InspectCertificatesCommand creates a new command to inspect certificates.
func InspectCertificatesCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.INSPECTCERTIFICATES
	}
	return &Command{
		Mode:    model.INSPECTCERTIFICATES,
		InFiles: inFiles,
		Conf:    conf}
}

// ImportCertificatesCommand creates a new command to import certificates.
func ImportCertificatesCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.IMPORTCERTIFICATES
	}
	return &Command{
		Mode:    model.IMPORTCERTIFICATES,
		InFiles: inFiles,
		Conf:    conf}
}

// ValidateSignaturesCommand creates a new command to validate encountered digital signatures in inFile.
func ValidateSignaturesCommand(inFile string, all, full bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.VALIDATESIGNATURES
	}
	return &Command{
		Mode:     model.VALIDATESIGNATURES,
		InFile:   &inFile,
		BoolVal1: all,
		BoolVal2: full,
		Conf:     conf}
}

// RemoveSignaturesCommand creates a new command to remove all digital signatures from inFile.
// Writes to outFile if supplied else overwrites inFile.
func RemoveSignaturesCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.REMOVESIGNATURES
	}
	return &Command{
		Mode:    model.REMOVESIGNATURES,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}
