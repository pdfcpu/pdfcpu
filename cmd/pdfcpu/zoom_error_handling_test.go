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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func preserveZoomCLIFlags(t *testing.T) {
	t.Helper()
	selectedPagesSave, unitSave, forceSave := selectedPages, unit, force
	t.Cleanup(func() {
		selectedPages, unit, force = selectedPagesSave, unitSave, forceSave
	})
	selectedPages, unit, force = "", "", false
}

// TestZoomCLIHandlerRejectsMissingConfigurationAndArguments verifies direct handler guards.
func TestZoomCLIHandlerRejectsMissingConfigurationAndArguments(t *testing.T) {
	preserveZoomCLIFlags(t)
	tests := []struct {
		name string
		conf *model.Configuration
		args []string
		want error
	}{
		{name: "nil configuration", args: []string{"factor:.5", "in.pdf"}, want: api.ErrMissingConfiguration},
		{name: "missing zoom configuration", conf: model.NewDefaultConfiguration(), want: api.ErrMissingZoomConfiguration},
		{name: "missing input", conf: model.NewDefaultConfiguration(), args: []string{"factor:.5"}, want: api.ErrMissingPDFInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleZoomCommand(tt.conf, tt.args); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestZoomCLICommandAcceptsOnlyDocumentedArgumentRange verifies Cobra argument validation.
func TestZoomCLICommandAcceptsOnlyDocumentedArgumentRange(t *testing.T) {
	cmd := zoomCmd()
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "missing input", args: []string{"factor:.5"}, wantErr: true},
		{name: "input", args: []string{"factor:.5", "in.pdf"}},
		{name: "input and output", args: []string{"factor:.5", "in.pdf", "out.pdf"}},
		{name: "extra output", args: []string{"factor:.5", "in.pdf", "out.pdf", "extra.pdf"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Args(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v, wantErr=%t", err, tt.wantErr)
			}
		})
	}
}

// TestZoomCLIConfigurationErrorIncludesContext verifies malformed configuration context.
func TestZoomCLIConfigurationErrorIncludesContext(t *testing.T) {
	preserveZoomCLIFlags(t)
	err := handleZoomCommand(model.NewDefaultConfiguration(), []string{"bad", "in.pdf"})
	if err == nil || !strings.Contains(err.Error(), "zoom: parse configuration") {
		t.Fatalf("expected configuration context, got %v", err)
	}
}

// TestZoomCLIUnitErrorIncludesContext verifies display-unit phase ownership.
func TestZoomCLIUnitErrorIncludesContext(t *testing.T) {
	preserveZoomCLIFlags(t)
	unit = "bad"
	err := handleZoomCommand(model.NewDefaultConfiguration(), []string{"factor:.5", "in.pdf"})
	if err == nil || !strings.Contains(err.Error(), "zoom: configure display unit") {
		t.Fatalf("expected display-unit context, got %v", err)
	}
}

// TestZoomCLIPDFArgumentErrorsIncludeContext verifies input and output argument context.
func TestZoomCLIPDFArgumentErrorsIncludeContext(t *testing.T) {
	preserveZoomCLIFlags(t)
	tests := []struct {
		name string
		args []string
	}{
		{name: "input", args: []string{"factor:.5", "in.txt"}},
		{name: "output", args: []string{"factor:.5", "in.pdf", "out.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleZoomCommand(model.NewDefaultConfiguration(), tt.args)
			if err == nil || !strings.Contains(err.Error(), "zoom: parse arguments") {
				t.Fatalf("expected PDF argument context, got %v", err)
			}
		})
	}
}

// TestZoomCLIPageSelectionErrorIncludesContext verifies selected-page parsing context.
func TestZoomCLIPageSelectionErrorIncludesContext(t *testing.T) {
	preserveZoomCLIFlags(t)
	selectedPages = "foo"
	err := handleZoomCommand(model.NewDefaultConfiguration(), []string{"factor:.5", "in.pdf"})
	if err == nil || !strings.Contains(err.Error(), "zoom: parse page selection") {
		t.Fatalf("expected page-selection context, got %v", err)
	}
}
