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
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestCreateOfflineSkipsRemoteImageDNS verifies configuration reaches the JSON image renderer.
func TestCreateOfflineSkipsRemoteImageDNS(t *testing.T) {
	var lookups atomic.Int32
	old := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		lookups.Add(1)
		return nil, errors.New("test DNS blocked")
	}}
	t.Cleanup(func() { net.DefaultResolver = old })
	const input = `{"paper":"A4","pages":{"1":{"content":{"image":[{"src":"http://pdfcpu-offline-create.invalid/image.png","width":20,"height":20}]}}}}`
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	var out bytes.Buffer
	if err := Create(t.Context(), nil, strings.NewReader(input), &out, conf); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 0 {
		t.Fatalf("offline DNS attempts: %d", lookups.Load())
	}
	if out.Len() == 0 {
		t.Fatal("missing offline PDF output")
	}
	conf.Offline = false
	out.Reset()
	if err := Create(t.Context(), nil, strings.NewReader(input), &out, conf); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() == 0 {
		t.Fatal("online control made no DNS attempt")
	}
}
