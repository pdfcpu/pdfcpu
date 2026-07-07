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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type cryptoCommandOperation func(*Command) ([]string, error)

func TestCryptoCommandsRejectMissingFields(t *testing.T) {
	inFile, empty, fileOut := "in.pdf", "", ""
	operations := []struct {
		name string
		fn   cryptoCommandOperation
	}{
		{name: "encrypt", fn: Encrypt},
		{name: "decrypt", fn: Decrypt},
	}
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: api.ErrMissingPDFInput},
		{name: "nil input", cmd: &Command{}, want: api.ErrMissingPDFInput},
		{name: "empty input", cmd: &Command{InFile: &empty}, want: api.ErrMissingPDFInput},
		{name: "nil output", cmd: &Command{InFile: &inFile}, want: api.ErrMissingPDFOutput},
		{
			name: "nil configuration",
			cmd:  &Command{InFile: &inFile, OutFile: &fileOut},
			want: api.ErrMissingConfiguration,
		},
	}

	for _, op := range operations {
		for _, tt := range tests {
			t.Run(op.name+" "+tt.name, func(t *testing.T) {
				if _, err := op.fn(tt.cmd); !errors.Is(err, tt.want) {
					t.Fatalf("got %v, want %v", err, tt.want)
				}
			})
		}
	}
}

func TestCryptoCommandConstructorsSupplyConfiguration(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
		fn   func(string, string, *model.Configuration) *Command
	}{
		{name: "encrypt", mode: model.ENCRYPT, fn: EncryptCommand},
		{name: "decrypt", mode: model.DECRYPT, fn: DecryptCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.fn("in.pdf", "out.pdf", nil)
			if cmd == nil || cmd.InFile == nil || cmd.OutFile == nil || cmd.Conf == nil {
				t.Fatalf("incomplete command: %#v", cmd)
			}
			if cmd.Mode != tt.mode || cmd.Conf.Cmd != tt.mode {
				t.Fatalf("got command modes %d/%d, want %d", cmd.Mode, cmd.Conf.Cmd, tt.mode)
			}
		})
	}
}

func TestCryptoCLIFileErrorsPreserveAPIContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := ""
	operations := []struct {
		name string
		fn   cryptoCommandOperation
	}{
		{name: "encrypt", fn: Encrypt},
		{name: "decrypt", fn: Decrypt},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{
				InFile:  &missingInput,
				OutFile: &outFile,
				Conf:    model.NewDefaultConfiguration(),
			}
			_, err := tt.fn(cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": open input " + missingInput
			if strings.Count(err.Error(), want) != 1 {
				t.Fatalf("expected one %q context, got %q", want, err)
			}
		})
	}
}

func TestCryptoCLIStreamingIOErrorsIncludeOperationContext(t *testing.T) {
	operations := []struct {
		name string
		fn   cryptoCommandOperation
	}{
		{name: "encrypt", fn: Encrypt},
		{name: "decrypt", fn: Decrypt},
	}

	for _, tt := range operations {
		t.Run(tt.name+" open input", func(t *testing.T) {
			missingInput := filepath.Join(t.TempDir(), "missing.pdf")
			outFile := "-"
			cmd := &Command{
				InFile:  &missingInput,
				OutFile: &outFile,
				Conf:    model.NewDefaultConfiguration(),
			}
			_, err := tt.fn(cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": open input " + missingInput
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %q", want, err)
			}
		})

		t.Run(tt.name+" create output", func(t *testing.T) {
			useStdin(t, "not a PDF")
			inFile := "-"
			outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
			cmd := &Command{
				InFile:  &inFile,
				OutFile: &outFile,
				Conf:    model.NewDefaultConfiguration(),
			}
			_, err := tt.fn(cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": create output " + outFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %q", want, err)
			}
		})
	}
}

