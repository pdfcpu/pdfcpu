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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type linkTestResolver struct {
	ips []net.IPAddr
	err error
}

func (r linkTestResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.ips, r.err
}

func linkTestIPs(values ...string) []net.IPAddr {
	ips := make([]net.IPAddr, 0, len(values))
	for _, value := range values {
		ips = append(ips, net.IPAddr{IP: net.ParseIP(value)})
	}
	return ips
}

func TestValidateLinkURL(t *testing.T) {
	for _, tt := range []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{"HTTP", "http://example.com/path", false},
		{"UppercaseHTTP", "HTTP://example.com/path", false},
		{"HTTPS", "https://example.com/path", false},
		{"Credentials", "https://user:secret@example.com/path", true},
		{"File", "file:///tmp/data", true},
		{"MissingHost", "https:///path", true},
		{"LoopbackIPv4", "http://127.0.0.1/path", true},
		{"LoopbackIPv6", "http://[::1]/path", true},
		{"Private", "http://10.0.0.1/path", true},
		{"LinkLocal", "http://169.254.169.254/path", true},
		{"Shared", "http://100.64.0.1/path", true},
		{"NAT64Public", "http://[64:ff9b::808:808]/path", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatal(err)
			}
			err = validateLinkURL(u)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate URL: got %v, want error %t", err, tt.wantErr)
			}
		})
	}
}

