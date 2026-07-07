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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestFontExecutorsRejectNilCommands(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func(*Command) ([]string, error)
	}{
		{name: "list", fn: ListFonts},
		{name: "install", fn: InstallFonts},
		{name: "cheatsheet", fn: CreateCheatSheetsFonts},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn(nil)
			if err == nil || !strings.Contains(err.Error(), "missing command") {
				t.Fatalf("expected missing command error, got %v", err)
			}
		})
	}
}

func TestValidateFontsCommandRejectsInappropriateModes(t *testing.T) {
	if err := validateFontsCommand(&Command{Mode: model.LISTFONTS}, model.INSTALLFONTS); err == nil {
		t.Fatal("expected mismatched mode error")
	}
	if err := validateFontsCommand(&Command{Mode: model.LISTFONTS}, model.VALIDATE); err == nil {
		t.Fatal("expected inappropriate expected mode error")
	}
}

func TestInstallFontsExecutorPreservesAPICause(t *testing.T) {
	_, err := InstallFonts(&Command{Mode: model.INSTALLFONTS})
	if !errors.Is(err, api.ErrMissingFontInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingFontInput, err)
	}
}
