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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestAES256OwnerPasswordWithPaddedU checks that excess U bytes do not affect owner authentication.
func TestAES256OwnerPasswordWithPaddedU(t *testing.T) {
	for _, revision := range []int{5, 6} {
		for _, extraBytes := range []int{1, 79} {
			t.Run(fmt.Sprintf("R%d/U=%d", revision, 48+extraBytes), func(t *testing.T) {
				ctx, d := aes256PasswordContext(t, revision, "user", "owner")
				if err := calcOAndU(ctx, d); err != nil {
					t.Fatal(err)
				}
				wantKey := bytes.Clone(ctx.EncKey)
				paddedU := append(bytes.Clone(ctx.E.U), bytes.Repeat([]byte{0xa5}, extraBytes)...)
				d.Update("U", types.NewHexLiteral(paddedU))
				var err error
				ctx.E, err = supportedEncryption(ctx, d)
				if err != nil {
					t.Fatal(err)
				}
				checkAES256Password(t, ctx, false, "user", wantKey)
				checkAES256Password(t, ctx, true, "owner", wantKey)
			})
		}
	}
}
