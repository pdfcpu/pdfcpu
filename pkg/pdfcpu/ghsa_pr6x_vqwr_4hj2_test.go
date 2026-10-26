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

// GHSA-pr6x-vqwr-4hj2: encoded stream data must count toward the per-read budget, a stream length beyond the end of the
// input must not size the buffer, object-stream indexes must not be shared, and lazy object-stream decodes must use a
// separate cumulative budget from the XRefTable budget. Otherwise shared or nested streams can make a small file hold
// many independent buffers. Other decoded buffers are covered in package model.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-pr6x-vqwr-4hj2

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	ghsaPr6xStreamBytes = 64 << 10
	ghsaPr6xCopies      = 400
	ghsaPr6xAllocLimit  = 8 << 20
)

var (
	ghsaPr6xData   = strings.Repeat("A", ghsaPr6xStreamBytes)
	ghsaPr6xLength = fmt.Sprintf("<</Length %d>>", ghsaPr6xStreamBytes)
)

// ghsaPr6xPDF writes a one-page document followed by body and an xref table whose entries 4.. use offsets.
func ghsaPr6xPDF(body func(b *bytes.Buffer) []int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offs := []int{b.Len()}
	b.WriteString("1 0 obj\n<</Type/Catalog/Pages 2 0 R>>\nendobj\n")
	offs = append(offs, b.Len())
	b.WriteString("2 0 obj\n<</Type/Pages/Kids[3 0 R]/Count 1>>\nendobj\n")
	offs = append(offs, b.Len())
	b.WriteString("3 0 obj\n<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>\nendobj\n")
	offs = append(offs, body(&b)...)

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, off := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(offs)+1, xref)
	return b.Bytes()
}

// ghsaPr6xSharedOffset lists copies xref entries for one stream object with dict and data.
func ghsaPr6xSharedOffset(copies int, dict, data string) []byte {
	return ghsaPr6xPDF(func(b *bytes.Buffer) []int {
		off := b.Len()
		fmt.Fprintf(b, "4 0 obj\n%s\nstream\n%s\nendstream\nendobj\n", dict, data)
		offs := make([]int, copies)
		for i := range offs {
			offs[i] = off
		}
		return offs
	})
}

// ghsaPr6xNested nests copies stream object headers inside one stream body; every stream ends at the same endstream.
func ghsaPr6xNested(copies int) []byte {
	return ghsaPr6xPDF(func(b *bytes.Buffer) []int {
		const header = "%d 0 obj\n<</Length %010d>>\nstream\n"
		start := b.Len()
		offs := make([]int, copies)
		lens := make([]int, copies)
		pos := start
		for i := range offs {
			offs[i] = pos
			lens[i] = len(fmt.Sprintf(header, 4+i, 0))
			pos += lens[i]
		}
		end := pos + ghsaPr6xStreamBytes
		for i := range offs {
			fmt.Fprintf(b, header, 4+i, end-(offs[i]+lens[i]))
		}
		b.WriteString(strings.Repeat("A", ghsaPr6xStreamBytes))
		b.WriteString("\nendstream\nendobj\n")
		return offs
	})
}

// ghsaPr6xDeclaredLength writes copies small streams, each declaring streamLength.
func ghsaPr6xDeclaredLength(copies, streamLength int, data string) []byte {
	return ghsaPr6xPDF(func(b *bytes.Buffer) []int {
		offs := make([]int, copies)
		for i := range offs {
			offs[i] = b.Len()
			fmt.Fprintf(b, "%d 0 obj\n<</Length %d>>\nstream\n%s\nendstream\nendobj\n", 4+i, streamLength, data)
		}
		return offs
	})
}

