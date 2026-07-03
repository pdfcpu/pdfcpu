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
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestImageBoxRemoteURL(t *testing.T) {
	for _, tt := range []struct {
		name    string
		src     string
		remote  bool
		wantErr bool
	}{
		{"file", "testdata/logo.png", false, false},
		{"publicHTTP", "http://example.com/logo.png", true, false},
		{"publicHTTPS", "https://example.com/logo.png", true, false},
		{"credentials", "https://user:pass@example.com/logo.png", true, true},
		{"localhost", "http://127.0.0.1/logo.png", true, true},
		{"metadata", "http://169.254.169.254/latest/meta-data", true, true},
		{"private", "http://10.0.0.1/logo.png", true, true},
		{"ipv6Loopback", "http://[::1]/logo.png", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, gotRemote, err := imageBoxRemoteURL(tt.src)
			if gotRemote != tt.remote {
				t.Fatalf("remote: got %t want %t", gotRemote, tt.remote)
			}
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("err: got %t want %t: %v", gotErr, tt.wantErr, err)
			}
		})
	}
}

func TestImageBoxRedirect(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	req := &http.Request{URL: u}
	if err := imageBoxRedirect(req, nil); err == nil {
		t.Fatal("expected redirect to private address to fail")
	}
}

func TestRejectImageBoxIPs(t *testing.T) {
	ips := []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}
	if err := rejectImageBoxIPs("example.com", ips); err != nil {
		t.Fatalf("rejectImageBoxIPs: %v", err)
	}
	ips = append(ips, net.IPAddr{IP: net.ParseIP("127.0.0.1")})
	if err := rejectImageBoxIPs("example.com", ips); err == nil {
		t.Fatal("expected mixed public/private DNS results to fail")
	}
}