func TestLinkRequestURLClassifiesSchemes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		uri       string
		wantURL   string
		wantClass error
		wantErr   bool
	}{
		{"HTTP", "HTTP://example.com/path", "http://example.com/path", nil, false},
		{"HTTPS", "HTTPS://example.com/path", "https://example.com/path", nil, false},
		{"Normalized", "HTTPS://EXAMPLE.COM/path#section", "https://example.com/path", nil, false},
		{"Mail", "mailto:user@example.com", "", errLinkSchemeSkipped, true},
		{"FTP", "ftp://example.com/file", "", errLinkSchemeSkipped, true},
		{"Relative", "relative/path", "", errLinkSchemeSkipped, true},
		{"File", "file:///tmp/data", "", errLinkTargetBlocked, true},
		{"Data", "data:text/plain,hello", "", errLinkTargetBlocked, true},
		{"JavaScript", "javascript:alert(1)", "", errLinkTargetBlocked, true},
		{"VBScript", "vbscript:msgbox(1)", "", errLinkTargetBlocked, true},
		{"Private", "http://127.0.0.1/path", "", errLinkTargetBlocked, true},
		{"Malformed", "http://[::1", "", nil, true},
		{"MalformedEscape", "https://example.com/%zz", "", nil, true},
		{"ControlCharacter", "https://example.com/\x00", "", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := linkRequestURL(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got %v, want error %t", err, tt.wantErr)
			}
			if tt.wantClass != nil && !errors.Is(err, tt.wantClass) {
				t.Fatalf("got %v, want %v", err, tt.wantClass)
			}
			if got != tt.wantURL {
				t.Fatalf("request URL: got %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestLinkRequestURLResolvesCatalogBase(t *testing.T) {
	for _, tt := range []struct {
		name   string
		base   string
		target string
		want   string
	}{
		{
			name:   "relative path",
			base:   "https://Example.COM/docs/current/",
			target: "../guide?lang=en#intro",
			want:   "https://example.com/docs/guide?lang=en",
		},
		{
			name:   "network path",
			base:   "https://example.com/docs/",
			target: "//cdn.example.com/manual#page",
			want:   "https://cdn.example.com/manual",
		},
		{
			name:   "absolute target ignores base",
			base:   "http://127.0.0.1/private/",
			target: "HTTPS://Public.Example/path#section",
			want:   "https://public.example/path",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := linkRequestURLWithBase(tt.target, tt.base)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("resolved URI: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinkRequestURLBlocksCatalogBase(t *testing.T) {
	for _, base := range []string{
		"http://127.0.0.1/private/",
		"file:///tmp/private/",
		"https://user:password@example.com/private/",
	} {
		if _, err := linkRequestURLWithBase("child", base); !errors.Is(err, errLinkTargetBlocked) {
			t.Fatalf("base %q: got %v, want blocked target", base, err)
		}
	}
}

func TestValidateLinkIPs(t *testing.T) {
	for _, tt := range []struct {
		name    string
		ips     []net.IPAddr
		wantErr bool
	}{
		{"Public", linkTestIPs("8.8.8.8"), false},
		{"Loopback", linkTestIPs("127.0.0.1"), true},
		{"Private", linkTestIPs("10.0.0.1"), true},
		{"Mixed", linkTestIPs("8.8.8.8", "127.0.0.1"), true},
		{"Shared", linkTestIPs("100.64.0.1"), true},
		{"Unresolved", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLinkIPs("example.com", tt.ips)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate IPs: got %v, want error %t", err, tt.wantErr)
			}
		})
	}
}

func TestLinkDialContextPinsValidatedIP(t *testing.T) {
	var target string
	dial := func(_ context.Context, _, addr string) (net.Conn, error) {
		target = addr
		client, server := net.Pipe()
		t.Cleanup(func() {
			client.Close()
			server.Close()
		})
		return client, nil
	}
	dialContext := linkDialContext(linkTestResolver{ips: linkTestIPs("8.8.8.8")}, dial)
	conn, err := dialContext(t.Context(), "tcp", "example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if target != "8.8.8.8:443" {
		t.Fatalf("dial target: got %q, want 8.8.8.8:443", target)
	}
}

func TestLinkDialContextBlocksPrivateDNS(t *testing.T) {
	called := false
	dial := func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, errors.New("unexpected dial")
	}
	dialContext := linkDialContext(linkTestResolver{ips: linkTestIPs("10.0.0.1")}, dial)
	_, err := dialContext(t.Context(), "tcp", "internal.example:80")
	if !errors.Is(err, errLinkTargetBlocked) || !strings.Contains(err.Error(), "disallowed address") {
		t.Fatalf("got %v, want disallowed address error", err)
	}
	if called {
		t.Fatal("dial called for a blocked address")
	}
}

func TestLinkRedirect(t *testing.T) {
	valid, err := http.NewRequest(http.MethodGet, "https://example.com/path", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := linkRedirect(valid, nil); err != nil {
		t.Fatal(err)
	}
	private, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/path", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := linkRedirect(private, nil); err == nil {
		t.Fatal("expected private redirect rejection")
	}
	via := make([]*http.Request, maxLinkRedirects)
	if err := linkRedirect(valid, via); err == nil {
		t.Fatal("expected redirect limit rejection")
	}
}

func TestLinkHTTPClientUsesSafeDefaults(t *testing.T) {
	client := linkHTTPClient(0)
	if client.Timeout != defaultLinkHTTPTimeout {
		t.Fatalf("client timeout: got %v, want %v", client.Timeout, defaultLinkHTTPTimeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type: %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("link transport must not inherit proxy settings")
	}
	if transport.ResponseHeaderTimeout != defaultLinkHTTPTimeout {
		t.Fatalf("response header timeout: got %v, want %v", transport.ResponseHeaderTimeout, defaultLinkHTTPTimeout)
	}
	if transport.MaxResponseHeaderBytes != maxLinkResponseHeaderBytes {
		t.Fatalf(
			"response header limit: got %d, want %d",
			transport.MaxResponseHeaderBytes,
			maxLinkResponseHeaderBytes,
		)
	}
	if transport.MaxConnsPerHost != maxLinkConnectionsPerHost ||
		transport.MaxIdleConns != maxLinkIdleConnections ||
		transport.MaxIdleConnsPerHost != maxLinkIdleConnectionsPerHost {
		t.Fatalf(
			"connection limits: got total=%d idle=%d idle-per-host=%d",
			transport.MaxConnsPerHost,
			transport.MaxIdleConns,
			transport.MaxIdleConnsPerHost,
		)
	}
	if !transport.DisableCompression {
		t.Fatal("link transport enables automatic response decompression")
	}
	configured := linkHTTPClient(3 * time.Second)
	if configured.Timeout != 3*time.Second {
		t.Fatalf("configured timeout: got %v, want 3s", configured.Timeout)
	}
}

func readLinkPipeRequest(conn net.Conn) error {
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return err
	}
	return req.Body.Close()
}

func TestLinkHTTPTransportRejectsOversizedResponseHeaders(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	transport := linkHTTPTransport(time.Second, func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	})
	t.Cleanup(func() {
		transport.CloseIdleConnections()
		clientConn.Close()
		serverConn.Close()
	})

	serverDone := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		if err := readLinkPipeRequest(serverConn); err != nil {
			serverDone <- err
			return
		}
		header := strings.Repeat("a", maxLinkResponseHeaderBytes+8192)
		fmt.Fprintf(serverConn, "HTTP/1.1 200 OK\r\nX-Oversized: %s\r\nContent-Length: 0\r\n\r\n", header)
		serverDone <- nil
	}()

	res, err := (&http.Client{Transport: transport}).Get("http://example.com/")
	if res != nil {
		res.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "response headers exceeded") {
		t.Fatalf("got %v, want response header limit error", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("serve oversized response: %v", err)
	}
}

func TestLinkHTTPTransportTimesOutSlowResponseHeaders(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	const timeout = 25 * time.Millisecond
	transport := linkHTTPTransport(timeout, func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	})
	releaseServer := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		if err := readLinkPipeRequest(serverConn); err != nil {
			serverDone <- err
			return
		}
		<-releaseServer
		serverDone <- nil
	}()

	res, err := (&http.Client{Transport: transport}).Get("http://example.com/")
	if res != nil {
		res.Body.Close()
	}
	close(releaseServer)
	transport.CloseIdleConnections()
	clientConn.Close()
	if serverErr := <-serverDone; serverErr != nil {
		t.Fatalf("serve slow response: %v", serverErr)
	}
	var timeoutErr net.Error
	if !errors.As(err, &timeoutErr) || !timeoutErr.Timeout() {
		t.Fatalf("got %v, want response header timeout", err)
	}
}

