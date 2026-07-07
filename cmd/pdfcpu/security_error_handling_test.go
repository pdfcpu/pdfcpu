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
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

func TestCryptoCLIHandlersRejectMissingConfigurationAndArguments(t *testing.T) {
	defaultEncryptOptions := func() *encryptOptions {
		return &encryptOptions{mode: "aes", key: "256", perm: "none"}
	}
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "decrypt nil configuration",
			run: func() error {
				return handleDecryptCommand(nil, []string{"in.pdf"})
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "decrypt missing input",
			run: func() error {
				return handleDecryptCommand(model.NewDefaultConfiguration(), nil)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "decrypt empty input",
			run: func() error {
				return handleDecryptCommand(model.NewDefaultConfiguration(), []string{""})
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "encrypt nil configuration",
			run: func() error {
				return handleEncryptCommand(nil, []string{"in.pdf"}, defaultEncryptOptions())
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "encrypt missing input",
			run: func() error {
				return handleEncryptCommand(model.NewDefaultConfiguration(), nil, defaultEncryptOptions())
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "encrypt empty input",
			run: func() error {
				return handleEncryptCommand(model.NewDefaultConfiguration(), []string{""}, defaultEncryptOptions())
			},
			want: api.ErrMissingPDFInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCryptoCLIHandlersRejectExcessArguments(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.OwnerPW = "owner"
	opts := &encryptOptions{mode: "aes", key: "256", perm: "none"}
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "encrypt",
			err:  handleEncryptCommand(conf, []string{"in.pdf", "out.pdf", "extra.pdf"}, opts),
		},
		{
			name: "decrypt",
			err:  handleDecryptCommand(conf, []string{"in.pdf", "out.pdf", "extra.pdf"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || !strings.Contains(tt.err.Error(), tt.name+": expected 1 or 2 arguments") {
				t.Fatalf("expected argument count context, got %v", tt.err)
			}
		})
	}
}

func TestEncryptCLIRejectsMissingOptionsAndOwnerPassword(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	if err := handleEncryptCommand(conf, []string{"in.pdf"}, nil); err == nil ||
		!strings.Contains(err.Error(), "encrypt: missing options") {
		t.Fatalf("expected missing-options context, got %v", err)
	}

	opts := &encryptOptions{mode: "aes", key: "256", perm: "none"}
	if err := handleEncryptCommand(conf, []string{"in.pdf"}, opts); err == nil ||
		!strings.Contains(err.Error(), "owner password must not be empty") ||
		!strings.Contains(err.Error(), "--opw") {
		t.Fatalf("expected actionable owner-password error, got %v", err)
	}
}

// TestEncryptCLIMissingOwnerPasswordPreservesSentinel verifies the actionable CLI error remains classifiable.
func TestEncryptCLIMissingOwnerPasswordPreservesSentinel(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	opts := &encryptOptions{mode: "aes", key: "256", perm: "none"}

	err := handleEncryptCommand(conf, []string{"in.pdf"}, opts)
	if !errors.Is(err, pdfcpu.ErrOwnerPasswordRequired) {
		t.Fatalf("got %v, want %v", err, pdfcpu.ErrOwnerPasswordRequired)
	}
	if !strings.Contains(err.Error(), "--opw") {
		t.Fatalf("expected actionable --opw guidance, got %v", err)
	}
}

func TestEncryptCLIFlagErrorsIncludeContext(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.OwnerPW = "owner"
	tests := []struct {
		name string
		opts *encryptOptions
		want string
	}{
		{
			name: "mode",
			opts: &encryptOptions{mode: "bad", key: "256", perm: "none"},
			want: "valid modes",
		},
		{
			name: "rc4 key",
			opts: &encryptOptions{mode: "rc4", key: "41", perm: "none"},
			want: "supported RC4 key lengths",
		},
		{
			name: "aes key",
			opts: &encryptOptions{mode: "aes", key: "41", perm: "none"},
			want: "supported AES key lengths",
		},
		{
			name: "permissions",
			opts: &encryptOptions{mode: "aes", key: "256", perm: "bad"},
			want: "supported permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleEncryptCommand(conf, []string{"in.pdf"}, tt.opts)
			if err == nil || !strings.Contains(err.Error(), "encrypt: validate flags") ||
				!strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected flag context containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestEncryptCLIEmptyKeyUsesDocumentedDefault(t *testing.T) {
	tests := []struct {
		name string
		opts *encryptOptions
		want string
	}{
		{name: "aes", opts: &encryptOptions{mode: "aes"}, want: "256"},
		{name: "default mode", opts: &encryptOptions{}, want: "256"},
		{name: "rc4", opts: &encryptOptions{mode: "rc4"}, want: "128"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEncryptModeFlag(tt.opts); err != nil {
				t.Fatal(err)
			}
			if tt.opts.key != tt.want {
				t.Fatalf("got key length %q, want %q", tt.opts.key, tt.want)
			}
		})
	}
}

// TestEncryptCommandOmittedRC4KeySelects128Bit verifies Cobra's default key is normalized for RC4.
func TestEncryptCommandOmittedRC4KeySelects128Bit(t *testing.T) {
	oldOPW := opw
	t.Cleanup(func() {
		opw = oldOPW
	})

	cmd := encryptCmd()
	if err := cmd.ParseFlags([]string{"--mode", "rc4", "--opw", "owner", "in.pdf", "out.pdf"}); err != nil {
		t.Fatal(err)
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		t.Fatal(err)
	}
	key, err := cmd.Flags().GetString("key")
	if err != nil {
		t.Fatal(err)
	}
	if key != "" {
		t.Fatalf("got omitted key %q, want empty", key)
	}

	opts := &encryptOptions{mode: mode, key: key}
	if err := validateEncryptModeFlag(opts); err != nil {
		t.Fatal(err)
	}
	if opts.mode != "rc4" || opts.key != "128" {
		t.Fatalf("got mode/key %s/%s, want rc4/128", opts.mode, opts.key)
	}
}

// TestEncryptCommandExplicitRC4Key256ReturnsActionableError verifies invalid explicit keys are not normalized.
func TestEncryptCommandExplicitRC4Key256ReturnsActionableError(t *testing.T) {
	oldOPW := opw
	t.Cleanup(func() {
		opw = oldOPW
	})

	cmd := encryptCmd()
	if err := cmd.ParseFlags([]string{
		"--mode", "rc4",
		"--key", "256",
		"--opw", "owner",
		"in.pdf", "out.pdf",
	}); err != nil {
		t.Fatal(err)
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		t.Fatal(err)
	}
	key, err := cmd.Flags().GetString("key")
	if err != nil {
		t.Fatal(err)
	}
	if key != "256" {
		t.Fatalf("got explicit key %q, want 256", key)
	}

	err = validateEncryptModeFlag(&encryptOptions{mode: mode, key: key})
	if err == nil || !strings.Contains(err.Error(), "supported RC4 key lengths: 40,128 default:128") {
		t.Fatalf("expected actionable RC4 key-length error, got %v", err)
	}
}

func TestCryptoCLIPDFArgumentErrorsIncludeContext(t *testing.T) {
	encryptConf := model.NewDefaultConfiguration()
	encryptConf.OwnerPW = "owner"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "encrypt input",
			err: handleEncryptCommand(
				encryptConf,
				[]string{"in.txt"},
				&encryptOptions{mode: "aes", key: "256", perm: "none"},
			),
			want: "encrypt: parse arguments",
		},
		{
			name: "decrypt input",
			err:  handleDecryptCommand(model.NewDefaultConfiguration(), []string{"in.txt"}),
			want: "decrypt: parse arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
		})
	}
}

func TestCryptoCLICommandsAcceptDocumentedArgumentRange(t *testing.T) {
	commands := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "encrypt", cmd: encryptCmd()},
		{name: "decrypt", cmd: decryptCmd()},
	}
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "missing input", wantErr: true},
		{name: "input", args: []string{"in.pdf"}},
		{name: "input and output", args: []string{"in.pdf", "out.pdf"}},
		{name: "extra output", args: []string{"in.pdf", "out.pdf", "extra.pdf"}, wantErr: true},
	}

	for _, command := range commands {
		for _, tt := range tests {
			t.Run(command.name+" "+tt.name, func(t *testing.T) {
				err := command.cmd.Args(command.cmd, tt.args)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error=%v, wantErr=%t", err, tt.wantErr)
				}
			})
		}
	}
}

