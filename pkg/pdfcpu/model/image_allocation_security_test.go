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
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/hhrutter/tiff"
)

func singlePageTIFF(t *testing.T) ([]byte, int) {
	t.Helper()
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	ifd := int(binary.LittleEndian.Uint32(data[4:8]))
	entries := int(binary.LittleEndian.Uint16(data[ifd : ifd+2]))
	return data, ifd + 2 + entries*12
}

func TestCreateImageResourcesRejectsCyclicTIFF(t *testing.T) {
	data, next := singlePageTIFF(t)
	ifd := binary.LittleEndian.Uint32(data[4:8])
	binary.LittleEndian.PutUint32(data[next:next+4], ifd)

	c, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := CreateImageResources(imageConstructionXRefTable(false), bytes.NewReader(data), false, false)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "cyclic TIFF IFD offset") {
			t.Fatalf("got %v, want cyclic TIFF IFD error", err)
		}
	case <-c.Done():
		t.Fatal("cyclic TIFF import did not stop within two seconds")
	}
}

func TestCreateImageResourcesRejectsInvalidNextTIFFOffset(t *testing.T) {
	data, next := singlePageTIFF(t)
	binary.LittleEndian.PutUint32(data[next:next+4], uint32(len(data)))
	_, err := CreateImageResources(imageConstructionXRefTable(false), bytes.NewReader(data), false, false)
	if err == nil || !strings.Contains(err.Error(), "invalid TIFF IFD offset") {
		t.Fatalf("got %v, want invalid TIFF IFD offset error", err)
	}
}

func TestCreateImageResourcesEnforcesTIFFPageCount(t *testing.T) {
	var buf bytes.Buffer
	pages := []image.Image{
		image.NewGray(image.Rect(0, 0, 1, 1)),
		image.NewGray(image.Rect(0, 0, 1, 1)),
	}
	if err := tiff.EncodeAll(&buf, pages, nil); err != nil {
		t.Fatal(err)
	}
	xRefTable := imageConstructionXRefTable(false)
	xRefTable.Conf.Limits.MaxObjectCount = 1
	_, err := CreateImageResources(xRefTable, bytes.NewReader(buf.Bytes()), false, false)
	if err == nil || !strings.Contains(err.Error(), "TIFF page count exceeds limit 1") {
		t.Fatalf("got %v, want TIFF page count limit error", err)
	}
}