type countingLinkTransport struct {
	calls int
}

func (tr *countingLinkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return nil, errors.New("unexpected request")
}

type successfulLinkTransport struct {
	calls int
}

func (tr *successfulLinkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}, nil
}

func TestLogURISuccessesRequiresInfoLogger(t *testing.T) {
	var output bytes.Buffer
	pdfcpuLog.SetCLILogger(stdlog.New(&output, "", 0))
	pdfcpuLog.SetInfoLogger(nil)
	t.Cleanup(func() {
		pdfcpuLog.SetCLILogger(nil)
		pdfcpuLog.SetInfoLogger(nil)
	})
	xRefTable := &model.XRefTable{URIs: map[int]map[string]string{1: {
		"https://example.com/ok": "",
		"https://example.com/no": "404",
	}}}

	logURISuccesses(xRefTable, []int{1})
	if output.Len() != 0 {
		t.Fatalf("normal output: %q", output.String())
	}

	pdfcpuLog.SetInfoLogger(stdlog.New(io.Discard, "", 0))
	logURISuccesses(xRefTable, []int{1})
	want := "pdfcpu checked: page 1: https://example.com/ok - HTTP status 200\n"
	if got := output.String(); got != want {
		t.Fatalf("verbose output: got %q, want %q", got, want)
	}
}

func TestCheckLinksSkipsMalformedTargetMetadata(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	const uri = "https://example.com/unchecked"
	ctx.URIs = map[int]map[string]string{1: {uri: "malformed"}}
	tr := &countingLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if !failed || tr.calls != 0 || ctx.URIs[1][uri] != "k" {
		t.Fatalf("failed=%t calls=%d status=%q", failed, tr.calls, ctx.URIs[1][uri])
	}
	notices := ctx.ValidationReport().Notices()
	if len(notices) != 1 || !strings.Contains(notices[0].Message, "invalid link target metadata") {
		t.Fatalf("notices: %+v", notices)
	}
}

type recordingLinkTransport struct {
	calls int
	urls  []string
}

func (tr *recordingLinkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.calls++
	tr.urls = append(tr.urls, req.URL.String())
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}, nil
}

func TestURIReferenceTargetsAreClassifiedWithoutTransport(t *testing.T) {
	for _, offline := range []bool{false, true} {
		t.Run(fmt.Sprintf("offline=%t", offline), func(t *testing.T) {
			ctx := externalReferenceContext(t)
			page := ctx.CurPage
			for _, uri := range []string{
				"http://127.0.0.1/base/", "http://[::1", "https://example.com/base/",
			} {
				setLinkTestTarget(
					ctx.XRefTable, page, uri,
					linkTarget{kind: linkTargetURIReference, source: linkSourceMediaClipBaseURL},
				)
			}
			tr := &countingLinkTransport{}
			var failed bool
			var err error
			if offline {
				failed, err = checkLinksOffline(t.Context(), ctx.XRefTable, []int{page})
			} else {
				failed, err = checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{page})
			}
			if err != nil {
				t.Fatal(err)
			}
			if !failed {
				t.Fatal("URI reference findings were not reported")
			}
			if tr.calls != 0 {
				t.Fatalf("HTTP requests: got %d, want 0", tr.calls)
			}

			wantStatuses := map[string]string{
				"http://127.0.0.1/base/":    "b",
				"http://[::1":               "i",
				"https://example.com/base/": "k",
			}
			for uri, want := range wantStatuses {
				if got := ctx.URIs[page][uri]; got != want {
					t.Fatalf("URI %q: got status %q, want %q", uri, got, want)
				}
			}
		})
	}
}