func ghsaPr6xFlateStream(t *testing.T, decodedBytes int) (string, string) {
	t.Helper()
	var encoded bytes.Buffer
	zw := zlib.NewWriter(&encoded)
	chunk := []byte(strings.Repeat("A", min(decodedBytes, 32<<10)))
	for remaining := decodedBytes; remaining > 0; remaining -= len(chunk) {
		if _, err := zw.Write(chunk[:min(remaining, len(chunk))]); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("<</Length %d/Filter/FlateDecode>>", encoded.Len()), encoded.String()
}

func ghsaPr6xRead(t *testing.T, data []byte, mode int) (*model.Context, uint64, error) {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	// Keep the budget, which includes MaxObjectBytes, and the allocations of this test small.
	conf.Limits.MaxObjectBytes = 256 << 10

	c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	ctx, err := Read(c, bytes.NewReader(data), conf)
	runtime.ReadMemStats(&after)
	return ctx, after.TotalAlloc - before.TotalAlloc, err
}

// ghsaPr6xRetained returns the capacity of all stream buffers held by ctx.
func ghsaPr6xRetained(ctx *model.Context) int64 {
	var n int64
	for _, entry := range ctx.Table {
		if sd, ok := entry.Object.(types.StreamDict); ok {
			n += int64(cap(sd.Raw))
		}
	}
	return n
}

func TestGHSA_pr6x_vqwr_4hj2_RepeatedStreamDataHasBudget(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		modes []int
	}{
		{"shared offset", ghsaPr6xSharedOffset(ghsaPr6xCopies, ghsaPr6xLength, ghsaPr6xData), []int{model.ValidationStrict, model.ValidationRelaxed}},
		{"shared offset without length", ghsaPr6xSharedOffset(ghsaPr6xCopies, "<<>>", ghsaPr6xData), []int{model.ValidationRelaxed}},
		{"nested streams", ghsaPr6xNested(ghsaPr6xCopies), []int{model.ValidationStrict, model.ValidationRelaxed}},
	}

	for _, tt := range tests {
		for _, mode := range tt.modes {
			t.Run(fmt.Sprintf("%s/mode=%d", tt.name, mode), func(t *testing.T) {
				_, alloc, err := ghsaPr6xRead(t, tt.data, mode)
				if !errors.Is(err, errObjectReadBudget) {
					t.Fatalf("got %v, want object read budget error", err)
				}
				if alloc > ghsaPr6xAllocLimit {
					t.Fatalf("read allocated %d bytes for %d-byte input, want at most %d", alloc, len(tt.data), ghsaPr6xAllocLimit)
				}
			})
		}
	}
}

func TestGHSA_pr6x_vqwr_4hj2_StreamLengthBeyondInputIsNotAllocated(t *testing.T) {
	const copies, streamLength = 64, 16 << 20
	data := ghsaPr6xDeclaredLength(copies, streamLength, "AAAA")

	ctx, alloc, err := ghsaPr6xRead(t, data, model.ValidationRelaxed)
	if err != nil {
		t.Fatal(err)
	}
	if alloc > ghsaPr6xAllocLimit {
		t.Fatalf("read allocated %d bytes for %d-byte input, want at most %d", alloc, len(data), ghsaPr6xAllocLimit)
	}
	if retained, limit := ghsaPr6xRetained(ctx), objectReadBudgetLimit(ctx); retained > limit {
		t.Fatalf("read holds %d bytes of stream buffers, want at most the %d-byte read budget", retained, limit)
	}
	sd, ok := ctx.Table[4+copies-1].Object.(types.StreamDict)
	if !ok {
		t.Fatalf("got %T, want stream dict", ctx.Table[4+copies-1].Object)
	}
	if got := string(sd.Raw); got != "AAAA\n" {
		t.Fatalf("got stream data %q, want %q", got, "AAAA\n")
	}
}

func TestGHSA_pr6x_vqwr_4hj2_ShortStreamsStayWithinBudget(t *testing.T) {
	const copies = 1000
	tests := []struct {
		name  string
		data  []byte
		modes []int
	}{
		{"empty stream", ghsaPr6xSharedOffset(copies, "<</Length 0>>", ""), []int{model.ValidationStrict, model.ValidationRelaxed}},
		{"empty stream without length", ghsaPr6xSharedOffset(copies, "<<>>", ""), []int{model.ValidationRelaxed}},
		{"short stream without length", ghsaPr6xSharedOffset(copies, "<<>>", "AAAA"), []int{model.ValidationRelaxed}},
	}

	for _, tt := range tests {
		for _, mode := range tt.modes {
			t.Run(fmt.Sprintf("%s/mode=%d", tt.name, mode), func(t *testing.T) {
				ctx, _, err := ghsaPr6xRead(t, tt.data, mode)
				if err != nil {
					t.Fatal(err)
				}
				if retained, limit := ghsaPr6xRetained(ctx), objectReadBudgetLimit(ctx); retained > limit {
					t.Fatalf("read holds %d bytes of stream buffers, want at most the %d-byte read budget", retained, limit)
				}
			})
		}
	}
}

