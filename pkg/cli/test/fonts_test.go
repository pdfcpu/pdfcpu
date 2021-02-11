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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestInstallFontsCommand(t *testing.T) {
	msg := "TestInstallFontsCommand"
	userFontName := filepath.Join(fontDir, "Roboto-Regular.ttf")
	cmd := cli.InstallFontsCommand([]string{userFontName}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s install fonts: %v\n", msg, err)
	}
}

func TestInstallTTCFontsCommand(t *testing.T) {
	msg := "TestInstallTTCFontsCommand"
	userFontName := filepath.Join(fontDir, "Songti.ttc")
	cmd := cli.InstallFontsCommand([]string{userFontName}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s install fonts: %v\n", msg, err)
	}
}

func TestListFontsCommand(t *testing.T) {
	msg := "TestListFontsCommand"
	cmd := cli.ListFontsCommand(nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s list fonts: %v\n", msg, err)
	}
}

func TestCreateCheatSheetsFontsCommand(t *testing.T) {
	msg := "TestCreateCheatSheetsFontsCommand"
	userFontName := filepath.Join(fontDir, "Songti.ttc")
	cmd := cli.CreateCheatSheetsFontsCommand([]string{userFontName}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s create cheat sheets fonts: %v\n", msg, err)
	}
}