func checkURIReferenceNotices(t *testing.T, mode int, notices []model.ValidationNotice) {
	t.Helper()
	if mode == model.ValidationStrict && len(notices) != 0 {
		t.Fatalf("strict notices: %+v", notices)
	}
	if mode == model.ValidationRelaxed {
		if len(notices) != 1 || notices[0].Severity != model.NoticeSeverityWarning ||
			notices[0].Disposition != model.NoticeSkipped ||
			notices[0].Message != "page 3: https://example.com/media/ - MediaClip base URL is not fetched" {
			t.Fatalf("relaxed notices: %+v", notices)
		}
	}
}

func testURIReferenceReportingMode(t *testing.T, mode int) {
	ctx := externalReferenceContext(t)
	ctx.XRefTable.ValidationMode = mode
	page := ctx.CurPage
	target := "https://example.com/media/"
	setLinkTestTarget(
		ctx.XRefTable, page, target,
		linkTarget{kind: linkTargetURIReference, source: linkSourceMediaClipBaseURL},
	)
	tr := &countingLinkTransport{}
	err := checkForBrokenLinksUsing(t.Context(), ctx, http.Client{Transport: tr})
	if mode == model.ValidationStrict && !errors.Is(err, errLinkVerification) {
		t.Fatalf("strict mode: got %v, want link verification error", err)
	}
	if mode == model.ValidationRelaxed && err != nil {
		t.Fatalf("relaxed mode: %v", err)
	}
	if tr.calls != 0 {
		t.Fatalf("mode %d: HTTP requests: got %d, want 0", mode, tr.calls)
	}
	if got := ctx.URIs[page][target]; got != "k" {
		t.Fatalf("mode %d: got status %q, want k", mode, got)
	}
	checkURIReferenceNotices(t, mode, ctx.ValidationReport().Notices())
}

func TestURIReferenceReportingModes(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		testURIReferenceReportingMode(t, mode)
	}
}

func TestURIActionUsesCatalogBaseForLinkCheck(t *testing.T) {
	ctx := externalReferenceContext(t)
	root := types.Dict{"URI": types.Dict{"Base": types.StringLiteral("https://example.com/docs/current/")}}
	ctx.RootDict = root
	if err := validateURI(ctx.XRefTable, root, OPTIONAL, model.V11); err != nil {
		t.Fatal(err)
	}
	uri := "../guide#intro"
	action := types.Dict{"S": types.Name("URI"), "URI": types.StringLiteral(uri)}
	if err := validateActionDictObject(t.Context(), ctx.XRefTable, action, action, "URI action"); err != nil {
		t.Fatal(err)
	}
	tr := &recordingLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{ctx.CurPage})
	if err != nil {
		t.Fatal(err)
	}
	if failed || tr.calls != 1 || len(tr.urls) != 1 || tr.urls[0] != "https://example.com/docs/guide" {
		t.Fatalf("relative URI check: failed=%t, calls=%d, URLs=%v", failed, tr.calls, tr.urls)
	}
	if got := ctx.URIs[ctx.CurPage][uri]; got != "" {
		t.Fatalf("link status: got %q, want success", got)
	}
}

func TestURIActionBlocksPrivateCatalogBaseBeforeTransport(t *testing.T) {
	ctx := externalReferenceContext(t)
	root := types.Dict{"URI": types.Dict{"Base": types.StringLiteral("http://127.0.0.1/private/")}}
	ctx.RootDict = root
	if err := validateURI(ctx.XRefTable, root, OPTIONAL, model.V11); err != nil {
		t.Fatal(err)
	}
	uri := "child"
	action := types.Dict{"S": types.Name("URI"), "URI": types.StringLiteral(uri)}
	if err := validateActionDictObject(t.Context(), ctx.XRefTable, action, action, "URI action"); err != nil {
		t.Fatal(err)
	}
	tr := &countingLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{ctx.CurPage})
	if err != nil {
		t.Fatal(err)
	}
	if !failed || tr.calls != 0 || ctx.URIs[ctx.CurPage][uri] != "b" {
		t.Fatalf("private base: failed=%t, calls=%d, status=%q", failed, tr.calls, ctx.URIs[ctx.CurPage][uri])
	}
}