// TestSecurityCLIHandlersRejectMissingConfigurationAndArguments verifies handler boundaries do not panic.
func TestSecurityCLIHandlersRejectMissingConfigurationAndArguments(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	oldPerm := perm
	t.Cleanup(func() {
		perm = oldPerm
	})
	perm = "none"

	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "list permissions nil configuration",
			run: func() error {
				return handleListPermissionsCommand(nil, []string{"in.pdf"})
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "list permissions missing input",
			run: func() error {
				return handleListPermissionsCommand(conf, nil)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "list permissions empty input",
			run: func() error {
				return handleListPermissionsCommand(conf, []string{""})
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "set permissions nil configuration",
			run: func() error {
				return handleSetPermissionsCommand(nil, []string{"in.pdf"})
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "set permissions missing input",
			run: func() error {
				return handleSetPermissionsCommand(conf, nil)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "set permissions empty input",
			run: func() error {
				return handleSetPermissionsCommand(conf, []string{""})
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "change user password nil configuration",
			run: func() error {
				return handleChangeUserPasswordCommand(nil, []string{"in.pdf", "old", "new"})
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "change user password missing input",
			run: func() error {
				return handleChangeUserPasswordCommand(conf, nil)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "change user password empty input",
			run: func() error {
				return handleChangeUserPasswordCommand(conf, []string{"", "old", "new"})
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "change owner password nil configuration",
			run: func() error {
				return handleChangeOwnerPasswordCommand(nil, []string{"in.pdf", "old", "new"})
			},
			want: api.ErrMissingConfiguration,
		},
		{
			name: "change owner password missing input",
			run: func() error {
				return handleChangeOwnerPasswordCommand(conf, nil)
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "change owner password empty input",
			run: func() error {
				return handleChangeOwnerPasswordCommand(conf, []string{"", "old", "new"})
			},
			want: api.ErrMissingPDFInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

// TestSecurityCLIHandlersRejectInvalidArguments verifies stable operation context for bad CLI input.
func TestSecurityCLIHandlersRejectInvalidArguments(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	oldPerm := perm
	t.Cleanup(func() {
		perm = oldPerm
	})

	tests := []struct {
		name     string
		run      func() error
		want     string
		sentinel error
	}{
		{
			name: "list malformed glob",
			run: func() error {
				return handleListPermissionsCommand(conf, []string{"*["})
			},
			want: "list permissions: expand input pattern",
		},
		{
			name: "list invalid extension",
			run: func() error {
				return handleListPermissionsCommand(conf, []string{"in.txt"})
			},
			want: "list permissions: parse arguments",
		},
		{
			name: "set extra argument",
			run: func() error {
				perm = "none"
				return handleSetPermissionsCommand(conf, []string{"in.pdf", "out.pdf", "extra.pdf"})
			},
			want: "set permissions: expected 1 or 2 arguments",
		},
		{
			name: "set invalid permissions",
			run: func() error {
				perm = "invalid"
				return handleSetPermissionsCommand(conf, []string{"in.pdf"})
			},
			want: "set permissions: validate permissions",
		},
		{
			name: "change user missing passwords",
			run: func() error {
				return handleChangeUserPasswordCommand(conf, []string{"in.pdf"})
			},
			want: "change user password: expected 3 or 4 arguments",
		},
		{
			name: "change user extra argument",
			run: func() error {
				return handleChangeUserPasswordCommand(conf, []string{"in.pdf", "old", "new", "out.pdf", "extra.pdf"})
			},
			want: "change user password: expected 3 or 4 arguments",
		},
		{
			name: "change owner missing passwords",
			run: func() error {
				return handleChangeOwnerPasswordCommand(conf, []string{"in.pdf"})
			},
			want: "change owner password: expected 3 or 4 arguments",
		},
		{
			name: "change owner empty new password",
			run: func() error {
				return handleChangeOwnerPasswordCommand(conf, []string{"in.pdf", "old", ""})
			},
			want:     "change owner password: new owner password must not be empty",
			sentinel: pdfcpu.ErrOwnerPasswordRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, err)
			}
			if tt.sentinel != nil && !errors.Is(err, tt.sentinel) {
				t.Fatalf("got %v, want %v", err, tt.sentinel)
			}
		})
	}
}

// TestSecurityCLIPermissionNormalizationStaysLocal verifies shorthand expansion feeds configuration only.
func TestSecurityCLIPermissionNormalizationStaysLocal(t *testing.T) {
	oldPerm := perm
	t.Cleanup(func() {
		perm = oldPerm
	})

	perm = "p"
	conf := model.NewDefaultConfiguration()
	inFile := filepath.Join(t.TempDir(), "missing.pdf")

	if err := handleSetPermissionsCommand(conf, []string{inFile}); err == nil {
		t.Fatal("expected missing input error")
	}
	if perm != "p" {
		t.Fatalf("CLI permission flag changed to %q", perm)
	}
	if conf.Permissions != model.PermissionsPrint {
		t.Fatalf("got permissions %d, want %d", conf.Permissions, model.PermissionsPrint)
	}
	if isHex("") {
		t.Fatal("empty permission must not be hexadecimal")
	}
}
