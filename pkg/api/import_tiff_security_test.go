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
	"context"
	"encoding/binary"
	"image"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/hhrutter/tiff"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestImportImagesRejectsCyclicTIFF(t *testing.T) {
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	ifd := binary.LittleEndian.Uint32(data[4:8])
	entries := binary.LittleEndian.Uint16(data[ifd : ifd+2])
	next := int(ifd) + 2 + int(entries)*12
	binary.LittleEndian.PutUint32(data[next:next+4], ifd)

	c, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		var output bytes.Buffer
		done <- ImportImages(c, nil, &output, []io.Reader{bytes.NewReader(data)}, nil, model.NewStatelessConfiguration())
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
