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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func secretFile(t *testing.T, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestPasswordFileBytes verifies that only one terminal line ending is removed.
func TestPasswordFileBytes(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"", ""}, {"\n", ""}, {"\r\n", ""}, {"secret", "secret"},
		{" secret \n", " secret "}, {"secret\r\n", "secret"},
		{"secret\n\n", "secret\n"}, {"secret\r", "secret\r"},
		{"a\nb", "a\nb"}, {"秘密\n", "秘密"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			cmd := decryptCmd()
			if err := cmd.Flags().Set("upw-file", secretFile(t, tt.input)); err != nil {
				t.Fatal(err)
			}
			got, err := readPasswordFile(cmd, "upw-file")
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func runSecretCommand(t *testing.T, command *cobra.Command, args ...string) (string, error) {
	t.Helper()
	previousConf, previousUpw, previousOpw := conf, upw, opw
	t.Cleanup(func() { conf, upw, opw = previousConf, previousUpw, previousOpw })
	conf, upw, opw = "disable", "", ""
	root := &cobra.Command{Use: "pdfcpu", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(command)
	root.SetArgs(append([]string{command.Name()}, args...))
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	err := root.ExecuteContext(t.Context())
	return output.String(), err
}

// TestPasswordFileCommandFailures verifies conflicts, required inputs and secret-safe failures before PDF processing.
func TestPasswordFileCommandFailures(t *testing.T) {
	secret := "distinctive-password-value"
	path := secretFile(t, secret)
	for _, tt := range []struct {
		name    string
		command func() *cobra.Command
		args    []string
		want    string
	}{
		{"literal conflict", decryptCmd, []string{"--upw", secret, "--upw-file", path, "missing.pdf"}, "upw"},
		{"empty literal conflict", decryptCmd, []string{"--upw=", "--upw-file", path, "missing.pdf"}, "upw"},
		{"missing file", decryptCmd, []string{"--upw-file", path + "-missing", "missing.pdf"}, "upw-file"},
		{"empty path", decryptCmd, []string{"--upw-file=", "missing.pdf"}, "filesystem path"},
		{"stdin path", decryptCmd, []string{"--upw-file=-", "missing.pdf"}, "filesystem path"},
		{"directory", decryptCmd, []string{"--upw-file", t.TempDir(), "missing.pdf"}, "upw-file"},
		{"incomplete pair", changeupwCmd, []string{"--upwold-file", path, "missing.pdf"}, "together"},
		{"mixed positional", changeopwCmd, []string{"--opwold-file", path, "--opwnew-file", path, "missing.pdf", secret, secret}, "without positional passwords"},
		{"empty owner", encryptCmd, []string{"--opw-file", secretFile(t, "\n"), "missing.pdf"}, "owner password must not be empty"},
		{"empty new owner", changeopwCmd, []string{"--opwold-file", path, "--opwnew-file", secretFile(t, ""), "missing.pdf"}, "new owner password must not be empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runSecretCommand(t, tt.command(), tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %s", err, tt.want)
			}
			if strings.Contains(output+err.Error(), secret) {
				t.Fatal("secret disclosed")
			}
		})
	}
}

// TestPasswordFileRoundTrip verifies encryption, inherited flags, both password changes and literal compatibility.
func TestPasswordFileRoundTrip(t *testing.T) {
	input := copyKeywordCommandTestPDF(t)
	dir := t.TempDir()
	encrypted := filepath.Join(dir, "encrypted.pdf")
	changedUser := filepath.Join(dir, "user.pdf")
	changedOwner := filepath.Join(dir, "owner.pdf")
	emptyUser := filepath.Join(dir, "empty.pdf")
	decrypted := filepath.Join(dir, "plain.pdf")
	owner := secretFile(t, "owner-secret\r\n")
	user := secretFile(t, "user-secret\n")
	newUser := secretFile(t, "new-user-secret")
	newOwner := secretFile(t, "new-owner-secret")
	for _, tt := range []struct {
		command func() *cobra.Command
		args    []string
	}{
		{encryptCmd, []string{"--opw-file", owner, "--upw-file", user, input, encrypted}},
		{permissionsCmd, []string{"list", "--upw-file", user, encrypted}},
		{changeupwCmd, []string{"--opw-file", owner, "--upwold-file", user, "--upwnew-file", newUser, encrypted, changedUser}},
		{changeopwCmd, []string{"--upw-file", newUser, "--opwold-file", owner, "--opwnew-file", newOwner, changedUser, changedOwner}},
		{changeupwCmd, []string{"--opw", "new-owner-secret", changedOwner, "new-user-secret", "", emptyUser}},
		{decryptCmd, []string{"--opw-file", newOwner, emptyUser, decrypted}},
	} {
		if _, err := runSecretCommand(t, tt.command(), tt.args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(decrypted); err != nil {
		t.Fatal(err)
	}
}

// TestPasswordChangeFileForms rejects incomplete pairs, positional mixing and unrelated flags for both commands.
func TestPasswordChangeFileForms(t *testing.T) {
	secret := "password-form-secret"
	path := secretFile(t, secret)
	for _, factory := range []func() *cobra.Command{changeupwCmd, changeopwCmd} {
		cmd := factory()
		prefix := strings.TrimPrefix(cmd.Name(), "change")
		oldFlag, newFlag := "--"+prefix+"old-file", "--"+prefix+"new-file"
		other := "opw"
		if prefix == "opw" {
			other = "upw"
		}
		cases := []struct {
			name string
			args []string
			want string
		}{
			{"missing old", []string{newFlag, path, "missing.pdf"}, "together"},
			{"missing new", []string{oldFlag, path, "missing.pdf"}, "together"},
			{"positional mixing", []string{oldFlag, path, newFlag, path, "missing.pdf", secret, secret}, "without positional passwords"},
			{"other password type", []string{"--" + other + "old-file", path, "missing.pdf"}, "unknown flag"},
		}
		for _, flag := range []string{"--oldpw", "--newpw", "--oldpw-file", "--newpw-file"} {
			cases = append(cases, struct {
				name string
				args []string
				want string
			}{flag, []string{flag, secret, "missing.pdf"}, "unknown flag"})
		}
		for _, tt := range cases {
			t.Run(cmd.Name()+"/"+tt.name, func(t *testing.T) {
				output, err := runSecretCommand(t, factory(), tt.args...)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("got %v; want %s", err, tt.want)
				}
				if strings.Contains(output+err.Error(), secret) {
					t.Fatal("secret disclosed")
				}
			})
		}
	}
}

// TestOwnerPasswordPositionalCompatibility verifies the original owner-password arguments still work.
func TestOwnerPasswordPositionalCompatibility(t *testing.T) {
	input := copyKeywordCommandTestPDF(t)
	dir := t.TempDir()
	encrypted, changed := filepath.Join(dir, "encrypted.pdf"), filepath.Join(dir, "changed.pdf")
	if _, err := runSecretCommand(t, encryptCmd(), "--opw", "owner", input, encrypted); err != nil {
		t.Fatal(err)
	}
	if _, err := runSecretCommand(t, changeopwCmd(), encrypted, "owner", "new-owner", changed); err != nil {
		t.Fatal(err)
	}
	if _, err := runSecretCommand(t, decryptCmd(), "--opw", "new-owner", changed, filepath.Join(dir, "plain.pdf")); err != nil {
		t.Fatal(err)
	}
}
