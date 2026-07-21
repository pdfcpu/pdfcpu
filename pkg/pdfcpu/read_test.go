/*
Copyright 2024 The pdfcpu Authors.

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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type headerErrorReader struct {
	seekErr error
	readErr error
}

func (r headerErrorReader) Read(_ []byte) (int, error) {
	return 0, r.readErr
}

func (r headerErrorReader) Seek(_ int64, _ int) (int64, error) {
	return 0, r.seekErr
}

func intPtr(i int) *int {
	return &i
}

type wrappedEOFReader struct {
	data []byte
}

func (r *wrappedEOFReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, fmt.Errorf("wrapped eof: %w", io.EOF)
}

// TestReadFileContext verifies context-aware file reading.
func TestReadFileContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadFileWithContext(ctx, inFile, nil); err == nil {
		t.Errorf("reading should have failed, got %v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

// TestReadContext verifies context-aware reading.
func TestReadContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	fp, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadWithContext(ctx, fp, nil); err == nil {
		t.Errorf("reading should have failed, got %v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

func TestReadClassifiesEmptyInput(t *testing.T) {
	_, err := Read(bytes.NewReader(nil), nil)
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("got %v, want ErrEmptyInput", err)
	}
}

func TestReadMissingReaderReturnsError(t *testing.T) {
	_, err := Read(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMissingXRefSectionClassificationAcceptsWrappedSentinel(t *testing.T) {
	err := fmt.Errorf("scan trailer: %w", ErrMissingXRefSection)
	if !isMissingXRefSection(err) {
		t.Fatalf("expected wrapped %v to classify as missing xref section", ErrMissingXRefSection)
	}
	if isMissingXRefSection(errors.New("not xref")) {
		t.Fatal("unexpected missing xref section classification")
	}
}

func TestOffsetLastXRefSectionClassifiesEpilogueErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{
			name:    "missing EOF",
			in:      "%PDF-1.7\nstartxref\n12\n",
			wantErr: errMissingXRefEOF,
		},
		{
			name:    "invalid offset",
			in:      "%PDF-1.7\nstartxref\nnot-an-offset\n%%EOF",
			wantErr: errInvalidLastXRefSection,
		},
		{
			name:    "missing startxref",
			in:      "%PDF-1.7\n%%EOF",
			wantErr: ErrMissingXRefSection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader([]byte(tt.in)), nil)
			if err != nil {
				t.Fatal(err)
			}

			_, err = offsetLastXRefSection(ctx, 0)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOffsetLastXRefSectionPreservesReadError(t *testing.T) {
	readErr := errors.New("read failed")
	ctx := &model.Context{
		Read: &model.ReadContext{
			RS:       headerErrorReader{readErr: readErr},
			FileSize: 10,
		},
	}

	_, err := offsetLastXRefSection(ctx, 0)
	if !errors.Is(err, readErr) {
		t.Fatalf("expected %v, got %v", readErr, err)
	}
	if !strings.Contains(err.Error(), "scan for startxref") {
		t.Fatalf("expected startxref scan context, got %q", err.Error())
	}
}

func TestDecodeSubsectionClassifiesCorruptEntryType(t *testing.T) {
	_, _, _, err := decodeSubsection([]string{"0000000000", "65535", "x"})
	if !errors.Is(err, errCorruptXRefSubsection) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefSubsection)
	}
}

func TestParseXRefTableSubSectionClassifiesIncompleteSubsection(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("trailer\n"))

	_, err := parseXRefTableSubSection(xRefTable, s, []string{"0", "1"}, 0, 0)
	if !errors.Is(err, errIncompleteXRefSubsection) {
		t.Fatalf("got %v, want %v", err, errIncompleteXRefSubsection)
	}
}

func TestScanLineRawClassifiesMissingLine(t *testing.T) {
	_, err := scanLineRaw(bufio.NewScanner(strings.NewReader("")))
	if !errors.Is(err, errMissingScannerLine) {
		t.Fatalf("got %v, want %v", err, errMissingScannerLine)
	}
}

func TestParseTrailerClassifiesMissingEntries(t *testing.T) {
	tests := []struct {
		name    string
		table   *model.XRefTable
		d       types.Dict
		wantErr error
	}{
		{
			name:    "missing size",
			table:   &model.XRefTable{},
			d:       types.Dict{},
			wantErr: errMissingTrailerSize,
		},
		{
			name:    "missing root",
			table:   &model.XRefTable{Size: intPtr(1)},
			d:       types.Dict{},
			wantErr: errMissingTrailerRoot,
		},
		{
			name: "invalid id",
			table: &model.XRefTable{
				Size:           intPtr(1),
				Root:           types.NewIndirectRef(1, 0),
				ValidationMode: model.ValidationStrict,
			},
			d:       types.Dict{"ID": types.Array{}},
			wantErr: errInvalidTrailerID,
		},
		{
			name: "missing id",
			table: &model.XRefTable{
				Size:    intPtr(1),
				Root:    types.NewIndirectRef(1, 0),
				Encrypt: types.NewIndirectRef(2, 0),
			},
			d:       types.Dict{},
			wantErr: errMissingTrailerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseTrailer(tt.table, tt.d); !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestXRefStreamDictErrorsIncludeObjectContext(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = xRefStreamDict(context.Background(), ctx, types.Integer(1), 7, 0)
	if !errors.Is(err, errMissingXRefStreamDict) {
		t.Fatalf("got %v, want %v", err, errMissingXRefStreamDict)
	}
	if !strings.Contains(err.Error(), "xref stream obj#7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}

	_, err = xRefStreamDict(context.Background(), ctx, types.Dict{}, 8, 0)
	if !errors.Is(err, errMissingXRefStreamLength) {
		t.Fatalf("got %v, want %v", err, errMissingXRefStreamLength)
	}
	if !strings.Contains(err.Error(), "xref stream obj#8") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

func TestParseXRefStreamClassifiesCorruptStreamObject(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	offset := int64(0)

	_, err = parseXRefStream(context.Background(), ctx, strings.NewReader("1 0 obj\n<<>>\nendobj\n"), &offset, 0, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
	if !strings.Contains(err.Error(), "xref stream object") {
		t.Fatalf("expected xref stream object context, got %q", err.Error())
	}
}

func TestDereferencedObjectClassifiesUnregisteredObject(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dereferencedObject(context.Background(), ctx, 7)
	if !errors.Is(err, errUnregisteredObject) {
		t.Fatalf("got %v, want %v", err, errUnregisteredObject)
	}
	if !strings.Contains(err.Error(), "object 7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

func TestRegisterXRefObjectsTracksEOLOffsets(t *testing.T) {
	for _, eol := range []string{"\n", "\r\n"} {
		t.Run(fmt.Sprintf("EOL=%q", eol), func(t *testing.T) {
			pdf := []byte("%PDF-1.7" + eol + "%\x80\x81\x82\x83" + eol + "7 0 obj" + eol + "<<>>" + eol + "endobj" + eol)
			ctx, err := model.NewContext(bytes.NewReader(pdf), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := registerXRefObjects(ctx, 1); err != nil {
				t.Fatal(err)
			}
			entry, ok := ctx.Find(7)
			if !ok || *entry.Offset != int64(bytes.Index(pdf, []byte("7 0 obj"))) {
				t.Fatalf("unexpected xref entry: %+v", entry)
			}
		})
	}
}

func TestDereferencedTypedObjectClassification(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Name("NotInteger"))
	ctx.Table[8] = model.NewXRefTableEntryGen0(types.Integer(1))

	_, err = dereferencedInteger(context.Background(), ctx, 7)
	if !errors.Is(err, errCorruptIntegerObject) {
		t.Fatalf("got %v, want %v", err, errCorruptIntegerObject)
	}
	if !strings.Contains(err.Error(), "object 7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}

	_, err = dereferencedDict(context.Background(), ctx, 8)
	if !errors.Is(err, errCorruptDictObject) {
		t.Fatalf("got %v, want %v", err, errCorruptDictObject)
	}
	if !strings.Contains(err.Error(), "object 8") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

func TestDecompressXRefTableEntryClassifiesObjectStreamErrors(t *testing.T) {
	objStreamNr := 9
	objStreamInd := 0
	tests := []struct {
		name    string
		table   map[int]*model.XRefTableEntry
		wantErr error
	}{
		{
			name:    "missing xref entry",
			table:   map[int]*model.XRefTableEntry{},
			wantErr: errMissingObjectStreamEntry,
		},
		{
			name: "missing object stream",
			table: map[int]*model.XRefTableEntry{
				objStreamNr: model.NewXRefTableEntryGen0(types.Dict{}),
			},
			wantErr: errMissingObjectStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := &model.XRefTable{Table: tt.table}
			entry := &model.XRefTableEntry{
				Compressed:      true,
				ObjectStream:    &objStreamNr,
				ObjectStreamInd: &objStreamInd,
			}

			err := decompressXRefTableEntry(xRefTable, 7, entry)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), "object stream 9") {
				t.Fatalf("expected object stream context, got %q", err.Error())
			}
		})
	}
}

func TestCompressedObjectRejectsStreamObjects(t *testing.T) {
	_, err := compressedObject(context.Background(), "<< /Length 1 >>")
	if !errors.Is(err, errObjectStreamContainsStream) {
		t.Fatalf("got %v, want %v", err, errObjectStreamContainsStream)
	}
}

func TestParseObjectStreamClassifiesCorruptDict(t *testing.T) {
	osd := &types.ObjectStreamDict{
		StreamDict:     types.StreamDict{Content: []byte("1")},
		FirstObjOffset: 1,
	}

	err := parseObjectStream(context.Background(), osd, model.DefaultResourceLimits())
	if !errors.Is(err, errCorruptObjectStreamDict) {
		t.Fatalf("got %v, want %v", err, errCorruptObjectStreamDict)
	}
}

func TestParseObjectStreamClassifiesObjectCountLimit(t *testing.T) {
	content := []byte("1 0 2 1")
	osd := &types.ObjectStreamDict{
		StreamDict:     types.StreamDict{Content: content},
		FirstObjOffset: len(content),
	}
	limits := model.DefaultResourceLimits()
	limits.MaxObjectStreamCount = 1

	err := parseObjectStream(context.Background(), osd, limits)
	if !errors.Is(err, errObjectStreamObjectCountLimit) {
		t.Fatalf("got %v, want %v", err, errObjectStreamObjectCountLimit)
	}
}

func TestDecodeObjectStreamClassifiesMissingEntry(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = decodeObjectStream(context.Background(), ctx, 7)
	if !errors.Is(err, errMissingObjectStreamEntry) {
		t.Fatalf("got %v, want %v", err, errMissingObjectStreamEntry)
	}
	if !strings.Contains(err.Error(), "object stream obj#7") {
		t.Fatalf("expected object stream context, got %q", err.Error())
	}
}

func TestDecodeObjectStreamClassifiesCorruptObjectStream(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader([]byte("7 0 obj\n1\nendobj\n")), nil)
	if err != nil {
		t.Fatal(err)
	}
	zeroOffset := int64(0)
	zeroGen := 0
	ctx.Table[7] = &model.XRefTableEntry{
		Offset:     &zeroOffset,
		Generation: &zeroGen,
	}

	err = decodeObjectStream(context.Background(), ctx, 7)
	if !errors.Is(err, errCorruptObjectStream) {
		t.Fatalf("got %v, want %v", err, errCorruptObjectStream)
	}
	if !strings.Contains(err.Error(), "object stream obj#7") {
		t.Fatalf("expected object stream context, got %q", err.Error())
	}
}

func TestHeaderVersionRejectsPostScript(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"Plain", "%!PS-Adobe-3.0\n%%Title: report.txt\n"},
		{"BOMAndWhitespace", "\xef\xbb\xbf \r\n%!PS-Adobe-3.0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := headerVersion(strings.NewReader(tt.in))
			if !errors.Is(err, ErrPostScriptInput) {
				t.Fatalf("got %v, want ErrPostScriptInput", err)
			}
		})
	}
}

func TestHeaderVersionClassifiesCorruptHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "NoPDFHeader", in: "not a pdf\n"},
		{name: "MissingEOL", in: "%PDF-1.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := headerVersion(strings.NewReader(tt.in))
			if !errors.Is(err, ErrCorruptHeader) {
				t.Fatalf("got %v, want ErrCorruptHeader", err)
			}
		})
	}
}

func TestHeaderVersionUnknownVersionIncludesContext(t *testing.T) {
	_, _, _, err := headerVersion(strings.NewReader("%PDF-9.9\n% padding for header scan\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown PDF header version") {
		t.Fatalf("expected unknown version context, got %q", err.Error())
	}
}

func TestHeaderVersionPreservesIOErrors(t *testing.T) {
	seekErr := errors.New("seek failed")
	_, _, _, err := headerVersion(headerErrorReader{seekErr: seekErr})
	if !errors.Is(err, seekErr) {
		t.Fatalf("expected %v, got %v", seekErr, err)
	}
	if !strings.Contains(err.Error(), "seek header") {
		t.Fatalf("expected seek context, got %q", err.Error())
	}

	readErr := errors.New("read failed")
	_, _, _, err = headerVersion(headerErrorReader{readErr: readErr})
	if !errors.Is(err, readErr) {
		t.Fatalf("expected %v, got %v", readErr, err)
	}
	if !strings.Contains(err.Error(), "read header") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

func TestHandleUnencryptedFileClassifiesErrors(t *testing.T) {
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}

	ctx.Cmd = model.DECRYPT
	if err := handleUnencryptedFile(ctx); !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want ErrNotEncrypted", err)
	}

	ctx.Cmd = model.ENCRYPT
	if err := handleUnencryptedFile(ctx); !errors.Is(err, ErrOwnerPasswordRequired) {
		t.Fatalf("got %v, want ErrOwnerPasswordRequired", err)
	}
}

func TestOwnerPasswordRequiredWithOPWPreservesSentinel(t *testing.T) {
	err := fmt.Errorf("%w with --opw", ErrOwnerPasswordRequired)
	if !errors.Is(err, ErrOwnerPasswordRequired) {
		t.Fatalf("got %v, want ErrOwnerPasswordRequired", err)
	}
}

func TestCheckForEncryptionClassifiesEncryptedError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.ENCRYPT
	ctx.Encrypt = types.NewIndirectRef(1, 0)

	if err := checkForEncryption(context.Background(), ctx); !errors.Is(err, ErrEncrypted) {
		t.Fatalf("got %v, want ErrEncrypted", err)
	} else if !strings.Contains(err.Error(), "encryption status") {
		t.Fatalf("expected encryption status context, got %q", err)
	}
}

func TestCheckForEncryptionClassifiesUnencryptedError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.DECRYPT

	err = checkForEncryption(context.Background(), ctx)
	if !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want ErrNotEncrypted", err)
	}
	if !strings.Contains(err.Error(), "encryption status") {
		t.Fatalf("expected encryption status context, got %q", err)
	}
}

func TestCheckForEncryptionRejectsIncompleteContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  *model.Context
		want error
	}{
		{name: "missing context", want: ErrMissingPDFContext},
		{
			name: "missing xref table",
			ctx:  &model.Context{Configuration: model.NewDefaultConfiguration()},
			want: ErrMissingXRefTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkForEncryption(context.Background(), tt.ctx); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCheckForEncryptionWrapsEncryptionDictionaryError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Encrypt = types.NewIndirectRef(7, 0)

	err = checkForEncryption(context.Background(), ctx)
	if !errors.Is(err, errUnregisteredObject) {
		t.Fatalf("got %v, want %v", err, errUnregisteredObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("expected encryption dictionary context, got %q", err.Error())
	}
}

// TestCheckForEncryptionClassifiesWrongTypeDictionary verifies both structural and semantic causes survive.
func TestCheckForEncryptionClassifiesWrongTypeDictionary(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Integer(1))

	err = checkForEncryption(context.Background(), ctx)
	if !errors.Is(err, errCorruptDictObject) {
		t.Fatalf("got %v, want %v", err, errCorruptDictObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
}

// TestCheckForEncryptionPreservesDictionaryReadError verifies operational I/O errors are not classified as malformed.
func TestCheckForEncryptionPreservesDictionaryReadError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	readErr := errors.New("read failed")
	ctx.Read.RS = headerErrorReader{readErr: readErr}
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	entry := model.NewXRefTableEntryGen0(nil)
	offset := int64(0)
	entry.Offset = &offset
	ctx.Table[7] = entry

	err = checkForEncryption(context.Background(), ctx)
	if !errors.Is(err, readErr) {
		t.Fatalf("got %v, want %v", err, readErr)
	}
	if errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("unexpected malformed encryption classification: %v", err)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("missing encryption object context: %v", err)
	}
}

// TestCheckForEncryptionClassifiesUnterminatedDictionary verifies parser and encryption causes survive.
func TestCheckForEncryptionClassifiesUnterminatedDictionary(t *testing.T) {
	rs := bytes.NewReader([]byte("7 0 obj\n<< /Filter /Standard\nendobj\n"))
	ctx, err := model.NewContext(rs, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	entry := model.NewXRefTableEntryGen0(nil)
	offset := int64(0)
	entry.Offset = &offset
	ctx.Table[7] = entry

	err = checkForEncryption(context.Background(), ctx)
	if !errors.Is(err, model.ErrDictionaryCorrupt) {
		t.Fatalf("got %v, want %v", err, model.ErrDictionaryCorrupt)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#") {
		t.Fatalf("missing encryption dictionary object context: %v", err)
	}
}

func TestSetupEncryptionKeyRejectsMissingID(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	setStrictPDF17XRefTable(ctx)

	err = setupEncryptionKey(ctx, newEncryptDict(false, false, 40, 0))
	if !errors.Is(err, errMissingTrailerID) {
		t.Fatalf("got %v, want %v", err, errMissingTrailerID)
	}
	if !strings.Contains(err.Error(), "read encryption ID") {
		t.Fatalf("expected encryption ID context, got %q", err)
	}
}

func TestSetupEncryptionKeyPreservesAuthenticationSentinels(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.CommandMode
		want error
	}{
		{name: "wrong password", cmd: model.DECRYPT, want: ErrWrongPassword},
		{name: "owner password required", cmd: model.CHANGEOPW, want: ErrOwnerPasswordRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader(nil), nil)
			if err != nil {
				t.Fatal(err)
			}
			setStrictPDF17XRefTable(ctx)
			id := types.NewHexLiteral([]byte("id"))
			ctx.ID = types.Array{id, id}
			ctx.Cmd = tt.cmd

			err = setupEncryptionKey(ctx, newEncryptDict(false, false, 40, 0))
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHandlePermissionsClassifiesPermissionDenied(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.ADDATTACHMENTS
	ctx.OwnerPW = "opw"
	ctx.E = &model.Enc{R: 4}

	if err := handlePermissions(ctx); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("got %v, want ErrPermissionDenied", err)
	}
}

func TestHandlePermissionsClassifiesInvalidPermissions(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.E = &model.Enc{R: 5, Perms: make([]byte, 16)}
	ctx.EncKey = make([]byte, 16)

	err = handlePermissions(ctx)
	if !errors.Is(err, errInvalidPermissions) {
		t.Fatalf("got %v, want %v", err, errInvalidPermissions)
	}
	if !strings.Contains(err.Error(), "user password permissions") {
		t.Fatalf("expected user password permissions context, got %q", err.Error())
	}
}

func TestExtractXRefStreamEntriesDefaultType(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	const objCount = 67
	xsd := &types.XRefStreamDict{
		Objects: make([]int, objCount),
		W:       [3]int{0, 3, 0},
	}
	buf := make([]byte, 0, objCount*3)
	offsets := make([]int64, objCount)
	for objNr := 0; objNr < objCount; objNr++ {
		xsd.Objects[objNr] = objNr
		offset := int64(1 + objNr*257)
		offsets[objNr] = offset
		buf = append(buf, byte(offset>>16), byte(offset>>8), byte(offset))
	}

	if err := extractXRefTableEntriesFromXRefStream(buf, 0, xsd, ctx, 0); err != nil {
		t.Fatal(err)
	}
	if len(ctx.Table) != objCount {
		t.Fatalf("got %d xref entries, want %d", len(ctx.Table), objCount)
	}
	for objNr, wantOffset := range offsets {
		entry := ctx.Table[objNr]
		if entry == nil {
			t.Fatalf("missing xref entry for object %d", objNr)
		}
		if entry.Free || entry.Compressed {
			t.Fatalf("object %d: expected an in-use xref entry", objNr)
		}
		if entry.Offset == nil || *entry.Offset != wantOffset {
			t.Fatalf("object %d: got offset %v, want %d", objNr, entry.Offset, wantOffset)
		}
	}
}

func TestExtractXRefStreamEntriesRejectsZeroWidths(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{W: [3]int{0, 0, 0}}
	err = extractXRefTableEntriesFromXRefStream(nil, 0, xsd, ctx, 0)
	if !errors.Is(err, errInvalidXRefStreamWArray) {
		t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
	}
}

func TestExtractXRefStreamEntriesClassifiesCorruptStream(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{
		Objects: []int{0},
		W:       [3]int{1, 1, 1},
	}
	err = extractXRefTableEntriesFromXRefStream([]byte{1, 2}, 0, xsd, ctx, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
}

// TestReadLargeDictObject verifies large dictionary objects are handled.
func TestReadLargeDictObject(t *testing.T) {
	// Test with "stream" and "endobj" inside the dictionary.
	var fp bytes.Buffer
	fp.WriteString("123 0 obj\n")
	data := make([]byte, 10*1024*1024)
	fp.WriteString("<<")
	fp.WriteString("/Foo <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString("/Bar (stream)\n")
	fp.WriteString("/Baz (endobj)\n")
	fp.WriteString("/Test <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString(">>\n")
	fp.WriteString("stream\n")
	fp.WriteString("Hello world!\n")
	fp.WriteString("endstream\n")
	fp.WriteString("endobj\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObjectWithContext(ctx, c, 0, 123, 0)
	if err != nil {
		t.Fatal(err)
	}

	d, ok := o.(types.StreamDict)
	if !ok {
		t.Fatalf("expected StreamDict, got %T", o)
	}

	if err := loadEncodedStreamContent(ctx, c, &d, true); err != nil {
		t.Fatal(err)
	}

	if foo := d.HexLiteralEntry("Foo"); foo == nil {
		t.Error("expected Foo entry")
	} else if expected := hex.EncodeToString(data); foo.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(foo.Value()))
	}

	if bar := d.StringEntry("Bar"); bar == nil {
		t.Error("expected Bar entry")
	} else if expected := "stream"; *bar != expected {
		t.Errorf("expected %s for Bar, got %s", expected, *bar)
	}

	if baz := d.StringEntry("Baz"); baz == nil {
		t.Error("expected Baz entry")
	} else if expected := "endobj"; *baz != expected {
		t.Errorf("expected %s for Baz, got %s", expected, *baz)
	}

	if err := d.Decode(); err != nil {
		t.Fatal(err)
	}

	if expected := "Hello world!"; string(d.Content) != expected {
		t.Errorf("expected stream content %s, got %s", expected, string(d.Content))
	}
}

// TestReadStreamContentRejectsStreamLimit verifies stream limits are enforced.
func TestReadStreamContentRejectsStreamLimit(t *testing.T) {
	_, err := readStreamContent(bytes.NewReader([]byte("abc")), 3, 2)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("got %v, want stream limit error", err)
	}
}

func TestReadStreamContentAcceptsWrappedEOF(t *testing.T) {
	rd := &wrappedEOFReader{data: []byte("abc\nendstream\n")}

	buf, err := readStreamContent(rd, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != "abc\n" {
		t.Fatalf("got %q, want %q", string(buf), "abc\n")
	}
}

// TestEnsureStreamLengthRejectsNilDict verifies malformed stream dicts do not panic.
func TestEnsureStreamLengthRejectsNilDict(t *testing.T) {
	sd := &types.StreamDict{
		Raw: []byte("abc"),
	}

	err := ensureStreamLength(sd, true)
	if !errors.Is(err, errCorruptStreamDict) {
		t.Fatalf("got %v, want %v", err, errCorruptStreamDict)
	}
}

func TestEnsureIndirectStreamLengthClassifiesMissingLength(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	sd := &types.StreamDict{}

	err = loadEncodedStreamContent(context.Background(), ctx, sd, false)
	if !errors.Is(err, errMissingStreamLength) {
		t.Fatalf("got %v, want %v", err, errMissingStreamLength)
	}
	if !strings.Contains(err.Error(), "stream length") {
		t.Fatalf("expected stream length context, got %q", err.Error())
	}
}

func TestEnsureIndirectStreamLengthIgnoresMissingReference(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Table[7] = &model.XRefTableEntry{Free: true}
	sd := &types.StreamDict{StreamLengthObjNr: intPtr(7)}

	if err := ensureIndirectStreamLength(context.Background(), ctx, sd, false); err != nil {
		t.Fatalf("expected missing stream length reference to be ignored, got %v", err)
	}
	if sd.StreamLength != nil {
		t.Fatalf("expected missing stream length reference to leave length nil, got %d", *sd.StreamLength)
	}
}

func TestStreamDictForObjectClassifiesMissingStreamOffset(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = streamDictForObject(context.Background(), ctx, types.Dict{}, 7, 0, 0, 0)
	if !errors.Is(err, errMissingStreamOffset) {
		t.Fatalf("got %v, want %v", err, errMissingStreamOffset)
	}
	if !strings.Contains(err.Error(), "stream obj#7") {
		t.Fatalf("expected stream object context, got %q", err.Error())
	}
}

func TestFilterPipelineClassifiesCorruptFilterArray(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pdfFilterPipeline(context.Background(), ctx, types.Dict{"Filter": types.Integer(1)})
	if !errors.Is(err, errCorruptFilterArray) {
		t.Fatalf("got %v, want %v", err, errCorruptFilterArray)
	}
	if !strings.Contains(err.Error(), "filter pipeline") {
		t.Fatalf("expected filter pipeline context, got %q", err.Error())
	}

	_, err = buildFilterPipeline(context.Background(), ctx, types.Array{types.Integer(1)}, nil)
	if !errors.Is(err, errCorruptFilterArray) {
		t.Fatalf("got %v, want %v", err, errCorruptFilterArray)
	}
	if !strings.Contains(err.Error(), "entry 0") {
		t.Fatalf("expected filter entry context, got %q", err.Error())
	}
}

func TestFilterPipelineClassifiesCorruptDecodeParms(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = singleFilter(context.Background(), ctx, filter.Flate, types.Dict{
		"DecodeParms": types.Array{types.Dict{}, types.Dict{}},
	})
	if !errors.Is(err, errCorruptDecodeParms) {
		t.Fatalf("got %v, want %v", err, errCorruptDecodeParms)
	}
	if !strings.Contains(err.Error(), filter.Flate) {
		t.Fatalf("expected filter name context, got %q", err.Error())
	}

	_, err = buildFilterPipeline(
		context.Background(),
		ctx,
		types.Array{types.Name(filter.Flate)},
		types.Array{types.Integer(1)},
	)
	if !errors.Is(err, errCorruptDecodeParms) {
		t.Fatalf("got %v, want %v", err, errCorruptDecodeParms)
	}
	if !strings.Contains(err.Error(), "entry 0") {
		t.Fatalf("expected DecodeParms entry context, got %q", err.Error())
	}
}

func TestSaveDecodedStreamContentIgnoresUnsupportedFilter(t *testing.T) {
	sd := &types.StreamDict{
		Raw:            []byte("abc"),
		FilterPipeline: []types.PDFFilter{{Name: filter.JBIG2}},
	}

	if err := saveDecodedStreamContent(nil, sd, 0, 0, true); err != nil {
		t.Fatalf("expected unsupported filter to be ignored, got %v", err)
	}
}

// TestReadLargeDictObjectStream verifies large object streams are handled.
func TestReadLargeDictObjectStream(t *testing.T) {
	// Test without "stream" and "endobj" inside the dictionary.
	var fp bytes.Buffer
	fp.WriteString("123 0 obj\n")
	data := make([]byte, 10*1024*1024)
	fp.WriteString("<<")
	fp.WriteString("/Foo <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString("/Bar (Test)\n")
	fp.WriteString("/Baz <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString(">>\n")
	fp.WriteString("stream\n")
	fp.WriteString("Hello world!\n")
	fp.WriteString("endstream\n")
	fp.WriteString("endobj\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObjectWithContext(ctx, c, 0, 123, 0)
	if err != nil {
		t.Fatal(err)
	}

	d, ok := o.(types.StreamDict)
	if !ok {
		t.Fatalf("expected StreamDict, got %T", o)
	}

	if err := loadEncodedStreamContent(ctx, c, &d, true); err != nil {
		t.Fatal(err)
	}

	if foo := d.HexLiteralEntry("Foo"); foo == nil {
		t.Error("expected Foo entry")
	} else if expected := hex.EncodeToString(data); foo.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(foo.Value()))
	}

	if bar := d.StringEntry("Bar"); bar == nil {
		t.Error("expected Bar entry")
	} else if expected := "Test"; *bar != expected {
		t.Errorf("expected %s for Bar, got %s", expected, *bar)
	}

	if baz := d.HexLiteralEntry("Baz"); baz == nil {
		t.Error("expected Baz entry")
	} else if expected := hex.EncodeToString(data); baz.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(baz.Value()))
	}

	if err := d.Decode(); err != nil {
		t.Fatal(err)
	}

	if expected := "Hello world!"; string(d.Content) != expected {
		t.Errorf("expected stream content %s, got %s", expected, string(d.Content))
	}
}