func TestCryptoCLIStreamingFailurePreservesExistingOutput(t *testing.T) {
	operations := []struct {
		name string
		fn   cryptoCommandOperation
	}{
		{name: "encrypt", fn: Encrypt},
		{name: "decrypt", fn: Decrypt},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "not a PDF")
			inFile := "-"
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			want := []byte("existing output")
			if err := os.WriteFile(outFile, want, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := &Command{
				InFile:  &inFile,
				OutFile: &outFile,
				Conf:    model.NewDefaultConfiguration(),
			}
			_, err := tt.fn(cmd)
			if err == nil || !strings.Contains(err.Error(), tt.name+":") {
				t.Fatalf("expected %s context, got %v", tt.name, err)
			}
			got, readErr := os.ReadFile(outFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("existing output changed: got %q, want %q", got, want)
			}
		})
	}
}

// TestSecurityMutationCLIStreamingIOErrorsIncludeOperationContext verifies operation-aware stream setup.
func TestSecurityMutationCLIStreamingIOErrorsIncludeOperationContext(t *testing.T) {
	operations := []struct {
		name      string
		fn        cryptoCommandOperation
		passwords bool
	}{
		{name: "change user password", fn: ChangeUserPassword, passwords: true},
		{name: "change owner password", fn: ChangeOwnerPassword, passwords: true},
		{name: "set permissions", fn: SetPermissions},
	}

	for _, tt := range operations {
		t.Run(tt.name+" open input", func(t *testing.T) {
			inFile := filepath.Join(t.TempDir(), "missing.pdf")
			outFile := "-"
			cmd := securityMutationCommand(
				&inFile,
				&outFile,
				model.NewDefaultConfiguration(),
				tt.passwords,
			)
			_, err := tt.fn(cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": open input " + inFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
		})

		t.Run(tt.name+" create output", func(t *testing.T) {
			useStdin(t, "not a PDF")
			inFile := "-"
			outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
			cmd := securityMutationCommand(
				&inFile,
				&outFile,
				model.NewDefaultConfiguration(),
				tt.passwords,
			)
			_, err := tt.fn(cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": create output " + outFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
		})
	}
}

// TestSecurityMutationCLIStreamingFailurePreservesExistingOutput verifies protected stream finalization.
func TestSecurityMutationCLIStreamingFailurePreservesExistingOutput(t *testing.T) {
	operations := []struct {
		name      string
		fn        cryptoCommandOperation
		passwords bool
	}{
		{name: "change user password", fn: ChangeUserPassword, passwords: true},
		{name: "change owner password", fn: ChangeOwnerPassword, passwords: true},
		{name: "set permissions", fn: SetPermissions},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "not a PDF")
			inFile := "-"
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			want := []byte("existing output")
			if err := os.WriteFile(outFile, want, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := securityMutationCommand(
				&inFile,
				&outFile,
				model.NewDefaultConfiguration(),
				tt.passwords,
			)
			_, err := tt.fn(cmd)
			if err == nil || !strings.Contains(err.Error(), tt.name+":") {
				t.Fatalf("expected %q operation context, got %v", tt.name, err)
			}
			if !errors.Is(err, pdfcpu.ErrCorruptHeader) {
				t.Fatalf("got %v, want %v", err, pdfcpu.ErrCorruptHeader)
			}
			got, readErr := os.ReadFile(outFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("existing output changed: got %q, want %q", got, want)
			}
			matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(outFile), ".out.pdf.tmp-*"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if len(matches) != 0 {
				t.Fatalf("temporary outputs remain: %v", matches)
			}
		})
	}
}

func securityMutationCommand(inFile, outFile *string, conf *model.Configuration, passwords bool) *Command {
	cmd := &Command{
		InFile:  inFile,
		OutFile: outFile,
		Conf:    conf,
	}
	if passwords {
		oldPW, newPW := "old", "new"
		cmd.PWOld = &oldPW
		cmd.PWNew = &newPW
	}
	return cmd
}

