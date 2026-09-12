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

package api

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	encryptedXRefTitle   = "issue-1467-title"
	encryptedXRefSubject = "issue-1467-subject"
)

type encryptedXRefTestCase struct {
	name      string
	aes       bool
	keyLength int
	userPW    string
}

type encryptedXRefState struct {
	content         []byte
	title           string
	subject         string
	binaryTotalSize int64
	encrypted       bool
}

func encryptedXRefSourcePDF(t *testing.T) ([]byte, []byte) {
	t.Helper()

	content := []byte("q\n0 0 10 10 re\nS\nQ\n")
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 4 0 R /Resources << >> >>"),
		fmt.Appendf(nil, "<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.Bytes()),
		[]byte("<< /Title (issue-1467-title) /Subject <69737375652D313436372D7375626A656374> >>"),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", i+1)
		buf.Write(obj)
		buf.WriteString("\nendobj\n")
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<< /Size %d /Root 1 0 R /Info 5 0 R ", len(offsets))
	buf.WriteString("/ID [<00112233445566778899AABBCCDDEEFF> <00112233445566778899AABBCCDDEEFF>] >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes(), content
}

func encryptedXRefConfig(tc encryptedXRefTestCase) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = tc.userPW
	conf.OwnerPW = "owner"
	conf.EncryptUsingAES = tc.aes
	conf.EncryptKeyLength = tc.keyLength
	conf.Permissions = model.PermissionsNone
	conf.PreserveInfoDict = true
	conf.WriteObjectStream = false
	conf.WriteXRefStream = false
	return conf
}

func encryptedXRefReadConfig(userPW string) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = userPW
	conf.PreserveInfoDict = true
	return conf
}

func corruptEncryptedStartXRef(t *testing.T, bb []byte) []byte {
	t.Helper()

	marker := []byte("startxref\n")
	i := bytes.LastIndex(bb, marker)
	if i < 0 {
		t.Fatal("missing startxref")
	}
	start := i + len(marker)
	end := bytes.IndexByte(bb[start:], '\n')
	if end < 0 {
		t.Fatal("missing startxref offset terminator")
	}
	end += start
	offset, err := strconv.Atoi(string(bb[start:end]))
	if err != nil {
		t.Fatal(err)
	}

	badOffset := fmt.Sprintf("%0*d", end-start, offset-15)
	if len(badOffset) != end-start {
		t.Fatalf("replacement startxref length = %d, want %d", len(badOffset), end-start)
	}
	repaired := bytes.Clone(bb)
	copy(repaired[start:end], badOffset)
	return repaired
}

func encryptedXRefFixture(t *testing.T, tc encryptedXRefTestCase) ([]byte, []byte, []byte) {
	t.Helper()

	source, content := encryptedXRefSourcePDF(t)
	var encrypted bytes.Buffer
	if err := Encrypt(t.Context(), bytes.NewReader(source), &encrypted, encryptedXRefConfig(tc)); err != nil {
		t.Fatal(err)
	}
	clean := encrypted.Bytes()
	return clean, corruptEncryptedStartXRef(t, clean), content
}

