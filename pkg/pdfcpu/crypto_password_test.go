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

package pdfcpu

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func aes256PasswordContext(t *testing.T, revision int, userPW, ownerPW string) (*model.Context, types.Dict) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	ctx, err := model.NewContext(bytes.NewReader(nil), conf)
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.HeaderVersion = &v
	ctx.UserPW, ctx.OwnerPW = userPW, ownerPW
	d := newEncryptDict(false, true, 256, -4)
	d["R"] = types.Integer(revision)
	ctx.E, err = supportedEncryption(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	ctx.E.ID = bytes.Repeat([]byte{1}, 16)
	return ctx, d
}

func checkAES256Password(t *testing.T, ctx *model.Context, owner bool, password string, wantKey []byte) {
	t.Helper()
	ctx.EncKey = nil
	var ok bool
	var err error
	if owner {
		ctx.OwnerPW = password
		ok, err = validateOwnerPassword(ctx)
	} else {
		ctx.UserPW = password
		ok, err = validateUserPassword(ctx)
	}
	if err != nil || !ok {
		t.Fatalf("password rejected: ok=%t err=%v", ok, err)
	}
	if !bytes.Equal(ctx.EncKey, wantKey) {
		t.Fatal("password did not recover the original file encryption key")
	}
}

func testAES256PasswordRoundTrip(t *testing.T, revision int, owner bool, password string) {
	t.Helper()
	userPW, ownerPW := password, "owner"
	if owner {
		userPW, ownerPW = "user", password
	}
	ctx, d := aes256PasswordContext(t, revision, userPW, ownerPW)
	if err := calcOAndU(ctx, d); err != nil {
		t.Fatal(err)
	}
	key := bytes.Clone(ctx.EncKey)
	checkAES256Password(t, ctx, owner, password, key)
}

// TestAES256PasswordWriteReadSymmetric checks passwords changed by normalization or the 127-byte limit.
func TestAES256PasswordWriteReadSymmetric(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "NFKC", password: "\ufb01le"},
		{name: "127-byte limit", password: strings.Repeat("x", 130)},
	}
	for _, revision := range []int{5, 6} {
		for _, owner := range []bool{false, true} {
			for _, tt := range tests {
				t.Run(fmt.Sprintf("R%d/owner=%t/%s", revision, owner, tt.name), func(t *testing.T) {
					testAES256PasswordRoundTrip(t, revision, owner, tt.password)
				})
			}
		}
	}
}

// TestAES256PasswordSASLprepAcceptsSpacesAndSymbols checks passwords rejected by the former PRECIS profile.
func TestAES256PasswordSASLprepAcceptsSpacesAndSymbols(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "space", password: "correct horse"},
		{name: "symbol", password: "\u20acuro"},
	}
	for _, revision := range []int{5, 6} {
		for _, owner := range []bool{false, true} {
			for _, tt := range tests {
				t.Run(fmt.Sprintf("R%d/owner=%t/%s", revision, owner, tt.name), func(t *testing.T) {
					testAES256PasswordRoundTrip(t, revision, owner, tt.password)
				})
			}
		}
	}
}
