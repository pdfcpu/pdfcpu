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

package netutil

import (
	"net"
	"testing"
)

func TestBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"invalid", true},
		{"8.8.8.8", false},
		{"2606:4700:4700::1111", false},
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"169.254.0.1", true},
		{"224.0.0.1", true},
		{"0.0.0.0", true},
		{"::1", true},
		{"::", true},
		{"fd00::1", true},
		{"fe80::1", true},
		{"ff02::1", true},
		{"::ffff:10.0.0.1", true},
		{"::ffff:8.8.8.8", false},
		{"100.63.255.255", false},
		{"100.64.0.0", true},
		{"100.127.255.255", true},
		{"100.128.0.0", false},
		{"2002::", true},
		{"2002:ffff:ffff:ffff:ffff:ffff:ffff:ffff", true},
		{"2003::", false},
		{"2001::", true},
		{"2001:0:ffff:ffff:ffff:ffff:ffff:ffff", true},
		{"2001:1::", false},
		{"::2", true},
		{"::ffff:ffff", true},
		{"::1:0:0", false},
		{"100::", true},
		{"100::ffff:ffff:ffff:ffff", true},
		{"100:0:0:1::", false},
		{"64:ff9b::a00:1", true},
		{"64:ff9b::7f00:1", true},
		{"64:ff9b::a9fe:1", true},
		{"64:ff9b::6440:1", true},
		{"64:ff9b::e000:1", true},
		{"64:ff9b::", true},
		{"64:ff9b::808:808", false},
		{"64:ff9b:1::", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := BlockedIP(net.ParseIP(tt.ip)); got != tt.blocked {
				t.Fatalf("BlockedIP(%q) = %t, want %t", tt.ip, got, tt.blocked)
			}
		})
	}
}