func readEncryptedXRefState(t *testing.T, bb []byte, userPW string) encryptedXRefState {
	t.Helper()

	ctx, err := ReadContext(t.Context(), bytes.NewReader(bb), encryptedXRefReadConfig(userPW))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateContext(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	pageDict, _, _, err := ctx.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	content, err := ctx.PageContent(pageDict, 1)
	if err != nil {
		t.Fatal(err)
	}
	return encryptedXRefState{
		content:         content,
		title:           ctx.Title,
		subject:         ctx.Subject,
		binaryTotalSize: ctx.Read.BinaryTotalSize,
		encrypted:       ctx.Encrypt != nil,
	}
}

func assertEncryptedXRefState(t *testing.T, state encryptedXRefState, content []byte, encrypted bool) {
	t.Helper()

	if !bytes.Equal(state.content, content) {
		t.Fatalf("page content = %q, want %q", state.content, content)
	}
	if state.title != encryptedXRefTitle {
		t.Fatalf("Title = %q, want %q", state.title, encryptedXRefTitle)
	}
	if state.subject != encryptedXRefSubject {
		t.Fatalf("Subject = %q, want %q", state.subject, encryptedXRefSubject)
	}
	if state.encrypted != encrypted {
		t.Fatalf("encrypted = %t, want %t", state.encrypted, encrypted)
	}
}

func writeEncryptedXRefFixture(t *testing.T, filename string, bb []byte) {
	t.Helper()
	if err := os.WriteFile(filename, bb, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertEncryptedXRefFile(t *testing.T, filename, userPW string, content []byte, encrypted bool) {
	t.Helper()
	bb, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	assertEncryptedXRefState(t, readEncryptedXRefState(t, bb, userPW), content, encrypted)
}

func assertEncryptedMergeRejected(t *testing.T, filename, outFile string) {
	t.Helper()
	err := MergeCreateFile(t.Context(), []string{filename}, outFile, false, model.NewDefaultConfiguration())
	if !errors.Is(err, pdfcpu.ErrEncrypted) {
		t.Fatalf("got %v, want ErrEncrypted", err)
	}
}

func runEncryptedXRefReconstructionCase(t *testing.T, tc encryptedXRefTestCase) {
	t.Helper()

	clean, repaired, content := encryptedXRefFixture(t, tc)
	cleanState := readEncryptedXRefState(t, clean, tc.userPW)
	repairedState := readEncryptedXRefState(t, repaired, tc.userPW)
	assertEncryptedXRefState(t, cleanState, content, true)
	assertEncryptedXRefState(t, repairedState, content, true)
	if repairedState.binaryTotalSize != cleanState.binaryTotalSize {
		t.Fatalf("repaired binary size = %d, want %d", repairedState.binaryTotalSize, cleanState.binaryTotalSize)
	}

	dir := t.TempDir()
	cleanFile := filepath.Join(dir, "clean.pdf")
	repairedFile := filepath.Join(dir, "repaired.pdf")
	writeEncryptedXRefFixture(t, cleanFile, clean)
	writeEncryptedXRefFixture(t, repairedFile, repaired)

	optimizedFile := filepath.Join(dir, "optimized.pdf")
	if err := OptimizeFile(t.Context(), repairedFile, optimizedFile, encryptedXRefReadConfig(tc.userPW), nil); err != nil {
		t.Fatal(err)
	}
	assertEncryptedXRefFile(t, optimizedFile, tc.userPW, content, true)

	decryptedFile := filepath.Join(dir, "decrypted.pdf")
	if err := DecryptFile(t.Context(), repairedFile, decryptedFile, encryptedXRefReadConfig(tc.userPW)); err != nil {
		t.Fatal(err)
	}
	assertEncryptedXRefFile(t, decryptedFile, "", content, false)
	if err := ValidateFile(t.Context(), decryptedFile, model.NewDefaultConfiguration(), nil); err != nil {
		t.Fatal(err)
	}
	decryptedOptimizedFile := filepath.Join(dir, "decrypted-optimized.pdf")
	if err := OptimizeFile(t.Context(), decryptedFile, decryptedOptimizedFile, model.NewDefaultConfiguration(), nil); err != nil {
		t.Fatal(err)
	}
	assertEncryptedXRefFile(t, decryptedOptimizedFile, "", content, false)

	assertEncryptedMergeRejected(t, cleanFile, filepath.Join(dir, "merge-clean.pdf"))
	assertEncryptedMergeRejected(t, repairedFile, filepath.Join(dir, "merge-repaired.pdf"))

	if tc.userPW != "" {
		conf := encryptedXRefReadConfig("wrong")
		_, err := ReadContext(t.Context(), bytes.NewReader(repaired), conf)
		if !errors.Is(err, pdfcpu.ErrWrongPassword) {
			t.Fatalf("got %v, want ErrWrongPassword", err)
		}
	}
}

// TestEncryptedXRefReconstruction verifies repaired encrypted inputs are decrypted exactly once.
func TestEncryptedXRefReconstruction(t *testing.T) {
	tests := []encryptedXRefTestCase{
		{name: "rc4-40-empty-user-password", keyLength: 40},
		{name: "aes-128-user-password", aes: true, keyLength: 128, userPW: "user"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runEncryptedXRefReconstructionCase(t, tc)
		})
	}
}
