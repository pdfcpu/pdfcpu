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

// Package netutil provides address checks for outbound HTTP requests.
package netutil

import (
	"net"
	"net/netip"
)

// These ranges are blocked in addition to the standard private and local ranges.
var blockedPrefixes = [...]netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), // Shared address space (RFC 6598).
	netip.MustParsePrefix("2002::/16"),     // 6to4 tunneling (RFC 3056).
	netip.MustParsePrefix("2001::/32"),     // Teredo tunneling (RFC 4380).
	netip.MustParsePrefix("::/96"),         // Deprecated IPv4-compatible addresses (RFC 4291, section 2.5.5.1).
	netip.MustParsePrefix("100::/64"),      // Discard-only addresses (RFC 6666).
}

// The well-known NAT64 prefix has a fixed IPv4 suffix (RFC 6052, section 2.2).
var nat64Prefix = netip.MustParsePrefix("64:ff9b::/96")

// BlockedIP reports whether ip is disallowed for outbound HTTP requests.
// It rejects invalid, private, local and selected special-purpose addresses.
// Tunneling prefixes are blocked conservatively, including their public destinations.
// Standard NAT64 uses the embedded IPv4 policy so public destinations remain usable.
// Network-specific translation and routing cannot be inferred from an IP address.
func BlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	if nat64Prefix.Contains(addr) {
		raw := addr.As16()
		addr = netip.AddrFrom4([4]byte(raw[12:]))
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
