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
	"context"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type flushFailingWriter struct {
	err   error
	calls int
}

func (w *flushFailingWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, w.err
}

func lazyWriteFailure(err error) types.Object {
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("object")},
	}
	return types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, err
	})
}

func requireWriteAndFlushCauses(t *testing.T, err, writeErr, flushErr error, w *flushFailingWriter) {
	t.Helper()
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write cause %v, got %v", writeErr, err)
	}
	if !errors.Is(err, flushErr) {
		t.Fatalf("expected flush cause %v, got %v", flushErr, err)
	}
	if w.calls == 0 {
		t.Fatal("expected buffered output to be flushed")
	}
}

func TestWriteContextJoinsWriteAndFlushErrors(t *testing.T) {
	writeErr := errors.New("bind name trees failed")
	flushErr := errors.New("flush failed")
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	ir, err := ctx.IndRefForNewObject(lazyWriteFailure(writeErr))
	if err != nil {
		t.Fatal(err)
	}
	ctx.RootDict["Names"] = *ir
	ctx.Names["JavaScript"] = &model.Node{D: types.NewDict()}
	w := &flushFailingWriter{err: flushErr}

	err = WriteContext(ctx, w)

	requireWriteAndFlushCauses(t, err, writeErr, flushErr, w)
}

func TestWriteIncrementJoinsWriteAndFlushErrors(t *testing.T) {
	writeErr := errors.New("write increment failed")
	flushErr := errors.New("flush failed")
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	valueRef, err := ctx.IndRefForNewObject(types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	lengthRef, err := ctx.IndRefForNewObject(lazyWriteFailure(writeErr))
	if err != nil {
		t.Fatal(err)
	}
	streamLength := int64(4)
	streamRef, err := ctx.IndRefForNewObject(types.StreamDict{
		Dict:         types.Dict{"Length": *lengthRef},
		Raw:          []byte("data"),
		StreamLength: &streamLength,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx.Write.ObjNrs = []int{valueRef.ObjectNumber.Value(), streamRef.ObjectNumber.Value()}
	w := &flushFailingWriter{err: flushErr}

	err = WriteIncrement(ctx, w)

	requireWriteAndFlushCauses(t, err, writeErr, flushErr, w)
}
