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

package sign

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRevocationHTTPTimeout = 10 * time.Second
	maxRevocationRedirects       = 10
)

type revocationResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type revocationDialer func(context.Context, string, string) (net.Conn, error)

func normalizeRevocationHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	return strings.TrimSuffix(host, ".")
}

func allowedRevocationHostSet(hosts []string) map[string]bool {
	allowed := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		if host = normalizeRevocationHost(host); host != "" {
			allowed[host] = true
		}
	}
	return allowed
}

func validateRevocationURL(u *url.URL) error {
	if u == nil {
		return errors.New("missing revocation URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("revocation URL scheme must be http or https: %s", u.Redacted())
	}
	if u.User != nil {
		return fmt.Errorf("revocation URL must not contain credentials: %s", u.Redacted())
	}
	if u.Hostname() == "" {
		return fmt.Errorf("revocation URL missing host: %s", u.Redacted())
	}
	return nil
}

func validateRevocationURLString(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("parse revocation URL: %w", err)
	}
	return validateRevocationURL(u)
}

func revocationBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func validateRevocationIPs(host string, ips []net.IPAddr, allowed map[string]bool) error {
	if len(ips) == 0 {
		return fmt.Errorf("revocation URL host does not resolve: %s", host)
	}
	if allowed[normalizeRevocationHost(host)] {
		return nil
	}
	for _, ip := range ips {
		if revocationBlockedIP(ip.IP) {
			return fmt.Errorf("revocation URL resolves to disallowed address: %s", host)
		}
	}
	return nil
}

func revocationDialContext(
	resolver revocationResolver,
	dial revocationDialer,
	allowed map[string]bool,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if err := validateRevocationIPs(host, ips, allowed); err != nil {
			return nil, err
		}

		var errs []error
		for _, ip := range ips {
			target := net.JoinHostPort(ip.IP.String(), port)
			conn, err := dial(ctx, network, target)
			if err == nil {
				return conn, nil
			}
			errs = append(errs, err)
		}
		return nil, errors.Join(errs...)
	}
}

func revocationRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRevocationRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRevocationRedirects)
	}
	return validateRevocationURL(req.URL)
}

func revocationHTTPClient(timeout time.Duration, allowedHosts []string) *http.Client {
	if timeout <= 0 {
		timeout = defaultRevocationHTTPTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           revocationDialContext(net.DefaultResolver, dialer.DialContext, allowedRevocationHostSet(allowedHosts)),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: revocationRedirect,
	}
}
