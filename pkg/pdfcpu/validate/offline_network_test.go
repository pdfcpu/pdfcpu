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

package validate

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type offlineLinkTransport struct{ calls int }

// RoundTrip records attempted HTTP requests without accessing the network.
func (tr *offlineLinkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return nil, errors.New("test HTTP blocked")
}

// TestOfflineLinksSkipHTTP verifies offline mode takes precedence over link validation.
func TestOfflineLinksSkipHTTP(t *testing.T) {
	tr := &offlineLinkTransport{}
	old := http.DefaultTransport
	http.DefaultTransport = tr
	t.Cleanup(func() { http.DefaultTransport = old })
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	ctx, err := model.NewContext(strings.NewReader(""), conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.URIs = map[int]map[string]string{1: {"https://pdfcpu-offline-link.invalid/": ""}}
	if err := checkForBrokenLinks(ctx); err != nil {
		t.Fatal(err)
	}
	if tr.calls != 0 {
		t.Fatalf("offline HTTP attempts: %d", tr.calls)
	}
	ctx.Offline = false
	if err := checkForBrokenLinks(ctx); err == nil {
		t.Fatal("online control did not report failed request")
	}
	if tr.calls != 1 {
		t.Fatalf("online HTTP attempts: %d", tr.calls)
	}
}
