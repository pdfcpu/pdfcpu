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
	"bytes"
	"errors"
	stdlog "log"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func TestInstallFontsCommandRendersStructuredWarnings(t *testing.T) {
	var output bytes.Buffer
	log.SetCLILogger(stdlog.New(&output, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	warning := errors.New("temporary backup retained")
	install := func([]string) (api.FontInstallResult, error) {
		return api.FontInstallResult{Warnings: []error{warning}}, nil
	}
	if _, err := installFontsCommand(InstallFontsCommand([]string{"Demo.ttf"}, nil), install); err != nil {
		t.Fatal(err)
	}
	want := "installing to " + font.UserFontDir + "...\nwarning: " + warning.Error() + "\n"
	if got := output.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCreateCheatSheetsFontsCommandRendersPublishedPaths(t *testing.T) {
	var output bytes.Buffer
	log.SetCLILogger(stdlog.New(&output, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	wantErr := errors.New("published output cleanup failed")
	create := func([]string) (api.FontCheatSheetResult, error) {
		return api.FontCheatSheetResult{Paths: []string{"Demo_BMP.pdf", "Demo_SMP.pdf"}}, wantErr
	}
	_, err := createCheatSheetsFontsCommand(CreateCheatSheetsFontsCommand([]string{"Demo"}, nil), create)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	want := "Demo_BMP.pdf\nDemo_SMP.pdf\n"
	if got := output.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFontStructuredPresentationHonorsQuietMode(t *testing.T) {
	log.SetCLILogger(nil)
	install := func([]string) (api.FontInstallResult, error) {
		return api.FontInstallResult{Warnings: []error{errors.New("cleanup warning")}}, nil
	}
	if _, err := installFontsCommand(InstallFontsCommand([]string{"Demo.ttf"}, nil), install); err != nil {
		t.Fatal(err)
	}

	create := func([]string) (api.FontCheatSheetResult, error) {
		return api.FontCheatSheetResult{Paths: []string{"Demo_BMP.pdf"}}, nil
	}
	if _, err := createCheatSheetsFontsCommand(CreateCheatSheetsFontsCommand([]string{"Demo"}, nil), create); err != nil {
		t.Fatal(err)
	}
}
