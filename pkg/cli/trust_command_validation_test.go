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
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type trustCommandExecutor func(*Command) ([]string, error)

// TestTrustExecutorsRejectNilCommand verifies every command-based trust executor has a safe nil boundary.
func TestTrustExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  trustCommandExecutor
	}{
		{"ListCertificates", ListCertificates},
		{"ImportCertificates", ImportCertificates},
		{"InspectCertificates", InspectCertificates},
		{"ValidateSignatures", ValidateSignatures},
		{"RemoveSignatures", RemoveSignatures},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(nil)
			if !errors.Is(err, ErrMissingCommand) {
				t.Fatalf("expected %v, got %v", ErrMissingCommand, err)
			}
		})
	}
}

// TestCertificateExecutorsRejectMissingInput verifies certificate paths are checked before trust-store access.
func TestCertificateExecutorsRejectMissingInput(t *testing.T) {
	tests := []struct {
		name string
		run  trustCommandExecutor
		cmd  *Command
	}{
		{"ImportMissing", ImportCertificates, &Command{}},
		{"ImportWhitespace", ImportCertificates, &Command{InFiles: []string{" "}}},
		{"InspectMissing", InspectCertificates, &Command{}},
		{"InspectWhitespace", InspectCertificates, &Command{InFiles: []string{"\t"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(tt.cmd)
			if !errors.Is(err, api.ErrMissingCertificateInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingCertificateInput, err)
			}
		})
	}
}

// TestDispatchRejectsIncompleteTrustCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteTrustCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
		want error
	}{
		{"ImportCertificates", model.IMPORTCERTIFICATES, api.ErrMissingCertificateInput},
		{"InspectCertificates", model.INSPECTCERTIFICATES, api.ErrMissingCertificateInput},
		{"ValidateSignatures", model.VALIDATESIGNATURES, api.ErrMissingPDFInput},
		{"RemoveSignatures", model.REMOVESIGNATURES, api.ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dispatch(&Command{Mode: tt.mode})
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			var panicErr fault.Panic
			if errors.As(err, &panicErr) {
				t.Fatalf("caller error returned as panic: %v", err)
			}
		})
	}
}
