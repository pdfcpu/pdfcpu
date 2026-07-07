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

package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestHandleInstallFontsCommandPreservesAPIInputErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  error
		wantText string
	}{
		{
			name:     "missing input",
			wantErr:  api.ErrMissingFontInput,
			wantText: "install fonts",
		},
		{
			name:     "unsupported mixed input",
			args:     []string{"valid.ttf", "invalid.otf"},
			wantErr:  api.ErrUnsupportedFontFile,
			wantText: "invalid.otf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleInstallFontsCommand(nil, tt.args)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("expected context %q, got %q", tt.wantText, err)
			}
		})
	}
}

func TestFontsCheatSheetCommandAcceptsNoNames(t *testing.T) {
	cmd, _, err := fontsCmd().Find([]string{"cheatsheet"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Args != nil {
		if err := cmd.Args(cmd, nil); err != nil {
			t.Fatalf("expected no font names to select all installed fonts: %v", err)
		}
	}
	if !strings.Contains(cmd.Use, "fontNames") {
		t.Fatalf("expected font-name usage, got %q", cmd.Use)
	}
}

func TestFontsListCommandRejectsArguments(t *testing.T) {
	cmd, _, err := fontsCmd().Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Args == nil {
		t.Fatal("expected explicit no-arguments boundary")
	}
	if err := cmd.Args(cmd, []string{"unexpected"}); err == nil {
		t.Fatal("expected list command to reject arguments")
	}
}

func TestFontsInstallAndCheatSheetArgumentContracts(t *testing.T) {
	install, _, err := fontsCmd().Find([]string{"install"})
	if err != nil {
		t.Fatal(err)
	}
	if install.Args == nil || install.Args(install, nil) == nil {
		t.Fatal("expected font installation to require inputs")
	}

	cheatsheet, _, err := fontsCmd().Find([]string{"cheatsheet"})
	if err != nil {
		t.Fatal(err)
	}
	if cheatsheet.Args != nil {
		if err := cheatsheet.Args(cheatsheet, nil); err != nil {
			t.Fatalf("expected cheatsheet names to remain optional: %v", err)
		}
	}
}
