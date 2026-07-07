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
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type encryptionFailingWriter struct {
	err error
}

func (w encryptionFailingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func failingEncryptionDictionaryObject(err error) types.Object {
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("object")},
	}
	return types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, err
	})
}

func newWriteEncryptionTestContext(t *testing.T) *model.Context {
	t.Helper()
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.HeaderVersion = &v
	return ctx
}

// TestSetupEncryptionRejectsUnsupportedAlgorithm verifies stable algorithm errors.
func TestSetupEncryptionRejectsUnsupportedAlgorithm(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.EncryptUsingAES = false
	ctx.EncryptKeyLength = 256

	err := setupEncryption(ctx)
	if !errors.Is(err, ErrUnsupportedEncryptionFeature) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedEncryptionFeature)
	}
	if !strings.Contains(err.Error(), "algorithm configuration") {
		t.Fatalf("missing algorithm context: %v", err)
	}
}

// TestSetupEncryptionRejectsEmptyID verifies empty trailer IDs do not panic.
func TestSetupEncryptionRejectsEmptyID(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.EncryptUsingAES = false
	ctx.EncryptKeyLength = 40
	ctx.ID = types.Array{}

	err := setupEncryption(ctx)
	if !errors.Is(err, errMissingTrailerID) {
		t.Fatalf("got %v, want %v", err, errMissingTrailerID)
	}
	if !strings.Contains(err.Error(), "encryption ID") {
		t.Fatalf("missing encryption ID context: %v", err)
	}
}

// TestHandleEncryptionAddsSetupContext verifies command-level encryption context.
func TestHandleEncryptionAddsSetupContext(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Cmd = model.ENCRYPT
	ctx.EncryptUsingAES = false
	ctx.EncryptKeyLength = 256

	err := handleEncryption(ctx)
	if !errors.Is(err, ErrUnsupportedEncryptionFeature) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedEncryptionFeature)
	}
	if !strings.Contains(err.Error(), "setup encryption") {
		t.Fatalf("missing setup context: %v", err)
	}
}

// TestPrepareContextForWritingAddsEncryptionContext verifies the write boundary.
func TestPrepareContextForWritingAddsEncryptionContext(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	v := model.V20
	ctx.HeaderVersion = &v
	ctx.Cmd = model.ENCRYPT
	ctx.EncryptUsingAES = false
	ctx.EncryptKeyLength = 256

	err := prepareContextForWriting(ctx)
	if !errors.Is(err, ErrUnsupportedEncryptionFeature) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedEncryptionFeature)
	}
	if !strings.Contains(err.Error(), "encryption: setup encryption") {
		t.Fatalf("missing write encryption context: %v", err)
	}
}

// TestWriteEncryptDictRejectsMissingDictionary verifies missing objects are not serialized.
func TestWriteEncryptDictRejectsMissingDictionary(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.EncKey = []byte{1}

	err := writeEncryptDict(ctx)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !errors.Is(err, model.ErrMissingEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, model.ErrMissingEncryptDictObject)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("missing object context: %v", err)
	}
}

// TestWriteEncryptDictClassifiesWrongTypeDictionary verifies both structural and semantic causes survive.
func TestWriteEncryptDictClassifiesWrongTypeDictionary(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.EncKey = []byte{1}
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := writeEncryptDict(ctx)
	if !errors.Is(err, model.ErrWrongTypeEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, model.ErrWrongTypeEncryptDictObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
}

// TestWriteEncryptionPreservesDictionaryReadError verifies operational decode errors are not classified as malformed.
func TestWriteEncryptionPreservesDictionaryReadError(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*model.Context) error
	}{
		{name: "write dictionary", fn: writeEncryptDict},
		{name: "update dictionary", fn: updateEncryption},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newWriteEncryptionTestContext(t)
			readErr := errors.New("read failed")
			ctx.Encrypt = types.NewIndirectRef(7, 0)
			ctx.EncKey = []byte{1}
			ctx.Table[7] = model.NewXRefTableEntryGen0(failingEncryptionDictionaryObject(readErr))

			err := tt.fn(ctx)
			if !errors.Is(err, readErr) {
				t.Fatalf("got %v, want %v", err, readErr)
			}
			if errors.Is(err, ErrMalformedEncryption) {
				t.Fatalf("unexpected malformed encryption classification: %v", err)
			}
			if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
				t.Fatalf("missing encryption object context: %v", err)
			}
		})
	}
}

// TestWriteEncryptDictPreservesWriteFailure verifies writer errors remain discoverable.
func TestWriteEncryptDictPreservesWriteFailure(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.EncKey = []byte{1}
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.NewDict())
	writeErr := errors.New("write failed")
	ctx.Write.Writer = bufio.NewWriterSize(encryptionFailingWriter{writeErr}, 1)

	err := writeEncryptDict(ctx)
	if !errors.Is(err, writeErr) {
		t.Fatalf("got %v, want %v", err, writeErr)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7: write") {
		t.Fatalf("missing write context: %v", err)
	}
}

// TestUpdateEncryptionPreservesNotEncrypted verifies the shared sentinel is returned.
func TestUpdateEncryptionPreservesNotEncrypted(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	if err := updateEncryption(ctx); !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want %v", err, ErrNotEncrypted)
	}
}

// TestUpdateEncryptionRejectsMissingDictionary verifies absent dictionaries do not panic.
func TestUpdateEncryptionRejectsMissingDictionary(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Encrypt = types.NewIndirectRef(7, 0)

	err := updateEncryption(ctx)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !errors.Is(err, model.ErrMissingEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, model.ErrMissingEncryptDictObject)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("missing object context: %v", err)
	}
}

// TestUpdateEncryptionClassifiesWrongTypeDictionary verifies semantic and model causes survive.
func TestUpdateEncryptionClassifiesWrongTypeDictionary(t *testing.T) {
	ctx := newWriteEncryptionTestContext(t)
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := updateEncryption(ctx)
	if !errors.Is(err, model.ErrWrongTypeEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, model.ErrWrongTypeEncryptDictObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
}
