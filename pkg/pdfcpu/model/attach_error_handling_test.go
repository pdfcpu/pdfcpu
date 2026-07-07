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

package model

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func attachmentErrorContext() *Context {
	conf := NewDefaultConfiguration()
	xRefTable := newXRefTable(conf)
	xRefTable.Valid = true
	xRefTable.Names["EmbeddedFiles"] = &Node{
		Kmin: "bad id",
		Kmax: "bad id",
		Names: []entry{
			{k: "bad id", v: types.Integer(1)},
		},
	}

	return &Context{
		Configuration: conf,
		XRefTable:     xRefTable,
	}
}

func TestListAttachmentsNilContext(t *testing.T) {
	var ctx *Context

	_, err := ctx.ListAttachments()
	if !errors.Is(err, ErrMissingPDFContext) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
	}
}

func TestListAttachmentsMissingXRefTable(t *testing.T) {
	_, err := (&Context{}).ListAttachments()
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
}

func TestListAttachmentsErrorsIncludeNameTreeAndFileSpecContext(t *testing.T) {
	_, err := attachmentErrorContext().ListAttachments()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"name tree", `file spec "bad id"`, "dereference dict"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestExtractAttachmentsErrorsIncludeNameTreeAndFileSpecContext(t *testing.T) {
	_, err := attachmentErrorContext().ExtractAttachments(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"name tree", `file spec "bad id"`, "dereference dict"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestSearchAttachmentsErrorsIncludeNameTreeAndFileSpecContext(t *testing.T) {
	_, _, err := attachmentErrorContext().SearchEmbeddedFilesNameTreeNodeByContent("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"name tree", `file spec "bad id"`, "dereference dict"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestRemoveAttachmentsWithoutNameTreeIsNoOp(t *testing.T) {
	conf := NewDefaultConfiguration()
	xRefTable := newXRefTable(conf)
	xRefTable.Valid = true

	ok, err := (&Context{Configuration: conf, XRefTable: xRefTable}).RemoveAttachments(nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no attachment removed")
	}
}

func TestFileSpecInfoRejectsMissingEmbeddedFileStream(t *testing.T) {
	conf := NewDefaultConfiguration()
	xRefTable := newXRefTable(conf)
	version := V17
	xRefTable.HeaderVersion = &version
	d := types.Dict{"UF": types.StringLiteral("attachment.txt")}

	_, _, _, _, err := fileSpecStreamDictInfo(xRefTable, "attachment", d, false)
	if err == nil || !strings.Contains(err.Error(), `file spec "attachment": missing embedded file stream`) {
		t.Fatalf("expected missing embedded file stream context, got %v", err)
	}
}

func modDateTestXRefTable() *XRefTable {
	conf := NewDefaultConfiguration()
	xRefTable := newXRefTable(conf)
	version := V17
	xRefTable.HeaderVersion = &version
	return xRefTable
}

func modDateFileSpec(modDate types.Object) types.Dict {
	sd := types.StreamDict{
		Dict: types.Dict{
			"Params": types.Dict{"ModDate": modDate},
		},
	}
	return types.Dict{
		"UF": types.StringLiteral("attachment.txt"),
		"EF": types.Dict{"F": sd},
	}
}

func TestFileSpecModDatePreservesDereferenceCause(t *testing.T) {
	wantErr := errors.New("decode lazy ModDate")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("ModDate")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(_ context.Context, _ string) (types.Object, error) {
		return nil, wantErr
	})

	xRefTable := modDateTestXRefTable()
	xRefTable.Table[7] = NewXRefTableEntryGen0(lazy)
	modDate := *types.NewIndirectRef(7, 0)

	_, _, _, _, err := fileSpecStreamDictInfo(xRefTable, "attachment", modDateFileSpec(modDate), false)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, want := range []string{`file spec "attachment"`, "ModDate", "dereference"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

func TestFileSpecModDatePreservesTextDecodingCause(t *testing.T) {
	sl := types.StringLiteral(string([]byte{0xFE, 0xFF, 0xD8, 0x00}))
	_, cause := types.StringLiteralToString(sl)
	if cause == nil {
		t.Fatal("expected text decoding error")
	}

	_, _, _, _, err := fileSpecStreamDictInfo(modDateTestXRefTable(), "attachment", modDateFileSpec(sl), false)
	if err == nil {
		t.Fatal("expected ModDate error")
	}
	root := err
	for errors.Unwrap(root) != nil {
		root = errors.Unwrap(root)
	}
	if root.Error() != cause.Error() {
		t.Fatalf("expected root cause %q, got %q", cause, root)
	}
	for _, want := range []string{`file spec "attachment"`, "ModDate", "decode text"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

func TestFileSpecModDateUsesStableSemanticErrors(t *testing.T) {
	tests := []struct {
		name    string
		modDate types.Object
		wantErr error
	}{
		{name: "invalid type", modDate: types.Integer(1), wantErr: errInvalidModDateType},
		{name: "invalid date", modDate: types.StringLiteral("not-a-date"), wantErr: errInvalidModDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := fileSpecStreamDictInfo(
				modDateTestXRefTable(),
				"attachment",
				modDateFileSpec(tt.modDate),
				false,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			for _, want := range []string{`file spec "attachment"`, "ModDate"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in %q", want, err)
				}
			}
		})
	}
}

type embeddedStreamFailingReader struct {
	err error
}

func (r embeddedStreamFailingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestAttachmentEmbeddedStreamCopyPreservesCauseAndID(t *testing.T) {
	wantErr := errors.New("copy attachment content")
	a := Attachment{
		Reader: embeddedStreamFailingReader{err: wantErr},
		ID:     "invoice.csv",
	}

	_, err := modDateTestXRefTable().NewFileSpecDictForAttachment(a)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, want := range []string{`attachment "invoice.csv"`, "create embedded stream", "copy content"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

func TestEmbeddedStreamEncodeFailurePreservesCause(t *testing.T) {
	sd := &types.StreamDict{
		Dict:           types.NewDict(),
		Content:        []byte("attachment"),
		FilterPipeline: []types.PDFFilter{{Name: filter.JBIG2}},
	}

	_, err := modDateTestXRefTable().finalizeEmbeddedStreamDict(sd, len(sd.Content), time.Now())

	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "encode stream") {
		t.Fatalf("expected stream encode context in %q", err)
	}
}

func TestEmbeddedStreamRecycledObjectInsertionFailure(t *testing.T) {
	xRefTable := modDateTestXRefTable()
	size := 1
	xRefTable.Size = &size
	xRefTable.Table[0] = NewXRefTableEntryGen0(nil)
	sd, err := xRefTable.NewStreamDictForBuf([]byte("attachment"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = xRefTable.finalizeEmbeddedStreamDict(sd, len(sd.Content), time.Now())

	if err == nil {
		t.Fatal("expected insertion error")
	}
	for _, want := range []string{"insert indirect object", "object #0 found, but not free"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

func TestFileSpecEntriesReportInvalidUnicodeContext(t *testing.T) {
	invalid := string([]byte{0xFF})
	indRef := *types.NewIndirectRef(1, 0)
	tests := []struct {
		name string
		f    string
		uf   string
		desc string
		want string
	}{
		{name: "F", f: invalid, uf: "attachment.txt", want: "file spec F: encode text"},
		{name: "UF", f: "attachment.txt", uf: invalid, want: "file spec UF: encode text"},
		{name: "Desc", f: "attachment.txt", uf: "attachment.txt", desc: invalid, want: "file spec Desc: encode text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := modDateTestXRefTable().NewFileSpecDict(tt.f, tt.uf, tt.desc, indRef)
			if !errors.Is(err, types.ErrInvalidUTF8) {
				t.Fatalf("expected %v, got %v", types.ErrInvalidUTF8, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err)
			}
		})
	}
}
