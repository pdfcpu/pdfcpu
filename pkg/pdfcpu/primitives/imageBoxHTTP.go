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
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

func imageBoxRemoteURL(s string) (*url.URL, bool, error) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		return nil, false, nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false, nil
	}
	if err := validateImageBoxRemoteURL(u); err != nil {
		return nil, true, err
	}
	return u, true, nil
}

func validateImageBoxRemoteURL(u *url.URL) error {
	if u.User != nil {
		return fmt.Errorf("image URL must not contain credentials: %s", u.Redacted())
	}
	if u.Hostname() == "" {
		return fmt.Errorf("image URL missing host: %s", u.Redacted())
	}
	if err := rejectPrivateImageBoxHost(u.Hostname()); err != nil {
		return err
	}
	return nil
}

func rejectPrivateImageBoxHost(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		return rejectPrivateImageBoxIP(host, ip)
	}
	return nil
}

func rejectPrivateImageBoxIP(host string, ip net.IP) error {
	if imageBoxBlockedIP(ip) {
		return fmt.Errorf("image URL resolves to disallowed address: %s", host)
	}
	return nil
}

func imageBoxBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func (pdf *PDF) imageBoxHTTPClient() *http.Client {
	if pdf.httpClient != nil {
		return pdf.httpClient
	}
	pdf.httpClient = &http.Client{
		Transport:     imageBoxTransport(time.Duration(pdf.Timeout) * time.Second),
		Timeout:       time.Duration(pdf.Timeout) * time.Second,
		CheckRedirect: imageBoxRedirect,
	}
	return pdf.httpClient
}

func imageBoxTransport(timeout time.Duration) *http.Transport {
	dialer := &net.Dialer{Timeout: timeout}
	return &http.Transport{
		DialContext:           imageBoxDialContext(dialer),
		Proxy:                 nil,
		ResponseHeaderTimeout: timeout,
		TLSHandshakeTimeout:   timeout,
	}
}

func imageBoxRedirect(req *http.Request, via []*http.Request) error {
	return validateImageBoxRemoteURL(req.URL)
}

func imageBoxDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if err := rejectImageBoxIPs(host, ips); err != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

func rejectImageBoxIPs(host string, ips []net.IPAddr) error {
	if len(ips) == 0 {
		return fmt.Errorf("image URL host does not resolve: %s", host)
	}
	for _, ip := range ips {
		if err := rejectPrivateImageBoxIP(host, ip.IP); err != nil {
			return err
		}
	}
	return nil
}