// TestSecurityMutationCommandsRejectMissingFields verifies mutation commands validate their execution boundary.
func TestSecurityMutationCommandsRejectMissingFields(t *testing.T) {
	inFile, empty, outFile := "in.pdf", "", ""
	conf := model.NewDefaultConfiguration()
	operations := []struct {
		name      string
		fn        cryptoCommandOperation
		passwords bool
	}{
		{name: "change user password", fn: ChangeUserPassword, passwords: true},
		{name: "change owner password", fn: ChangeOwnerPassword, passwords: true},
		{name: "set permissions", fn: SetPermissions},
	}
	tests := []struct {
		name string
		cmd  func(bool) *Command
		want error
	}{
		{
			name: "nil command",
			cmd:  func(bool) *Command { return nil },
			want: api.ErrMissingPDFInput,
		},
		{
			name: "nil input",
			cmd: func(passwords bool) *Command {
				return securityMutationCommand(nil, &outFile, conf, passwords)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "empty input",
			cmd: func(passwords bool) *Command {
				return securityMutationCommand(&empty, &outFile, conf, passwords)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "nil output",
			cmd: func(passwords bool) *Command {
				return securityMutationCommand(&inFile, nil, conf, passwords)
			},
			want: api.ErrMissingPDFOutput,
		},
		{
			name: "nil configuration",
			cmd: func(passwords bool) *Command {
				return securityMutationCommand(&inFile, &outFile, nil, passwords)
			},
			want: api.ErrMissingConfiguration,
		},
	}

	for _, op := range operations {
		for _, tt := range tests {
			t.Run(op.name+" "+tt.name, func(t *testing.T) {
				if _, err := op.fn(tt.cmd(op.passwords)); !errors.Is(err, tt.want) {
					t.Fatalf("got %v, want %v", err, tt.want)
				}
			})
		}
	}
}

// TestPasswordChangeCommandsRejectMissingPasswordPointers verifies password pointers are checked before use.
func TestPasswordChangeCommandsRejectMissingPasswordPointers(t *testing.T) {
	inFile, outFile, oldPW, newPW := "in.pdf", "", "old", "new"
	conf := model.NewDefaultConfiguration()
	operations := []struct {
		name string
		fn   cryptoCommandOperation
	}{
		{name: "change user password", fn: ChangeUserPassword},
		{name: "change owner password", fn: ChangeOwnerPassword},
	}
	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "nil old password",
			cmd: &Command{
				InFile:  &inFile,
				OutFile: &outFile,
				PWNew:   &newPW,
				Conf:    conf,
			},
			want: "missing old password",
		},
		{
			name: "nil new password",
			cmd: &Command{
				InFile:  &inFile,
				OutFile: &outFile,
				PWOld:   &oldPW,
				Conf:    conf,
			},
			want: "missing new password",
		},
	}

	for _, op := range operations {
		for _, tt := range tests {
			t.Run(op.name+" "+tt.name, func(t *testing.T) {
				_, err := op.fn(tt.cmd)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("expected %q, got %v", tt.want, err)
				}
			})
		}
	}
}

