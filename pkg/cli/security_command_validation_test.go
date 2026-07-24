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

type securityCommandExecutor func(*Command) ([]string, error)

// TestSecurityExecutorsRejectNilCommand verifies every public security executor has a safe nil boundary.
func TestSecurityExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  securityCommandExecutor
	}{
		{"Encrypt", Encrypt},
		{"Decrypt", Decrypt},
		{"ChangeUserPassword", ChangeUserPassword},
		{"ChangeOwnerPassword", ChangeOwnerPassword},
		{"ListPermissions", ListPermissions},
		{"SetPermissions", SetPermissions},
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

// TestDispatchRejectsIncompleteSecurityCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteSecurityCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
	}{
		{"Encrypt", model.ENCRYPT},
		{"Decrypt", model.DECRYPT},
		{"ChangeUserPassword", model.CHANGEUPW},
		{"ChangeOwnerPassword", model.CHANGEOPW},
		{"ListPermissions", model.LISTPERMISSIONS},
		{"SetPermissions", model.SETPERMISSIONS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dispatch(&Command{Mode: tt.mode})
			if !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			var panicErr fault.Panic
			if errors.As(err, &panicErr) {
				t.Fatalf("caller error returned as panic: %v", err)
			}
		})
	}
}
