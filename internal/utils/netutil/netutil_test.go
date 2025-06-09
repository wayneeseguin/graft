package netutil

import (
	"net"
	"testing"
)

func TestIPAdd(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		offset   int
		expected string
	}{
		// IPv4 tests
		{"IPv4 simple add", "192.168.1.1", 1, "192.168.1.2"},
		{"IPv4 add zero", "192.168.1.1", 0, "192.168.1.1"},
		{"IPv4 add negative", "192.168.1.10", -5, "192.168.1.5"},
		{"IPv4 cross boundary", "192.168.1.250", 10, "192.168.2.4"},
		{"IPv4 cross multiple boundaries", "192.168.1.1", 256, "192.168.2.1"},
		{"IPv4 large offset", "10.0.0.0", 65536, "10.1.0.0"},
		{"IPv4 from zero", "0.0.0.0", 16843009, "1.1.1.1"},
		
		// IPv6 tests
		{"IPv6 simple add", "2001:db8::1", 1, "2001:db8::2"},
		{"IPv6 add zero", "2001:db8::1", 0, "2001:db8::1"},
		{"IPv6 add negative", "2001:db8::10", -5, "2001:db8::b"},
		{"IPv6 cross boundary", "2001:db8::ffff", 1, "2001:db8::1:0"},
		{"IPv6 large offset", "2001:db8::", 65536, "2001:db8::1:0"},
		
		// Edge cases
		{"IPv4 max address", "255.255.255.254", 1, "255.255.255.255"},
		{"IPv4 negative from start", "10.0.0.5", -5, "10.0.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			result := IPAdd(ip, tt.offset)
			if result == nil {
				t.Fatalf("IPAdd returned nil")
			}

			if result.String() != tt.expected {
				t.Errorf("IPAdd(%s, %d) = %s; want %s", tt.ip, tt.offset, result.String(), tt.expected)
			}
		})
	}
}

func TestIPAddWithCIDR(t *testing.T) {
	// Test the use case from op_ips.go
	ipStr := "10.0.0.0/24"
	ip, _, err := net.ParseCIDR(ipStr)
	if err != nil {
		t.Fatalf("Failed to parse CIDR: %v", err)
	}

	// Test adding offset to base IP
	result := IPAdd(ip, 5)
	expected := "10.0.0.5"
	if result.String() != expected {
		t.Errorf("IPAdd with CIDR base IP failed: got %s, want %s", result.String(), expected)
	}
}