// TestListPermissionsRejectsMissingFields verifies list commands reject malformed input collections.
func TestListPermissionsRejectsMissingFields(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: api.ErrMissingPDFInput},
		{name: "missing inputs", cmd: &Command{Conf: conf}, want: api.ErrMissingPDFInput},
		{name: "empty input", cmd: &Command{InFiles: []string{""}, Conf: conf}, want: api.ErrMissingPDFInput},
		{name: "nil configuration", cmd: &Command{InFiles: []string{"in.pdf"}}, want: api.ErrMissingConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ListPermissions(tt.cmd); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

// TestListPermissionsFileRejectsMissingFields verifies the exported file helper validates its inputs.
func TestListPermissionsFileRejectsMissingFields(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name    string
		inFiles []string
		conf    *model.Configuration
		want    error
	}{
		{name: "missing inputs", conf: conf, want: api.ErrMissingPDFInput},
		{name: "empty input", inFiles: []string{""}, conf: conf, want: api.ErrMissingPDFInput},
		{name: "nil configuration", inFiles: []string{"in.pdf"}, want: api.ErrMissingConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ListPermissionsFile(tt.inFiles, tt.conf); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

// TestListPermissionsFileReportsCloseError verifies close failures are not discarded.
func TestListPermissionsFileReportsCloseError(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	closeErr := errors.New("close failed")
	original := closeListPermissionsInput
	closeListPermissionsInput = func(f *os.File) error {
		if err := original(f); err != nil {
			return err
		}
		return closeErr
	}
	t.Cleanup(func() {
		closeListPermissionsInput = original
	})

	_, err := ListPermissionsFile([]string{inFile}, model.NewDefaultConfiguration())
	if !errors.Is(err, closeErr) {
		t.Fatalf("got %v, want %v", err, closeErr)
	}
	want := inFile + ": list permissions: close input"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestListPermissionsFileJoinsOperationAndCloseErrors verifies neither per-file failure is hidden.
func TestListPermissionsFileJoinsOperationAndCloseErrors(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	closeErr := errors.New("close failed")
	original := closeListPermissionsInput
	closeListPermissionsInput = func(f *os.File) error {
		if err := original(f); err != nil {
			return err
		}
		return closeErr
	}
	t.Cleanup(func() {
		closeListPermissionsInput = original
	})

	_, err := ListPermissionsFile([]string{inFile}, model.NewDefaultConfiguration())
	if !errors.Is(err, pdfcpu.ErrEmptyInput) || !errors.Is(err, closeErr) {
		t.Fatalf("expected operation and close causes, got %v", err)
	}
	if !strings.HasPrefix(err.Error(), inFile+":") ||
		!strings.Contains(err.Error(), "list permissions: close input") {
		t.Fatalf("expected filename and close context, got %v", err)
	}
}

// TestListPermissionsFileRetainsEachFilename verifies multi-file failures identify their source.
func TestListPermissionsFileRetainsEachFilename(t *testing.T) {
	dir := t.TempDir()
	inFiles := []string{
		filepath.Join(dir, "missing.pdf"),
		filepath.Join(dir, "empty.pdf"),
	}
	if err := os.WriteFile(inFiles[1], nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ListPermissionsFile(inFiles, model.NewDefaultConfiguration())
	if !errors.Is(err, os.ErrNotExist) || !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected open and operation causes, got %v", err)
	}
	for _, inFile := range inFiles {
		if !strings.Contains(err.Error(), inFile+":") {
			t.Fatalf("expected filename %q in %v", inFile, err)
		}
	}
}

// TestListPermissionsMixedInputsPreserveCausesAndSourceLabels verifies mixed failures remain attributable.
func TestListPermissionsMixedInputsPreserveCausesAndSourceLabels(t *testing.T) {
	tests := []struct {
		name      string
		fileNames []string
	}{
		{name: "operation and close failures", fileNames: []string{"input.pdf"}},
		{name: "multiple filenames", fileNames: []string{"first.pdf", "second.pdf"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "not a PDF")
			dir := t.TempDir()
			inFiles := make([]string, 0, len(tt.fileNames)+1)
			for _, name := range tt.fileNames {
				inFile := filepath.Join(dir, name)
				if err := os.WriteFile(inFile, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				inFiles = append(inFiles, inFile)
			}
			inFiles = append(inFiles, "-")

			closeErr := errors.New("close failed")
			original := closeListPermissionsInput
			closeListPermissionsInput = func(f *os.File) error {
				if err := original(f); err != nil {
					return err
				}
				return closeErr
			}
			t.Cleanup(func() {
				closeListPermissionsInput = original
			})

			cmd := &Command{
				InFiles: inFiles,
				Conf:    model.NewDefaultConfiguration(),
			}
			_, err := ListPermissions(cmd)
			for _, want := range []error{pdfcpu.ErrEmptyInput, pdfcpu.ErrCorruptHeader, closeErr} {
				if !errors.Is(err, want) {
					t.Fatalf("got %v, want cause %v", err, want)
				}
			}
			for _, source := range append(inFiles[:len(inFiles)-1], "stdin") {
				label := source + ":"
				if got := strings.Count(err.Error(), label); got != 1 {
					t.Fatalf("got %d %q labels, want 1 in %q", got, label, err)
				}
			}
		})
	}
}