func TestGHSA_pr6x_vqwr_4hj2_EagerDecodedDataDoesNotUseObjectBudget(t *testing.T) {
	dict, encoded := ghsaPr6xFlateStream(t, 2<<20)
	data := ghsaPr6xSharedOffset(1, dict, encoded)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	conf.DecodeAllStreams = true
	conf.Limits.MaxObjectBytes = 1 << 10

	ctx, err := Read(context.Background(), bytes.NewReader(data), conf)
	if err != nil {
		t.Fatal(err)
	}
	sd := ctx.Table[4].Object.(types.StreamDict)
	if len(sd.Content) != 2<<20 {
		t.Fatalf("got %d decoded bytes, want 2 MiB", len(sd.Content))
	}

	conf.Limits.MaxDecodeBytes = -1
	if _, err := Read(context.Background(), bytes.NewReader(data), conf); err != nil {
		t.Fatalf("read with disabled decode limit: %v", err)
	}
}

func ghsaPr6xObjectStreamPDF(alias bool) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	var offsets [6]int
	for i, body := range []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[3 0 R]/Count 1>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>",
	} {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}

	prolog, values, count := "6 0 7 3 ", "42 43", 2
	if alias {
		prolog, values, count = "6 0 ", "42", 1
	}
	offsets[4] = b.Len()
	fmt.Fprintf(&b, "4 0 obj\n<</Type/ObjStm/N %d/First %d/Length %d>>\nstream\n%s%s\nendstream\nendobj\n",
		count, len(prolog), len(prolog)+len(values), prolog, values)

	offsets[5] = b.Len()
	var xref bytes.Buffer
	writeEntry := func(kind byte, field2 uint32, field3 uint16) {
		xref.WriteByte(kind)
		xref.Write([]byte{byte(field2 >> 24), byte(field2 >> 16), byte(field2 >> 8), byte(field2)})
		xref.Write([]byte{byte(field3 >> 8), byte(field3)})
	}
	writeEntry(0, 0, 65535)
	for i := 1; i <= 5; i++ {
		writeEntry(1, uint32(offsets[i]), 0)
	}
	writeEntry(2, 4, 0)
	index := uint16(1)
	if alias {
		index = 0
	}
	writeEntry(2, 4, index)
	fmt.Fprintf(&b, "5 0 obj\n<</Type/XRef/Size 8/Root 1 0 R/W[1 4 2]/Length %d>>\nstream\n", xref.Len())
	b.Write(xref.Bytes())
	b.WriteString("\nendstream\nendobj\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", offsets[5])
	return b.Bytes()
}

func TestGHSA_pr6x_vqwr_4hj2_ObjectStreamIndexAliasesRejected(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = mode
			_, err := Read(context.Background(), bytes.NewReader(ghsaPr6xObjectStreamPDF(true)), conf)
			if !errors.Is(err, errCorruptXRefStream) || !strings.Contains(err.Error(), "objects 6 and 7") {
				t.Fatalf("got %v, want duplicate object-stream index error", err)
			}

			ctx, err := Read(context.Background(), bytes.NewReader(ghsaPr6xObjectStreamPDF(false)), conf)
			if err != nil {
				t.Fatal(err)
			}
			for objNr, want := range map[int]int{6: 42, 7: 43} {
				o, err := dereferencedObject(context.Background(), ctx, objNr)
				if err != nil {
					t.Fatal(err)
				}
				if got, ok := o.(types.Integer); !ok || got.Value() != want {
					t.Fatalf("object %d: got %v, want %d", objNr, o, want)
				}
			}
		})
	}
}

