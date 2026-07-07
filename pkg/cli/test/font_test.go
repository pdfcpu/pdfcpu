/*
Copyright 2020 The pdfcpu Authors.

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

package test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

// TestInstallFontsCommand verifies install fonts command.
func TestInstallFontsCommand(t *testing.T) {
	msg := "TestInstallFontsCommand"
	userFontName := filepath.Join(fontDir, "Roboto-Regular.ttf")
	cmd := cli.InstallFontsCommand([]string{userFontName}, conf)
	if _, err := cli.Dispatch(cmd); err != nil {
		t.Fatalf("%s install fonts: %v\n", msg, err)
	}
}

// TestInstallTTCFontsCommandReportsMissingInput verifies install errors reach CLI callers.
func TestInstallTTCFontsCommandReportsMissingInput(t *testing.T) {
	userFontName := filepath.Join(fontDir, "Songti.ttc")
	cmd := cli.InstallFontsCommand([]string{userFontName}, conf)
	_, err := cli.Dispatch(cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), userFontName) {
		t.Fatalf("expected font filename context, got %q", err)
	}
}

// TestListFontsCommand verifies list fonts command.
func TestListFontsCommand(t *testing.T) {
	msg := "TestListFontsCommand"
	cmd := cli.ListFontsCommand(conf)
	if _, err := cli.Dispatch(cmd); err != nil {
		t.Fatalf("%s list fonts: %v\n", msg, err)
	}
}

// TestCreateCheatSheetsFontsCommandReportsUnknownFont verifies unknown fonts reach CLI callers.
func TestCreateCheatSheetsFontsCommandReportsUnknownFont(t *testing.T) {
	userFontName := filepath.Join(fontDir, "Songti.ttc")
	cmd := cli.CreateCheatSheetsFontsCommand([]string{userFontName}, conf)
	_, err := cli.Dispatch(cmd)
	if !errors.Is(err, api.ErrUserFontNotFound) {
		t.Fatalf("expected %v, got %v", api.ErrUserFontNotFound, err)
	}
	if !strings.Contains(err.Error(), userFontName) {
		t.Fatalf("expected font name context, got %q", err)
	}
}
