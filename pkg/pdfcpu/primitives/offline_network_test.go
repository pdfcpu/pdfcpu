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

package primitives

import (
	"errors"
	"net/http"
	"testing"
)

type offlineImageTransport struct{ calls int }

// RoundTrip records attempted HTTP requests without accessing the network.
func (tr *offlineImageTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return nil, errors.New("test HTTP blocked")
}

// TestOfflineImageSkipsHTTP verifies an existing image client cannot bypass offline mode.
func TestOfflineImageSkipsHTTP(t *testing.T) {
	tr := &offlineImageTransport{}
	pdf := &PDF{Offline: true, httpClient: &http.Client{Transport: tr}}
	ib := &ImageBox{Src: "https://pdfcpu-offline-image.invalid/image.png", pdf: pdf}
	reader, err := ib.resource()
	if err != nil || reader != nil || tr.calls != 0 {
		t.Fatalf("offline image: reader=%v, err=%v, HTTP=%d", reader, err, tr.calls)
	}
	pdf.Offline = false
	_, err = ib.resource()
	if err == nil || tr.calls != 1 {
		t.Fatalf("online control: err=%v, HTTP=%d", err, tr.calls)
	}
}