func TestGHSA_pr6x_vqwr_4hj2_ObjectStreamIndexAliasesBeyondTableSize(t *testing.T) {
	// Object numbers need not be contiguous, so aliases above the entry count must be found as well.
	ctx := &model.Context{
		XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{0: {Free: true}}},
		Read:      &model.ReadContext{ObjectStreams: types.IntSet{4: true}},
	}
	stream, index := 4, 0
	for _, objNr := range []int{100, 101} {
		ctx.Table[objNr] = &model.XRefTableEntry{Compressed: true, ObjectStream: &stream, ObjectStreamInd: &index}
	}

	err := validateObjectStreamIndexUniqueness(ctx)
	if !errors.Is(err, errCorruptXRefStream) || !strings.Contains(err.Error(), "objects 100 and 101") {
		t.Fatalf("got %v, want duplicate object-stream index error", err)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_ObjectStreamBudgetSemantics(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	if got := newObjectStreamDecodeBudget(conf).limit; got != 1<<30 {
		t.Fatalf("got default budget %d, want 1 GiB", got)
	}
	conf.Limits.MaxDecodeBytes = 2 << 30
	if got := newObjectStreamDecodeBudget(conf).limit; got != 4<<30 {
		t.Fatalf("got raised budget %d, want 4 GiB", got)
	}
	conf.Limits.MaxDecodeBytes = -1
	if got := newObjectStreamDecodeBudget(conf).limit; got != -1 {
		t.Fatalf("disabled limit: got %d, want -1", got)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_ObjectStreamCountIgnoresStaleRevisions(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{}},
		Read:      &model.ReadContext{ObjectStreams: types.IntSet{4: true, 5: true, 99: true}},
	}
	stream4, stream5 := 4, 5
	ctx.Table[10] = &model.XRefTableEntry{Compressed: true, ObjectStream: &stream4}
	// Compressed entries are materialized during reading but retain their object-stream number for statistics.
	ctx.Table[11] = &model.XRefTableEntry{ObjectStream: &stream5}
	ctx.Table[12] = &model.XRefTableEntry{Free: true, Compressed: true, ObjectStream: new(int)}

	got := activeObjectStreams(ctx)
	if len(got) != 2 || got[0] != 4 || got[1] != 5 {
		t.Fatalf("got active object streams %v, want [4 5]", got)
	}
	if count, got := currentObjectStreamsString(ctx); count != 2 || got != "4,5" {
		t.Fatalf("got %d current object streams %q, want 2 and %q", count, got, "4,5")
	}
}

func TestGHSA_pr6x_vqwr_4hj2_LazyObjectStreamsUseCumulativeBudget(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	ctx, err := model.NewContext(bytes.NewReader(nil), conf)
	if err != nil {
		t.Fatal(err)
	}
	budget := &objectStreamDecodeBudget{limit: 1 << 30}

	lazyObject := func(decodedBytes int) types.LazyObjectStreamObject {
		content := append([]byte("6 0 "), bytes.Repeat([]byte(" "), decodedBytes-5)...)
		content = append(content, '0')
		sd := types.StreamDict{
			Dict:           types.Dict{"N": types.Integer(1), "First": types.Integer(4)},
			Content:        content,
			FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
		}
		if err := sd.Encode(); err != nil {
			t.Fatal(err)
		}
		sd.Content = nil

		osd, err := decodeObjectStreamObjects(context.Background(), ctx, &sd, 4, budget)
		if err != nil {
			t.Fatal(err)
		}
		if osd.MaxDecodeBytes != 512<<20 {
			t.Fatalf("got object-stream limit %d, want 512 MiB", osd.MaxDecodeBytes)
		}
		o, err := osd.IndexedObject(0)
		if err != nil {
			t.Fatal(err)
		}
		return o.(types.LazyObjectStreamObject)
	}

	accepted := lazyObject(2 << 20)
	if _, err := accepted.DecodedObject(context.Background()); err != nil {
		t.Fatal(err)
	}
	if budget.used == 0 {
		t.Fatal("decoded object stream was not charged")
	}

	budget.limit = budget.used + 1
	rejected := lazyObject(2 << 20)
	if _, err := rejected.DecodedObject(context.Background()); !errors.Is(err, filter.ErrDecodeLimitExceeded) ||
		!strings.Contains(err.Error(), "object stream decoding exceeded") ||
		!strings.Contains(err.Error(), "cumulative budget") {
		t.Fatalf("got %v, want cumulative object-stream budget error", err)
	}
}

// ghsaPr6xTwoObjectStream returns a Flate object stream holding a 1 MiB string and an integer, decoded through budget,
// its two lazy objects and the size of the string object.
func ghsaPr6xTwoObjectStream(t *testing.T, budget *objectStreamDecodeBudget) (*types.ObjectStreamDict, []types.LazyObjectStreamObject, int) {
	t.Helper()
	ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	firstObject := "(" + strings.Repeat("a", 1<<20) + ") "
	prolog := fmt.Sprintf("6 0 7 %d ", len(firstObject))
	sd := types.StreamDict{
		Dict:           types.Dict{"N": types.Integer(2), "First": types.Integer(len(prolog))},
		Content:        []byte(prolog + firstObject + "0"),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil

	osd, err := decodeObjectStreamObjects(context.Background(), ctx, &sd, 4, budget)
	if err != nil {
		t.Fatal(err)
	}
	lazy := make([]types.LazyObjectStreamObject, 2)
	for i := range lazy {
		o, err := osd.IndexedObject(i)
		if err != nil {
			t.Fatal(err)
		}
		lazy[i] = o.(types.LazyObjectStreamObject)
	}
	return osd, lazy, len(firstObject)
}

// ghsaPr6xGetDataConcurrently calls GetData on all objects at the same time and returns their errors.
func ghsaPr6xGetDataConcurrently(objs []types.LazyObjectStreamObject) []error {
	errs := make([]error, len(objs))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, o := range objs {
		wg.Go(func() {
			<-start
			_, errs[i] = o.GetData()
		})
	}
	close(start)
	wg.Wait()
	return errs
}

func TestGHSA_pr6x_vqwr_4hj2_RejectedObjectStreamStaysRejected(t *testing.T) {
	budget := &objectStreamDecodeBudget{limit: 1}
	osd, lazy, objectBytes := ghsaPr6xTwoObjectStream(t, budget)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, firstErr := lazy[0].GetData()
	runtime.ReadMemStats(&after)
	if !errors.Is(firstErr, filter.ErrDecodeLimitExceeded) || osd.Content != nil || budget.used != 0 {
		t.Fatalf("first access: content=%d budget=%d err=%v", len(osd.Content), budget.used, firstErr)
	}
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > uint64(objectBytes/2) {
		t.Fatalf("rejected decode allocated %d bytes, want at most %d", alloc, objectBytes/2)
	}

	// Later accesses must return the cached budget error without attempting to decode again or racing on Content.
	osd.Raw = []byte("not flate data")
	for _, err := range ghsaPr6xGetDataConcurrently(lazy) {
		if err == nil || err.Error() != firstErr.Error() {
			t.Fatalf("later access: err=%v, want %v", err, firstErr)
		}
	}
	if osd.Content != nil || budget.used != 0 {
		t.Fatalf("later accesses: content=%d budget=%d", len(osd.Content), budget.used)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_ObjectStreamCopiedOnce(t *testing.T) {
	budget := &objectStreamDecodeBudget{limit: 1 << 30}
	osd, lazy, _ := ghsaPr6xTwoObjectStream(t, budget)
	if _, err := lazy[0].GetData(); err != nil {
		t.Fatal(err)
	}
	if len(osd.Content) != cap(osd.Content) || budget.used != int64(len(osd.Content)) {
		t.Fatalf("decoded length=%d capacity=%d charged=%d", len(osd.Content), cap(osd.Content), budget.used)
	}
	// Every later object access reuses the decoded stream without copying it again.
	if allocs := testing.AllocsPerRun(10, func() { _, _ = lazy[1].GetData() }); allocs != 0 {
		t.Fatalf("repeated GetData allocated %.0f times, want 0", allocs)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_SingleStreamReads(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			ctx, _, err := ghsaPr6xRead(t, ghsaPr6xSharedOffset(1, ghsaPr6xLength, ghsaPr6xData), mode)
			if err != nil {
				t.Fatal(err)
			}
			sd, ok := ctx.Table[4].Object.(types.StreamDict)
			if !ok {
				t.Fatalf("got %T, want stream dict", ctx.Table[4].Object)
			}
			if len(sd.Raw) != ghsaPr6xStreamBytes {
				t.Fatalf("got %d stream bytes, want %d", len(sd.Raw), ghsaPr6xStreamBytes)
			}
		})
	}
}
