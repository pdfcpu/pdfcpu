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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func preserveCutCLIFlags(t *testing.T) {
	t.Helper()
	selectedPagesSave, unitSave, forceSave := selectedPages, unit, force
	t.Cleanup(func() {
		selectedPages, unit, force = selectedPagesSave, unitSave, forceSave
	})
	selectedPages, unit, force = "", "", false
}

// TestCutCLIHandlersRejectMissingConfigurationAndArguments verifies Cobra handler boundary guards.
func TestCutCLIHandlersRejectMissingConfigurationAndArguments(t *testing.T) {
	preserveCutCLIFlags(t)
	operations := []struct {
		name string
		run  func(*model.Configuration, []string) error
	}{
		{name: "poster", run: handlePosterCommand},
		{name: "ndown", run: handleNDownCommand},
		{name: "cut", run: handleCutCommand},
	}
	guards := []struct {
		name string
		conf *model.Configuration
		args []string
		want error
	}{
		{name: "nil configuration", args: []string{"config", "in.pdf", "out"}, want: api.ErrMissingConfiguration},
		{name: "missing cut configuration", conf: model.NewDefaultConfiguration(), want: api.ErrMissingCutConfiguration},
		{name: "missing input", conf: model.NewDefaultConfiguration(), args: []string{"config"}, want: api.ErrMissingPDFInput},
		{name: "missing output", conf: model.NewDefaultConfiguration(), args: []string{"config", "in.pdf"}, want: api.ErrMissingPDFOutput},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, guard := range guards {
				t.Run(guard.name, func(t *testing.T) {
					err := operation.run(guard.conf, guard.args)
					if !errors.Is(err, guard.want) {
						t.Fatalf("expected %v, got %v", guard.want, err)
					}
				})
			}
		})
	}
}

// TestNDownArgsRejectsShortArguments verifies direct argument parsing never indexes missing fields.
func TestNDownArgsRejectsShortArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want error
	}{
		{name: "missing cut configuration", want: api.ErrMissingCutConfiguration},
		{name: "missing input", args: []string{"2"}, want: api.ErrMissingPDFInput},
		{name: "missing output", args: []string{"2", "in.pdf"}, want: api.ErrMissingPDFOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, _, err := ndownArgs(tt.args, types.POINTS)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestCutCLIInputAndOutputChecksIncludeOperationPhases verifies filesystem-check ownership.
func TestCutCLIInputAndOutputChecksIncludeOperationPhases(t *testing.T) {
	preserveCutCLIFlags(t)
	outDir := t.TempDir()
	outFile := "existing.pdf"
	if err := os.WriteFile(filepath.Join(outDir, outFile), []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "poster input", run: func() error {
			return handlePosterCommand(model.NewDefaultConfiguration(), []string{"dim:100 100", "in.txt", t.TempDir()})
		}, want: "poster: check input"},
		{name: "ndown input", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"2", "in.txt", t.TempDir()})
		}, want: "ndown: check input"},
		{name: "cut input", run: func() error {
			return handleCutCommand(model.NewDefaultConfiguration(), []string{"hor:.5", "in.txt", t.TempDir()})
		}, want: "cut: check input"},
		{name: "poster output", run: func() error {
			return handlePosterCommand(model.NewDefaultConfiguration(), []string{"dim:100 100", "in.pdf", outDir, outFile})
		}, want: "poster: check output"},
		{name: "ndown output", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"2", "in.pdf", outDir, outFile})
		}, want: "ndown: check output"},
		{name: "cut output", run: func() error {
			return handleCutCommand(model.NewDefaultConfiguration(), []string{"hor:.5", "in.pdf", outDir, outFile})
		}, want: "cut: check output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestCutCLIConfigurationAndArgumentErrorsIncludeContext verifies command-owned parsing phases.
func TestCutCLIConfigurationAndArgumentErrorsIncludeContext(t *testing.T) {
	preserveCutCLIFlags(t)
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "poster configuration", run: func() error {
			return handlePosterCommand(model.NewDefaultConfiguration(), []string{"bad", "in.pdf", t.TempDir()})
		}, want: "poster: parse configuration"},
		{name: "cut configuration", run: func() error {
			return handleCutCommand(model.NewDefaultConfiguration(), []string{"bad", "in.pdf", t.TempDir()})
		}, want: "cut: parse configuration"},
		{name: "ndown value", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"description", "bad", "in.pdf", t.TempDir()})
		}, want: `ndown: parse arguments: parse n-down value "bad"`},
		{name: "ndown configuration", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"5", "in.pdf", t.TempDir()})
		}, want: "ndown: parse arguments: parse configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestCutCLIPageSelectionErrorsIncludeContext verifies operation-specific selection phases.
func TestCutCLIPageSelectionErrorsIncludeContext(t *testing.T) {
	preserveCutCLIFlags(t)
	selectedPages = "foo"
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "poster", run: func() error {
			return handlePosterCommand(model.NewDefaultConfiguration(), []string{"dim:100 100", "in.pdf", t.TempDir()})
		}, want: "poster: parse page selection"},
		{name: "ndown", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"2", "in.pdf", t.TempDir()})
		}, want: "ndown: parse page selection"},
		{name: "cut", run: func() error {
			return handleCutCommand(model.NewDefaultConfiguration(), []string{"hor:.5", "in.pdf", t.TempDir()})
		}, want: "cut: parse page selection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestCutCLIUnitErrorsIncludeContext verifies display-unit phase ownership.
func TestCutCLIUnitErrorsIncludeContext(t *testing.T) {
	preserveCutCLIFlags(t)
	unit = "bad"
	tests := []struct {
		operation string
		run       func() error
	}{
		{operation: "poster", run: func() error {
			return handlePosterCommand(model.NewDefaultConfiguration(), []string{"config", "in.pdf", "out"})
		}},
		{operation: "ndown", run: func() error {
			return handleNDownCommand(model.NewDefaultConfiguration(), []string{"2", "in.pdf", "out"})
		}},
		{operation: "cut", run: func() error {
			return handleCutCommand(model.NewDefaultConfiguration(), []string{"config", "in.pdf", "out"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.operation+": configure display unit") {
				t.Fatalf("expected unit context, got %v", err)
			}
		})
	}
}