func TestCheckLinksRecordsEveryURIWhenRequestLimitIsReached(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	ctx.URIs = map[int]map[string]string{
		1: {
			"https://example.com/01":  "",
			"https://example.com/02":  "",
			"https://example.com/03":  "",
			"https://example.com/04":  "",
			"mailto:user@example.com": "",
		},
	}
	encodeLinkTestURIs(ctx.XRefTable)
	tr := &successfulLinkTransport{}
	httpErr, err := checkLinksWithLimit(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{1}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !httpErr {
		t.Fatal("request limit was not reported")
	}
	if tr.calls != 2 {
		t.Fatalf("HTTP requests: got %d, want 2", tr.calls)
	}

	for uri, want := range map[string]string{
		"https://example.com/01":  "",
		"https://example.com/02":  "",
		"https://example.com/03":  "k",
		"https://example.com/04":  "k",
		"mailto:user@example.com": "k",
	} {
		if got := ctx.URIs[1][uri]; got != want {
			t.Fatalf("URI %q: got status %q, want %q", uri, got, want)
		}
	}
	notices := ctx.ValidationReport().Notices()
	if len(notices) != 3 {
		t.Fatalf("validation notices: got %d, want 3: %+v", len(notices), notices)
	}
	if !strings.Contains(notices[0].Message, "HTTP request limit reached (2)") ||
		!strings.Contains(notices[1].Message, "HTTP request limit reached (2)") {
		t.Fatalf("request-limit notices: %+v", notices)
	}
}

func TestCheckLinksReusesNormalizedTargetAcrossPages(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	ctx.URIs = map[int]map[string]string{
		1: {"HTTPS://EXAMPLE.COM/path#first": ""},
		2: {"https://example.com/path#second": ""},
		3: {"https://example.com/other": ""},
	}
	encodeLinkTestURIs(ctx.XRefTable)
	tr := &successfulLinkTransport{}
	httpErr, err := checkLinksWithLimit(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{1, 2, 3}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !httpErr {
		t.Fatal("expected unique target beyond request limit to be reported")
	}
	if tr.calls != 1 {
		t.Fatalf("HTTP requests: got %d, want 1", tr.calls)
	}
	if got := ctx.URIs[1]["HTTPS://EXAMPLE.COM/path#first"]; got != "" {
		t.Fatalf("first normalized target: got status %q, want success", got)
	}
	if got := ctx.URIs[2]["https://example.com/path#second"]; got != "" {
		t.Fatalf("second normalized target: got status %q, want success", got)
	}
	if got := ctx.URIs[3]["https://example.com/other"]; got != "k" {
		t.Fatalf("request-limited target: got status %q, want k", got)
	}
	if notices := ctx.ValidationReport().Notices(); len(notices) != 1 {
		t.Fatalf("validation notices: got %d, want 1: %+v", len(notices), notices)
	}
}

type redirectingLinkTransport struct {
	calls int
}

func (tr *redirectingLinkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.calls++
	status := http.StatusOK
	header := http.Header{}
	if req.URL.Path == "/start" {
		status = http.StatusFound
		header.Set("Location", "https://example.com/final")
	}
	return &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     header,
		Request:    req,
	}, nil
}

func TestCheckLinksCountsRedirectsAgainstRequestLimit(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	const uri = "https://example.com/start"
	ctx.URIs = map[int]map[string]string{1: {uri: ""}}
	encodeLinkTestURIs(ctx.XRefTable)
	tr := &redirectingLinkTransport{}
	client := http.Client{Transport: tr, CheckRedirect: linkRedirect}
	httpErr, err := checkLinksWithLimit(t.Context(), ctx.XRefTable, client, []int{1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !httpErr || tr.calls != 1 {
		t.Fatalf("redirect budget: failure=%t HTTP requests=%d, want true and 1", httpErr, tr.calls)
	}
	if got := ctx.URIs[1][uri]; got != "k" {
		t.Fatalf("redirect result: got status %q, want k", got)
	}
	notices := ctx.ValidationReport().Notices()
	if len(notices) != 1 || !strings.Contains(notices[0].Message, "HTTP request limit reached (1)") {
		t.Fatalf("redirect notices: %+v", notices)
	}
}

func TestCheckForBrokenLinksStrictReportsRequestLimitAfterAllLinks(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	ctx.URIs = map[int]map[string]string{1: {}}
	for i := range maxLinkHTTPRequests + 1 {
		uri := fmt.Sprintf("https://example.com/%03d", i)
		setLinkTestTarget(ctx.XRefTable, 1, uri, linkTarget{kind: linkTargetURI, source: linkSourceURIAction})
	}
	tr := &successfulLinkTransport{}
	err = checkForBrokenLinksUsing(t.Context(), ctx, http.Client{Transport: tr})
	if !errors.Is(err, errLinkVerification) {
		t.Fatalf("got %v, want link verification error", err)
	}
	if tr.calls != maxLinkHTTPRequests {
		t.Fatalf("HTTP requests: got %d, want %d", tr.calls, maxLinkHTTPRequests)
	}
	lastURI := fmt.Sprintf("https://example.com/%03d", maxLinkHTTPRequests)
	if got := ctx.URIs[1][lastURI]; got != "k" {
		t.Fatalf("last link status: got %q, want k", got)
	}
}

func TestCheckLinksRejectsPrivateLiteralBeforeRequest(t *testing.T) {
	tr := &countingLinkTransport{}
	xRefTable := &model.XRefTable{URIs: map[int]map[string]string{
		1: {"http://127.0.0.1/private": ""},
	}}
	encodeLinkTestURIs(xRefTable)
	httpErr, err := checkLinks(t.Context(), xRefTable, http.Client{Transport: tr}, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if !httpErr {
		t.Fatal("expected private URL to be reported")
	}
	if tr.calls != 0 {
		t.Fatalf("HTTP requests: got %d, want 0", tr.calls)
	}
	if got := xRefTable.URIs[1]["http://127.0.0.1/private"]; got != "b" {
		t.Fatalf("link status: got %q, want b", got)
	}
}

type linkResultTransport struct {
	calls int
}

func (tr *linkResultTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.calls++
	switch req.URL.Path {
	case "/ok":
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	case "/missing":
		return &http.Response{
			Status:     "404 Not Found",
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	case "/timeout":
		return nil, linkResultTimeoutError{}
	default:
		return nil, errors.New("connection failed")
	}
}

type linkResultTimeoutError struct{}

func (linkResultTimeoutError) Error() string { return "request timed out" }

func (linkResultTimeoutError) Timeout() bool { return true }

type trackedLinkBody struct {
	transport *bodyTrackingLinkTransport
	closed    bool
}

func (b *trackedLinkBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (b *trackedLinkBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	b.transport.openBodies--
	b.transport.closedBodies++
	return nil
}

type bodyTrackingLinkTransport struct {
	openBodies   int
	maxOpen      int
	closedBodies int
}

func (tr *bodyTrackingLinkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.openBodies++
	if tr.openBodies > tr.maxOpen {
		tr.maxOpen = tr.openBodies
	}
	status := http.StatusOK
	if req.URL.Path == "/missing" {
		status = http.StatusNotFound
	}
	return &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		Body:       &trackedLinkBody{transport: tr},
		Header:     http.Header{},
	}, nil
}

func TestCheckLinksClosesEachResponseBodyImmediately(t *testing.T) {
	xRefTable := &model.XRefTable{URIs: map[int]map[string]string{
		1: {
			"https://example.com/first":   "",
			"https://example.com/missing": "",
			"https://example.com/third":   "",
		},
	}}
	encodeLinkTestURIs(xRefTable)
	tr := &bodyTrackingLinkTransport{}
	httpErr, err := checkLinks(t.Context(), xRefTable, http.Client{Transport: tr}, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if !httpErr {
		t.Fatal("expected non-200 response to be reported")
	}
	if tr.maxOpen != 1 {
		t.Fatalf("simultaneously open response bodies: got %d, want 1", tr.maxOpen)
	}
	if tr.openBodies != 0 || tr.closedBodies != 3 {
		t.Fatalf("response bodies: got %d open and %d closed, want 0 open and 3 closed", tr.openBodies, tr.closedBodies)
	}
}

type cancelingLinkTransport struct {
	cancel context.CancelFunc
	calls  int
}

func (tr *cancelingLinkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.calls++
	tr.cancel()
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestCheckLinksCancelsInFlightRequest(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	tr := &cancelingLinkTransport{cancel: cancel}
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	ctx.URIs = map[int]map[string]string{
		1: {
			"https://example.com/first":  "",
			"https://example.com/second": "",
		},
	}
	encodeLinkTestURIs(ctx.XRefTable)
	err = checkForBrokenLinksUsing(c, ctx, http.Client{Transport: tr})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context cancellation", err)
	}
	if notices := ctx.ValidationReport().Notices(); len(notices) != 0 {
		t.Fatalf("canceled request produced link notices: %+v", notices)
	}
	if tr.calls != 1 {
		t.Fatalf("HTTP requests after cancellation: got %d, want 1", tr.calls)
	}
}

type cancelOnCloseLinkBody struct {
	cancel context.CancelFunc
}

func (b cancelOnCloseLinkBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (b cancelOnCloseLinkBody) Close() error {
	b.cancel()
	return nil
}

type cancelAfterResponseTransport struct {
	cancel context.CancelFunc
	calls  int
}

func (tr *cancelAfterResponseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.calls++
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       cancelOnCloseLinkBody{cancel: tr.cancel},
		Header:     http.Header{},
	}, nil
}

func TestCheckLinksStopsBetweenRequestsAfterCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	tr := &cancelAfterResponseTransport{cancel: cancel}
	xRefTable := &model.XRefTable{URIs: map[int]map[string]string{
		1: {
			"https://example.com/first":  "",
			"https://example.com/second": "",
		},
	}}
	encodeLinkTestURIs(xRefTable)
	httpErr, err := checkLinks(c, xRefTable, http.Client{Transport: tr}, []int{1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context cancellation", err)
	}
	if httpErr {
		t.Fatal("cancellation after a successful request was recorded as a link failure")
	}
	if tr.calls != 1 {
		t.Fatalf("HTTP requests after cancellation: got %d, want 1", tr.calls)
	}
}

func TestCheckLinksHonorsExpiredDeadlineBeforeRequest(t *testing.T) {
	c, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	tr := &countingLinkTransport{}
	xRefTable := &model.XRefTable{URIs: map[int]map[string]string{
		1: {"https://example.com/unchecked": ""},
	}}
	encodeLinkTestURIs(xRefTable)
	httpErr, err := checkLinks(c, xRefTable, http.Client{Transport: tr}, []int{1})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want deadline exceeded", err)
	}
	if httpErr {
		t.Fatal("expired deadline was recorded as a link failure")
	}
	if tr.calls != 0 {
		t.Fatalf("HTTP requests after expired deadline: got %d, want 0", tr.calls)
	}
}

func TestCheckLinksRecordsResultsInOrder(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	ctx.URIs = map[int]map[string]string{
		2: {
			"file:///tmp/data":            "",
			"http://[::1":                 "",
			"https://example.com/missing": "",
			"https://example.com/network": "",
			"https://example.com/timeout": "",
			"mailto:user@example.com":     "",
		},
		1: {
			"http://127.0.0.1/private":    "",
			"https://example.com/missing": "",
			"https://example.com/ok":      "",
		},
	}
	encodeLinkTestURIs(ctx.XRefTable)
	tr := &linkResultTransport{}
	err = checkForBrokenLinksUsing(t.Context(), ctx, http.Client{Transport: tr})
	if !errors.Is(err, errLinkVerification) {
		t.Fatalf("got %v, want link verification error", err)
	}

	want := map[int]map[string]string{
		1: {
			"http://127.0.0.1/private":    "b",
			"https://example.com/missing": "404",
			"https://example.com/ok":      "",
		},
		2: {
			"file:///tmp/data":            "b",
			"http://[::1":                 "i",
			"https://example.com/missing": "404",
			"https://example.com/network": "s",
			"https://example.com/timeout": "t",
			"mailto:user@example.com":     "k",
		},
	}
	for page, uris := range want {
		for uri, status := range uris {
			if got := ctx.URIs[page][uri]; got != status {
				t.Fatalf("page %d URI %q: got status %q, want %q", page, uri, got, status)
			}
		}
	}
	if tr.calls != 4 {
		t.Fatalf("HTTP requests: got %d, want 4", tr.calls)
	}
	if notices := ctx.ValidationReport().Notices(); len(notices) != 0 {
		t.Fatalf("strict link notices: got %+v, want none", notices)
	}
}

func TestCheckLinksRelaxedAddsSkippedNotices(t *testing.T) {
	var cliOutput bytes.Buffer
	pdfcpuLog.SetCLILogger(stdlog.New(&cliOutput, "", 0))
	t.Cleanup(func() { pdfcpuLog.SetCLILogger(nil) })

	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidateLinks = true
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	ctx.URIs = map[int]map[string]string{
		0: {"http://127.0.0.1/document": ""},
		3: {
			"http://127.0.0.1/private":    "",
			"https://example.com/missing": "",
			"mailto:user@example.com":     "",
		},
	}
	encodeLinkTestURIs(ctx.XRefTable)
	if err := checkForBrokenLinksUsing(t.Context(), ctx, http.Client{Transport: &linkResultTransport{}}); err != nil {
		t.Fatal(err)
	}

	report := ctx.ValidationReport()
	notices := report.Notices()
	if len(notices) != 4 {
		t.Fatalf("report: got %d notices, want 4", len(notices))
	}
	wantMessages := []string{
		"document: http://127.0.0.1/document - blocked by security policy",
		"page 3: http://127.0.0.1/private - blocked by security policy",
		"page 3: https://example.com/missing - HTTP status 404",
		"page 3: mailto:user@example.com - URI scheme is not checked: URL scheme \"mailto\"",
	}
	wantPages := []int{0, 3, 3, 3}
	for i, notice := range notices {
		if notice.Severity != model.NoticeSeverityWarning || notice.Disposition != model.NoticeSkipped ||
			notice.Phase != model.NoticePhaseValidate || notice.PageNumber != wantPages[i] {
			t.Fatalf("notice %d: got %+v", i, notice)
		}
		if notice.Message != wantMessages[i] {
			t.Fatalf("notice %d: got %q, want %q", i, notice.Message, wantMessages[i])
		}
	}
	if output := cliOutput.String(); strings.Contains(output, "Page 3:") || !strings.HasSuffix(output, "\n") {
		t.Fatalf("relaxed CLI progress output: %q", output)
	}
}

func TestURIActionRecordsRepeatedTargetOnEachPage(t *testing.T) {
	uri := "https://example.com/repeated"
	d := types.Dict{"URI": types.StringLiteral(uri)}
	xRefTable := sharedActionXRefTable(10, nil)
	xRefTable.ValidateLinks = true
	xRefTable.URIs = map[int]map[string]string{}
	for _, page := range []int{1, 2} {
		xRefTable.CurPage = page
		if err := validateURIActionDict(xRefTable, d, "URI action"); err != nil {
			t.Fatal(err)
		}
	}
	for _, page := range []int{1, 2} {
		if _, found := xRefTable.URIs[page][uri]; !found {
			t.Fatalf("page %d does not contain repeated URI", page)
		}
	}
}

func TestURIActionRecordsEveryScheme(t *testing.T) {
	xRefTable := sharedActionXRefTable(10, nil)
	xRefTable.ValidateLinks = true
	xRefTable.CurPage = 4
	xRefTable.URIs = map[int]map[string]string{}
	for _, uri := range []string{"HTTPS://example.com/upper", "file:///tmp/data", "mailto:user@example.com"} {
		d := types.Dict{"URI": types.StringLiteral(uri)}
		if err := validateURIActionDict(xRefTable, d, "URI action"); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(xRefTable.URIs[4]); got != 3 {
		t.Fatalf("collected URIs: got %d, want 3: %v", got, xRefTable.URIs[4])
	}
}

func readLinkTestContext(t *testing.T, validateLinks bool, validationMode int) *model.Context {
	t.Helper()
	f, err := os.Open("../../testdata/empty.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	conf := model.NewStatelessConfiguration()
	conf.ValidateLinks = validateLinks
	conf.ValidationMode = validationMode
	ctx, err := pdfcpu.Read(t.Context(), f, conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx.URIs = map[int]map[string]string{1: {"http://127.0.0.1/private": ""}}
	if validateLinks {
		encodeLinkTestURIs(ctx.XRefTable)
	}
	return ctx
}

// TestLinkVerificationFollowsConformance verifies a blocked link does not make a conforming PDF structurally invalid.
func TestLinkVerificationFollowsConformance(t *testing.T) {
	ctx := readLinkTestContext(t, true, model.ValidationStrict)
	err := XRefTable(t.Context(), ctx)
	if !errors.Is(err, errLinkVerification) {
		t.Fatalf("got %v, want link verification error", err)
	}
	if !ctx.XRefTable.Valid {
		t.Fatal("PDF conformance was not recorded before link verification")
	}
	if got := ctx.URIs[1]["http://127.0.0.1/private"]; got != "b" {
		t.Fatalf("link status: got %q, want b", got)
	}
}

// TestPDFValidationDoesNotCheckLinksByDefault verifies ordinary validation remains independent of network links.
func TestPDFValidationDoesNotCheckLinksByDefault(t *testing.T) {
	ctx := readLinkTestContext(t, false, model.ValidationRelaxed)
	if err := XRefTable(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	if !ctx.XRefTable.Valid {
		t.Fatal("PDF was not marked valid")
	}
	if got := ctx.URIs[1]["http://127.0.0.1/private"]; got != "" {
		t.Fatalf("unchecked link status: got %q, want empty", got)
	}
}
