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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type offlineLinkTransport struct{ calls int }

// RoundTrip records attempted HTTP requests without accessing the network.
func (tr *offlineLinkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return nil, errors.New("test HTTP blocked")
}

func offlineLinkContext(t *testing.T, mode int) *model.Context {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	ctx, err := model.NewContext(strings.NewReader(""), conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.XRefTable.ValidationMode = mode
	ctx.RootDict = types.Dict{"URI": types.Dict{"Base": types.StringLiteral("https://example.com/base/")}}
	setLinkTestTarget(ctx.XRefTable, 1, "child", linkTarget{kind: linkTargetURI, source: linkSourceURIAction})
	setLinkTestTarget(
		ctx.XRefTable, 1, "file:///tmp/private", linkTarget{kind: linkTargetURI, source: linkSourceURIAction},
	)
	setLinkTestTarget(
		ctx.XRefTable, 1, "http://127.0.0.1/private", linkTarget{kind: linkTargetURI, source: linkSourceURIAction},
	)
	setLinkTestTarget(
		ctx.XRefTable, 1, "https://example.com/public", linkTarget{kind: linkTargetURI, source: linkSourceURIAction},
	)
	setLinkTestTarget(
		ctx.XRefTable, 1, "mailto:user@example.com", linkTarget{kind: linkTargetURI, source: linkSourceURIAction},
	)
	setLinkTestTarget(ctx.XRefTable, 1, "remote.pdf", linkTarget{kind: linkTargetFile, source: linkSourceRemoteGoTo})
	setLinkTestTarget(
		ctx.XRefTable, 1, "viewer.exe", linkTarget{kind: linkTargetExecutable, source: linkSourceLaunch},
	)
	return ctx
}

func expectedOfflineNoticeCount(mode, resultCount int) int {
	if mode == model.ValidationRelaxed {
		return resultCount
	}
	return 0
}

func testOfflineLinksReportEveryTarget(t *testing.T, mode int) {
	ctx := offlineLinkContext(t, mode)
	tr := &offlineLinkTransport{}
	err := checkForBrokenLinksUsing(t.Context(), ctx, http.Client{Transport: tr})
	if mode == model.ValidationStrict && !errors.Is(err, errLinkVerification) {
		t.Fatalf("strict mode: got %v, want link verification error", err)
	}
	if mode == model.ValidationRelaxed && err != nil {
		t.Fatalf("relaxed mode: %v", err)
	}
	if tr.calls != 0 {
		t.Fatalf("mode %d: offline HTTP attempts: %d", mode, tr.calls)
	}

	wantStatuses := map[string]string{
		"child":                      "k",
		"file:///tmp/private":        "b",
		"http://127.0.0.1/private":   "b",
		"https://example.com/public": "k",
		"mailto:user@example.com":    "k",
		"remote.pdf":                 "k",
		"viewer.exe":                 "k",
	}
	for uri, want := range wantStatuses {
		if got := ctx.URIs[1][uri]; got != want {
			t.Fatalf("mode %d, URI %q: got status %q, want %q", mode, uri, got, want)
		}
	}
	notices := ctx.ValidationReport().Notices()
	wantNotices := expectedOfflineNoticeCount(mode, len(wantStatuses))
	if len(notices) != wantNotices {
		t.Fatalf("mode %d: got %d notices, want %d", mode, len(notices), wantNotices)
	}
	if mode == model.ValidationRelaxed {
		joined := ""
		for _, notice := range notices {
			joined += notice.Message + "\n"
		}
		if strings.Count(joined, "offline mode: HTTP link was not checked") != 2 {
			t.Fatalf("mode %d: offline notices: %q", mode, joined)
		}
	}
}

// TestOfflineLinksReportEveryTargetWithoutTransport verifies offline link assessment remains complete and local.
func TestOfflineLinksReportEveryTargetWithoutTransport(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		testOfflineLinksReportEveryTarget(t, mode)
	}
}
