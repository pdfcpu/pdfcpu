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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type commandFactory func(*model.Configuration) *Command

type commandFactoryCase struct {
	name string
	mode model.CommandMode
	new  commandFactory
}

func commandFactories() []commandFactoryCase {
	return []commandFactoryCase{
		{"content", model.LISTBOOKMARKS, func(conf *model.Configuration) *Command {
			return ListBookmarksCommand("in.pdf", conf)
		}},
		{"document", model.VALIDATE, func(conf *model.Configuration) *Command {
			return ValidateCommand([]string{"in.pdf"}, conf)
		}},
		{"extract", model.EXTRACTIMAGES, func(conf *model.Configuration) *Command {
			return ExtractImagesCommand("in.pdf", "out", nil, conf)
		}},
		{"forms", model.LISTFORMFIELDS, func(conf *model.Configuration) *Command {
			return ListFormFieldsCommand([]string{"in.pdf"}, conf)
		}},
		{"pages", model.ROTATE, func(conf *model.Configuration) *Command {
			return RotateCommand("in.pdf", "out.pdf", 90, nil, conf)
		}},
		{"resources", model.LISTFONTS, ListFontsCommand},
		{"security", model.ENCRYPT, func(conf *model.Configuration) *Command {
			return EncryptCommand("in.pdf", "out.pdf", conf)
		}},
		{"trust", model.LISTCERTIFICATES, func(conf *model.Configuration) *Command {
			return ListCertificatesCommand(false, conf)
		}},
	}
}

func TestCommandConstructorsPreserveSuppliedConfiguration(t *testing.T) {
	const originalMode = model.CommandMode(-1)

	for _, tt := range commandFactories() {
		t.Run(tt.name, func(t *testing.T) {
			conf := &model.Configuration{Cmd: originalMode}
			cmd := tt.new(conf)

			if cmd.Mode != tt.mode {
				t.Fatalf("command mode: got %d, want %d", cmd.Mode, tt.mode)
			}
			if cmd.Conf != conf {
				t.Fatal("constructor replaced the supplied configuration")
			}
			if conf.Cmd != originalMode {
				t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, originalMode)
			}
		})
	}
}

func TestCommandConstructorsDefaultNilConfiguration(t *testing.T) {
	for _, tt := range commandFactories() {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.new(nil)

			if cmd.Conf == nil {
				t.Fatal("constructor did not create a default configuration")
			}
			if cmd.Conf.Cmd != tt.mode {
				t.Fatalf("default configuration mode: got %d, want %d", cmd.Conf.Cmd, tt.mode)
			}
		})
	}
}

func TestConfigurationForModeOwnership(t *testing.T) {
	t.Run("matching mode reuses configuration", func(t *testing.T) {
		conf := &model.Configuration{Cmd: model.DUMP}
		if got := configurationForMode(conf, model.DUMP); got != conf {
			t.Fatal("matching mode cloned configuration")
		}
	})

	t.Run("different mode clones configuration", func(t *testing.T) {
		const originalMode = model.CommandMode(-1)
		conf := &model.Configuration{Cmd: originalMode}
		got := configurationForMode(conf, model.DUMP)
		if got == conf {
			t.Fatal("different mode reused caller configuration")
		}
		if got.Cmd != model.DUMP {
			t.Fatalf("operation mode: got %d, want %d", got.Cmd, model.DUMP)
		}
		if conf.Cmd != originalMode {
			t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, originalMode)
		}
	})
}

func TestDirectModeSettingHandlersPreserveSuppliedConfiguration(t *testing.T) {
	invalidPDF := filepath.Join(t.TempDir(), "invalid.pdf")
	if err := os.WriteFile(invalidPDF, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		run  func(*model.Configuration) error
	}{
		{"dump", func(conf *model.Configuration) error {
			_, err := dump(t.Context(), DumpCommand("missing.pdf", []int{0, 0}, conf))
			return err
		}},
		{"stdout page extraction", func(conf *model.Configuration) error {
			_, err := extractPages(t.Context(), ExtractPagesCommand(invalidPDF, "-", []string{"1"}, conf))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const originalMode = model.CommandMode(-1)
			conf := &model.Configuration{Cmd: originalMode}
			if err := tt.run(conf); err == nil {
				t.Fatal("expected operation error")
			}
			if conf.Cmd != originalMode {
				t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, originalMode)
			}
		})
	}
}
