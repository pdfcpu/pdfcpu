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

package test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func encryptionSecretFiles(t *testing.T) map[string]string {
	t.Helper()
	dir := t.TempDir()
	paths := make(map[string]string)
	for name, value := range map[string]string{
		"user": "file-user-secret\n", "owner": "file-owner-secret\r\n",
		"new-user": "file-new-user-secret", "new-owner": "file-new-owner-secret\n",
		"wrong": "file-wrong-secret", "empty": "",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		paths[name] = path
	}
	return paths
}

func runEncryptionCLI(t *testing.T, wantErr bool, args ...string) []byte {
	t.Helper()
	output, err := runPDFCPU(t, append([]string{"--offline"}, args...)...)
	for _, secret := range []string{"file-user-secret", "file-owner-secret", "file-new-user-secret", "file-new-owner-secret", "file-wrong-secret"} {
		if bytes.Contains(output, []byte(secret)) {
			t.Fatal("command output disclosed a password")
		}
	}
	if (err != nil) != wantErr {
		t.Fatalf("%s: error = %v, want error = %t; output: %s", args[0], err, wantErr, output)
	}
	return output
}

// TestEncryptionPasswordFiles verifies binary encryption, inspection, optimization and decryption with password files.
func TestEncryptionPasswordFiles(t *testing.T) {
	secrets := encryptionSecretFiles(t)
	for _, algorithm := range []struct{ mode, key string }{{"rc4", "128"}, {"aes", "128"}, {"aes", "256"}} {
		t.Run(algorithm.mode+algorithm.key, func(t *testing.T) {
			dir := t.TempDir()
			encrypted := filepath.Join(dir, "encrypted.pdf")
			runEncryptionCLI(t, false, "encrypt", repoFile(t, "pkg", "testdata", "test.pdf"), encrypted,
				"--mode", algorithm.mode, "--key", algorithm.key, "--upw-file", secrets["user"], "--opw-file", secrets["owner"])
			runEncryptionCLI(t, true, "validate", encrypted)
			for _, password := range []struct{ flag, file string }{{"--upw-file", secrets["user"]}, {"--opw-file", secrets["owner"]}} {
				optimized := filepath.Join(dir, password.flag+".pdf")
				runEncryptionCLI(t, false, "validate", encrypted, password.flag, password.file)
				runEncryptionCLI(t, false, "optimize", encrypted, optimized, password.flag, password.file)
				runEncryptionCLI(t, false, "validate", optimized, password.flag, password.file)
				runEncryptionCLI(t, true, "validate", optimized)
				plain := filepath.Join(dir, password.flag+"-plain.pdf")
				runEncryptionCLI(t, false, "decrypt", encrypted, plain, password.flag, password.file)
				runEncryptionCLI(t, false, "validate", plain)
			}
		})
	}
}

// TestEncryptionPasswordFileChanges verifies both file pairs, permission flags and empty user-password replacement.
func TestEncryptionPasswordFileChanges(t *testing.T) {
	secrets := encryptionSecretFiles(t)
	dir := t.TempDir()
	encrypted, userChanged := filepath.Join(dir, "encrypted.pdf"), filepath.Join(dir, "user.pdf")
	ownerChanged, unrestricted := filepath.Join(dir, "owner.pdf"), filepath.Join(dir, "permissions.pdf")
	emptyUser := filepath.Join(dir, "empty-user.pdf")
	runEncryptionCLI(t, false, "encrypt", repoFile(t, "pkg", "testdata", "test.pdf"), encrypted,
		"--upw-file", secrets["user"], "--opw-file", secrets["owner"])
	runEncryptionCLI(t, false, "changeupw", encrypted, "--upwold-file", secrets["user"],
		"--upwnew-file", secrets["new-user"], "--opw-file", secrets["owner"], userChanged)
	runEncryptionCLI(t, true, "validate", userChanged, "--upw-file", secrets["user"])
	runEncryptionCLI(t, false, "validate", userChanged, "--upw-file", secrets["new-user"])
	runEncryptionCLI(t, false, "changeopw", userChanged, "--opwold-file", secrets["owner"],
		"--opwnew-file", secrets["new-owner"], "--upw-file", secrets["new-user"], ownerChanged)
	runEncryptionCLI(t, true, "validate", ownerChanged, "--opw-file", secrets["owner"])
	runEncryptionCLI(t, false, "validate", ownerChanged, "--opw-file", secrets["new-owner"])
	runEncryptionCLI(t, false, "permissions", "set", ownerChanged, unrestricted, "--perm", "all",
		"--opw-file", secrets["new-owner"], "--upw-file", secrets["new-user"])
	output := runEncryptionCLI(t, false, "permissions", "list", unrestricted, "--upw-file", secrets["new-user"])
	if !bytes.Contains(output, []byte("permission bits: 111100111100")) {
		t.Fatalf("unexpected permissions: %s", output)
	}
	runEncryptionCLI(t, false, "changeupw", unrestricted, "--upwold-file", secrets["new-user"],
		"--upwnew-file", secrets["empty"], "--opw-file", secrets["new-owner"], emptyUser)
	runEncryptionCLI(t, false, "validate", emptyUser)
}

// TestEncryptionPasswordFileFailurePreservesOutput verifies wrong secrets cannot replace an existing destination.
func TestEncryptionPasswordFileFailurePreservesOutput(t *testing.T) {
	secrets := encryptionSecretFiles(t)
	dir := t.TempDir()
	encrypted, destination := filepath.Join(dir, "encrypted.pdf"), filepath.Join(dir, "existing.pdf")
	runEncryptionCLI(t, false, "encrypt", repoFile(t, "pkg", "testdata", "test.pdf"), encrypted,
		"--upw-file", secrets["user"], "--opw-file", secrets["owner"])
	original := []byte("existing output must survive")
	if err := os.WriteFile(destination, original, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"decrypt", encrypted, destination, "--upw-file", secrets["wrong"]},
		{"changeupw", encrypted, "--upwold-file", secrets["wrong"], "--upwnew-file", secrets["new-user"], "--opw-file", secrets["owner"], destination},
		{"changeopw", encrypted, "--opwold-file", secrets["wrong"], "--opwnew-file", secrets["new-owner"], "--upw-file", secrets["user"], destination},
	} {
		runEncryptionCLI(t, true, append(args, "--force")...)
		got, err := os.ReadFile(destination)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, original) {
			t.Fatal("failed command replaced the destination")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("staging files left behind: %v", entries)
	}
}
