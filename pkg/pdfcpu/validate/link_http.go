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
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pdfcpu/pdfcpu/internal/netutil"
)

const (
	defaultLinkHTTPTimeout        = 10 * time.Second
	maxLinkRedirects              = 10
	maxLinkResponseHeaderBytes    = 1 << 20
	maxLinkConnectionsPerHost     = 1
	maxLinkIdleConnections        = 1
	maxLinkIdleConnectionsPerHost = 1
	// maxLinkHTTPRequests caps all transport attempts, including redirects, for one document.
	maxLinkHTTPRequests = 100
)

var (
	errLinkRequestLimit  = errors.New("HTTP request limit reached")
	errLinkSchemeSkipped = errors.New("URI scheme is not checked")
	errLinkTargetBlocked = errors.New("link target blocked by security policy")
)

type linkResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type linkDialer func(context.Context, string, string) (net.Conn, error)

type linkRequestBudgetTransport struct {
	mu        sync.Mutex
	remaining int
	transport http.RoundTripper
}

func newLinkRequestBudgetTransport(transport http.RoundTripper, limit int) *linkRequestBudgetTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &linkRequestBudgetTransport{remaining: limit, transport: transport}
}

func (tr *linkRequestBudgetTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.mu.Lock()
	if tr.remaining <= 0 {
		tr.mu.Unlock()
		return nil, errLinkRequestLimit
	}
	tr.remaining--
	tr.mu.Unlock()
	return tr.transport.RoundTrip(req)
}

func (tr *linkRequestBudgetTransport) CloseIdleConnections() {
	if transport, ok := tr.transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}

func validateLinkURL(u *url.URL) error {
	if u == nil {
		return errors.New("missing link URL")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: URL scheme must be http or https: %s", errLinkTargetBlocked, u.Redacted())
	}
	if u.User != nil {
		return fmt.Errorf("%w: URL must not contain credentials: %s", errLinkTargetBlocked, u.Redacted())
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("link URL missing host: %s", u.Redacted())
	}
	if ip := net.ParseIP(host); ip != nil && netutil.BlockedIP(ip) {
		return fmt.Errorf("%w: URL resolves to disallowed address: %s", errLinkTargetBlocked, host)
	}
	return nil
}

func blockedLinkScheme(scheme string) bool {
	switch scheme {
	case "data", "file", "javascript", "vbscript":
		return true
	}
	return false
}

func normalizeLinkRequestURL(u *url.URL) (string, error) {
	scheme := strings.ToLower(u.Scheme)
	if blockedLinkScheme(scheme) {
		return "", fmt.Errorf("%w: URL scheme %q", errLinkTargetBlocked, scheme)
	}
	if scheme == "" {
		return "", fmt.Errorf("%w: relative URI", errLinkSchemeSkipped)
	}
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: URL scheme %q", errLinkSchemeSkipped, scheme)
	}
	u.Scheme = scheme
	if err := validateLinkURL(u); err != nil {
		return "", err
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawFragment = ""
	return u.String(), nil
}

func linkRequestURLWithBase(s, base string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("parse link URL: %w", err)
	}
	if u.IsAbs() || base == "" {
		return normalizeLinkRequestURL(u)
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse catalog base URI: %w", err)
	}
	if _, err := normalizeLinkRequestURL(baseURL); err != nil {
		return "", fmt.Errorf("catalog base URI: %w", err)
	}
	return normalizeLinkRequestURL(baseURL.ResolveReference(u))
}

func linkRequestURL(s string) (string, error) {
	return linkRequestURLWithBase(s, "")
}

func redactedLinkURI(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		// A malformed URI may contain credentials that cannot be reliably isolated.
		return "[invalid URI]"
	}
	if u.User == nil {
		return s
	}
	return u.Redacted()
}

func validateLinkIPs(host string, ips []net.IPAddr) error {
	if len(ips) == 0 {
		return fmt.Errorf("link URL host does not resolve: %s", host)
	}
	for _, ip := range ips {
		if netutil.BlockedIP(ip.IP) {
			return fmt.Errorf("%w: URL resolves to disallowed address: %s", errLinkTargetBlocked, host)
		}
	}
	return nil
}

func linkDialContext(resolver linkResolver, dial linkDialer) func(context.Context, string, string) (net.Conn, error) {
	return func(c context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupIPAddr(c, host)
		if err != nil {
			return nil, err
		}
		if err := validateLinkIPs(host, ips); err != nil {
			return nil, err
		}

		var errs []error
		for _, ip := range ips {
			target := net.JoinHostPort(ip.IP.String(), port)
			conn, err := dial(c, network, target)
			if err == nil {
				return conn, nil
			}
			errs = append(errs, err)
		}
		return nil, errors.Join(errs...)
	}
}

func linkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxLinkRedirects {
		return fmt.Errorf("stopped after %d redirects", maxLinkRedirects)
	}
	return validateLinkURL(req.URL)
}

func linkHTTPTransport(timeout time.Duration, dialContext func(context.Context, string, string) (net.Conn, error)) *http.Transport {
	return &http.Transport{
		Proxy:                  nil,
		DialContext:            dialContext,
		ForceAttemptHTTP2:      true,
		DisableCompression:     true,
		MaxIdleConns:           maxLinkIdleConnections,
		MaxIdleConnsPerHost:    maxLinkIdleConnectionsPerHost,
		MaxConnsPerHost:        maxLinkConnectionsPerHost,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    timeout,
		ExpectContinueTimeout:  time.Second,
		ResponseHeaderTimeout:  timeout,
		MaxResponseHeaderBytes: maxLinkResponseHeaderBytes,
	}
}

func linkHTTPClient(timeout time.Duration) http.Client {
	if timeout <= 0 {
		timeout = defaultLinkHTTPTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := linkHTTPTransport(timeout, linkDialContext(net.DefaultResolver, dialer.DialContext))
	return http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: linkRedirect,
	}
}
