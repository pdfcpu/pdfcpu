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

// GHSA-pr6x-vqwr-4hj2: deferred decoding must share a cumulative decoded-buffer budget across aliased streams.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-pr6x-vqwr-4hj2

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestGHSA_pr6x_vqwr_4hj2_CumulativeBudgetCoversDeferredAliases(t *testing.T) {
	conf := NewStatelessConfiguration()
	conf.Limits.MaxDecodeBytes = 64
	x := newXRefTable(conf)

	base := decodeLimitTestStream(t, true, false)
	first, second := base, base
	if err := x.DecodeStream(&first); err != nil {
		t.Fatal(err)
	}
	if x.decodedStreamBudget.used == 0 {
		t.Fatal("decoded buffer was not charged")
	}

	// Model other retained buffers bringing the document within one stream of its limit while keeping the fixture small.
	x.decodedStreamBudget.used = decodedStreamBufferLimit(conf) - int64(len(first.Content)) + 1
	err := x.DecodeStream(&second)
	if !errors.Is(err, filter.ErrDecodeLimitExceeded) || !strings.Contains(err.Error(), "cumulative budget") {
		t.Fatalf("got %v, want cumulative decoded-buffer error", err)
	}
	if second.Content != nil {
		t.Fatalf("failed decode retained %d bytes", len(second.Content))
	}
}

func TestGHSA_pr6x_vqwr_4hj2_CumulativeDecodeBudgetSemantics(t *testing.T) {
	conf := NewStatelessConfiguration()
	if got := decodedStreamBufferLimit(conf); got != 1<<30 {
		t.Fatalf("got default budget %d, want 1 GiB", got)
	}
	conf.Limits.MaxDecodeBytes = 2 << 30
	if got := decodedStreamBufferLimit(conf); got != 4<<30 {
		t.Fatalf("got raised budget %d, want 4 GiB", got)
	}
	conf.Limits.MaxDecodeBytes = -1
	if got := decodedStreamBufferLimit(conf); got != -1 {
		t.Fatalf("got disabled budget %d, want -1", got)
	}

	x := newXRefTable(conf)
	first := decodeLimitTestStream(t, true, false)
	second := first
	if err := x.DecodeStream(&first); err != nil {
		t.Fatal(err)
	}
	if err := x.DecodeStream(&second); err != nil {
		t.Fatal(err)
	}
	if x.decodedStreamBudget.used != 0 {
		t.Fatalf("disabled budget charged %d bytes", x.decodedStreamBudget.used)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_CumulativeDecodeBudgetCountsCapacityOnce(t *testing.T) {
	conf := NewStatelessConfiguration()
	x := newXRefTable(conf)
	sd := decodeLimitTestStream(t, true, false)
	if err := x.DecodeStream(&sd); err != nil {
		t.Fatal(err)
	}
	want := int64(cap(sd.Content))
	if got := x.decodedStreamBudget.used; got != want {
		t.Fatalf("charged %d bytes, want capacity %d", got, want)
	}
	if err := x.DecodeStream(&sd); err != nil {
		t.Fatal(err)
	}
	if got := x.decodedStreamBudget.used; got != want {
		t.Fatalf("cached decode charged %d bytes, want %d", got, want)
	}

	raw := decodeLimitTestStream(t, false, false)
	if err := x.DecodeStream(&raw); err != nil {
		t.Fatal(err)
	}
	if got := x.decodedStreamBudget.used; got != want {
		t.Fatalf("raw alias charged %d bytes, want %d", got, want)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_DecodeStopsAtRemainingBudget(t *testing.T) {
	const decodedBytes, remaining = 4 << 20, 1 << 10
	sd := types.StreamDict{
		Dict:           types.NewDict(),
		Content:        bytes.Repeat([]byte("q Q\n"), decodedBytes/4),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil

	conf := NewStatelessConfiguration()
	x := newXRefTable(conf)
	// The per-stream limit admits the stream, so only the remaining budget can stop the decoder early.
	x.decodedStreamBudget.used = decodedStreamBufferLimit(conf) - remaining

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	err := x.DecodeStream(&sd)
	runtime.ReadMemStats(&after)
	if !errors.Is(err, filter.ErrDecodeLimitExceeded) || !strings.Contains(err.Error(), "cumulative budget") {
		t.Fatalf("got %v, want cumulative decoded-buffer error", err)
	}
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > decodedBytes/4 {
		t.Fatalf("decode allocated %d bytes with %d bytes of budget left, want at most %d", alloc, remaining, decodedBytes/4)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_FailedChargeDropsContent(t *testing.T) {
	conf := NewStatelessConfiguration()
	probe := decodeLimitTestStream(t, true, false)
	if err := newXRefTable(conf).DecodeStream(&probe); err != nil {
		t.Fatal(err)
	}
	if cap(probe.Content) <= len(probe.Content) {
		t.Fatalf("decoded capacity %d equals length %d; the charge failure path needs spare capacity", cap(probe.Content), len(probe.Content))
	}

	// The remaining budget admits the decoded length but not the buffer capacity.
	x := newXRefTable(conf)
	used := decodedStreamBufferLimit(conf) - int64(len(probe.Content))
	x.decodedStreamBudget.used = used
	sd := decodeLimitTestStream(t, true, false)
	err := x.DecodeStream(&sd)
	if !errors.Is(err, filter.ErrDecodeLimitExceeded) || !strings.Contains(err.Error(), "cumulative budget") {
		t.Fatalf("got %v, want cumulative decoded-buffer error", err)
	}
	if sd.Content != nil {
		t.Fatalf("failed charge retained %d decoded bytes", len(sd.Content))
	}
	if x.decodedStreamBudget.used != used {
		t.Fatalf("failed charge changed budget use from %d to %d", used, x.decodedStreamBudget.used)
	}
}
