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

type resourceCommandExecutor func(*Command) ([]string, error)

// TestResourceExecutorsRejectNilCommand verifies every public resource executor has a safe nil boundary.
func TestResourceExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  resourceCommandExecutor
	}{
		{"ImportImages", ImportImages},
		{"CreateCheatSheetsFonts", CreateCheatSheetsFonts},
		{"ListFonts", ListFonts},
		{"InstallFonts", InstallFonts},
		{"ListImages", ListImages},
		{"UpdateImages", UpdateImages},
		{"ListAttachments", ListAttachments},
		{"AddAttachments", AddAttachments},
		{"RemoveAttachments", RemoveAttachments},
		{"ExtractAttachments", ExtractAttachments},
		{"ListKeywords", ListKeywords},
		{"AddKeywords", AddKeywords},
		{"RemoveKeywords", RemoveKeywords},
		{"ListProperties", ListProperties},
		{"AddProperties", AddProperties},
		{"RemoveProperties", RemoveProperties},
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

// TestDispatchRejectsIncompleteResourceCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteResourceCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
		want error
	}{
		{"ImportImages", model.IMPORTIMAGES, api.ErrMissingImageInput},
		{"InstallFonts", model.INSTALLFONTS, api.ErrMissingFontInput},
		{"ListImages", model.LISTIMAGES, api.ErrMissingPDFInput},
		{"UpdateImages", model.UPDATEIMAGES, api.ErrMissingPDFInput},
		{"ListAttachments", model.LISTATTACHMENTS, api.ErrMissingPDFInput},
		{"AddAttachments", model.ADDATTACHMENTS, api.ErrMissingPDFInput},
		{"AddPortfolioAttachments", model.ADDATTACHMENTSPORTFOLIO, api.ErrMissingPDFInput},
		{"RemoveAttachments", model.REMOVEATTACHMENTS, api.ErrMissingPDFInput},
		{"ExtractAttachments", model.EXTRACTATTACHMENTS, api.ErrMissingPDFInput},
		{"ListKeywords", model.LISTKEYWORDS, api.ErrMissingPDFInput},
		{"AddKeywords", model.ADDKEYWORDS, api.ErrMissingPDFInput},
		{"RemoveKeywords", model.REMOVEKEYWORDS, api.ErrMissingPDFInput},
		{"ListProperties", model.LISTPROPERTIES, api.ErrMissingPDFInput},
		{"AddProperties", model.ADDPROPERTIES, api.ErrMissingPDFInput},
		{"RemoveProperties", model.REMOVEPROPERTIES, api.ErrMissingPDFInput},
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
