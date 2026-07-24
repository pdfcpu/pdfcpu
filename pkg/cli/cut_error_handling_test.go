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
)

type cutCLIExec func(*Command) ([]string, error)

func runCutCLIWithoutPanic(t *testing.T, run cutCLIExec, cmd *Command) (err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	_, err = run(cmd)
	return err
}

// TestCutCLIExecutorsRejectMissingFields verifies exported slice command boundary guards.
func TestCutCLIExecutorsRejectMissingFields(t *testing.T) {
	inFile, empty, outDir := "in.pdf", "", "out"
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: ErrMissingCommand},
		{name: "missing input", cmd: &Command{}, want: api.ErrMissingPDFInput},
		{name: "empty input", cmd: &Command{InFile: &empty}, want: api.ErrMissingPDFInput},
		{name: "missing output file", cmd: &Command{InFile: &inFile}, want: api.ErrMissingPDFOutput},
		{name: "missing output directory", cmd: &Command{InFile: &inFile, OutFile: &empty}, want: api.ErrMissingPDFOutput},
		{name: "empty output directory", cmd: &Command{InFile: &inFile, OutFile: &empty, OutDir: &empty}, want: api.ErrMissingPDFOutput},
		{name: "missing cut configuration", cmd: &Command{InFile: &inFile, OutFile: &empty, OutDir: &outDir}, want: api.ErrMissingCutConfiguration},
	}
	operations := []struct {
		name string
		run  cutCLIExec
	}{
		{name: "poster", run: Poster},
		{name: "ndown", run: NDown},
		{name: "cut", run: Cut},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					err := runCutCLIWithoutPanic(t, operation.run, tt.cmd)
					if !errors.Is(err, tt.want) {
						t.Fatalf("expected %v, got %v", tt.want, err)
					}
				})
			}
		})
	}
}
