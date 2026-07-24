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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type positionalReadFailure struct {
	*bytes.Reader
	err error
}

func (r positionalReadFailure) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

// TestValidateSignaturesCommandInitializesBoundaryState verifies the CLI command carries operation state unchanged.
func TestValidateSignaturesCommandInitializesBoundaryState(t *testing.T) {
	cmd := ValidateSignaturesCommand("input.pdf", true, true, nil)
	if cmd.Mode != model.VALIDATESIGNATURES || cmd.Conf == nil || cmd.Conf.Cmd != model.VALIDATESIGNATURES {
		t.Fatalf("unexpected command state: mode=%v conf=%v", cmd.Mode, cmd.Conf)
	}
	if cmd.InFile == nil || *cmd.InFile != "input.pdf" {
		t.Fatalf("unexpected command input: %v", cmd.InFile)
	}
	if !cmd.BoolVal1 || !cmd.BoolVal2 {
		t.Fatalf("validation flags were not retained: all=%t full=%t", cmd.BoolVal1, cmd.BoolVal2)
	}
}

// TestSignatureCLICommandBoundaryGuards verifies public signature commands reject missing input safely.
func TestSignatureCLICommandBoundaryGuards(t *testing.T) {
	empty := ""
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{
			name: "validate nil command",
			call: func() error {
				_, err := ValidateSignatures(nil)
				return err
			},
			want: ErrMissingCommand,
		},
		{
			name: "validate missing input",
			call: func() error {
				_, err := ValidateSignatures(&Command{})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "validate empty input",
			call: func() error {
				_, err := ValidateSignatures(&Command{InFile: &empty})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "remove nil command",
			call: func() error {
				_, err := RemoveSignatures(nil)
				return err
			},
			want: ErrMissingCommand,
		},
		{
			name: "remove missing input",
			call: func() error {
				_, err := RemoveSignatures(&Command{})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "remove empty input",
			call: func() error {
				_, err := RemoveSignatures(&Command{InFile: &empty})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
	}

	for _, tt := range tests {
		if err := tt.call(); !errors.Is(err, tt.want) {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.want, err)
		}
	}

	missing := filepath.Join(t.TempDir(), "missing.pdf")
	if _, err := RemoveSignatures(&Command{InFile: &missing}); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("remove optional output: expected %v, got %v", os.ErrNotExist, err)
	}
}

// TestSignatureCLIFileErrorsRetainOperationAndReadContext verifies ordinary file dispatch.
func TestSignatureCLIFileErrorsRetainOperationAndReadContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "malformed.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command *Command
		op      string
	}{
		{
			name:    "validate",
			command: ValidateSignaturesCommand(inFile, false, false, nil),
			op:      "validate signatures",
		},
		{
			name:    "remove",
			command: RemoveSignaturesCommand(inFile, filepath.Join(t.TempDir(), "out.pdf"), nil),
			op:      "remove signatures",
		},
	}

	for _, tt := range tests {
		_, err := Dispatch(tt.command)
		if err == nil {
			t.Errorf("%s: expected malformed PDF failure", tt.name)
			continue
		}
		for _, want := range []string{tt.op, "read context"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: expected %q, got %q", tt.name, want, err)
			}
		}
		if strings.Contains(err.Error(), "optimize:") {
			t.Errorf("%s: leaked optimize context: %q", tt.name, err)
		}
	}
}

// TestSignatureCLIPreservesPositionalReadCause verifies the CLI boundary
// returns a fatal API signature-read error without flattening its cause.
func TestSignatureCLIPreservesPositionalReadCause(t *testing.T) {
	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("storage unavailable")
	operation := func(
		_ string,
		all, _ bool,
		conf *model.Configuration,
	) ([]string, error) {
		rs := positionalReadFailure{
			Reader: bytes.NewReader(bb),
			err:    cause,
		}
		_, err := api.ValidateSignaturesRaw(rs, all, conf)
		return nil, err
	}

	_, err = validateSignatures(
		ValidateSignaturesCommand("signed.pdf", false, false, nil),
		operation,
	)
	if err == nil {
		t.Fatal("expected fatal positional-read error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	for _, want := range []string{"validate signatures: verify signatures", "read signed data", "signature dict entry ByteRange"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

// TestSignatureCLIStdinErrorsRetainOperationAndCleanup verifies stdin dispatch and failed-output cleanup.
func TestSignatureCLIStdinErrorsRetainOperationAndCleanup(t *testing.T) {
	tests := []struct {
		name    string
		command func(string) *Command
		op      string
	}{
		{
			name: "validate",
			command: func(string) *Command {
				return ValidateSignaturesCommand("-", false, false, nil)
			},
			op: "validate signatures",
		},
		{
			name: "remove",
			command: func(outFile string) *Command {
				return RemoveSignaturesCommand("-", outFile, nil)
			},
			op: "remove signatures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "not a PDF")
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			_, err := Dispatch(tt.command(outFile))
			if err == nil {
				t.Fatal("expected malformed stdin failure")
			}
			for _, want := range []string{tt.op, "read context"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q, got %q", want, err)
				}
			}
			if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("unexpected output file after failure: %v", statErr)
			}
		})
	}
}

// TestSignatureCLIStagesSignedStdin verifies validation retains staging for a non-seekable CLI input.
func TestSignatureCLIStagesSignedStdin(t *testing.T) {
	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)

	out, err := Dispatch(ValidateSignaturesCommand("-", false, false, nil))
	if err != nil {
		t.Fatalf("validate staged stdin: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected signature validation output")
	}
}

// TestSignatureCLINoSignaturesPreservesSentinel verifies dispatch retains semantic error identity.
func TestSignatureCLINoSignaturesPreservesSentinel(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkSimple.pdf")
	tests := []struct {
		name    string
		command *Command
		op      string
	}{
		{
			name:    "validate",
			command: ValidateSignaturesCommand(inFile, false, false, nil),
			op:      "validate signatures",
		},
		{
			name:    "remove",
			command: RemoveSignaturesCommand(inFile, filepath.Join(t.TempDir(), "out.pdf"), nil),
			op:      "remove signatures",
		},
	}

	for _, tt := range tests {
		_, err := Dispatch(tt.command)
		if !errors.Is(err, api.ErrNoSignatures) {
			t.Errorf("%s: expected %v, got %v", tt.name, api.ErrNoSignatures, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.op) {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.op, err)
		}
	}
}
