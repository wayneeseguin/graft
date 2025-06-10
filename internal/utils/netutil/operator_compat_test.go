package netutil

import (
	"net"
	"testing"
)

// TestOperatorCompatibility verifies our IPAdd matches the behavior expected by op_ips.go
func TestOperatorCompatibility(t *testing.T) {
	// Test 1: The operator masks CIDR IPs before calling IPAdd
	t.Run("Operator masks CIDR", func(t *testing.T) {
		// Operator does: ip = ip.Mask(ipnet.Mask) before calling IPAdd
		ip, ipnet, _ := net.ParseCIDR("1.2.3.4/24")
		maskedIP := ip.Mask(ipnet.Mask) // This gives 1.2.3.0

		result := IPAdd(maskedIP, 20)
		if result.String() != "1.2.3.20" {
			t.Errorf("IPAdd(masked 1.2.3.4/24, 20) = %s; want 1.2.3.20", result.String())
		}
	})

	// Test 2: The operator handles negative CIDR offsets by adding network size
	t.Run("Operator handles negative CIDR offset", func(t *testing.T) {
		// For negative offsets in CIDR, operator does: start += netsize
		// So -20 becomes -20 + 256 = 236
		ip, ipnet, _ := net.ParseCIDR("1.2.3.4/24")
		maskedIP := ip.Mask(ipnet.Mask) // This gives 1.2.3.0

		// Operator converts -20 to 236
		result := IPAdd(maskedIP, 236)
		if result.String() != "1.2.3.236" {
			t.Errorf("IPAdd(masked 1.2.3.4/24, 236) = %s; want 1.2.3.236", result.String())
		}
	})

	// Test 3: Plain IPs use negative offsets directly
	t.Run("Plain IP negative offset", func(t *testing.T) {
		ip := net.ParseIP("1.2.3.4")
		result := IPAdd(ip, -20)
		if result.String() != "1.2.2.240" {
			t.Errorf("IPAdd(1.2.3.4, -20) = %s; want 1.2.2.240", result.String())
		}
	})

	// Test 4: Verify all the actual test cases from operator_test.go
	testCases := []struct {
		name      string
		ip        string
		offset    int
		expected  string
		needsMask bool
	}{
		// From operator tests - CIDR cases (operator masks these)
		{"CIDR offset 20", "1.2.3.4/24", 20, "1.2.3.20", true},
		{"CIDR negative -20", "1.2.3.4/24", 236, "1.2.3.236", true}, // -20+256

		// From operator tests - Plain IP cases
		{"IP offset 20", "1.2.3.4", 20, "1.2.3.24", false},
		{"IP negative -20", "1.2.3.4", -20, "1.2.2.240", false},

		// Multiple IP generation cases
		{"IP range start", "1.2.3.4", 20, "1.2.3.24", false},
		{"IP range next", "1.2.3.4", 21, "1.2.3.25", false},

		// Boundary cases from tests
		{"Small subnet", "192.168.1.16/29", 1, "192.168.1.17", true},
		{"Small subnet near end", "192.168.1.16/29", 6, "192.168.1.22", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var testIP net.IP

			if tc.needsMask {
				ip, ipnet, err := net.ParseCIDR(tc.ip)
				if err != nil {
					t.Fatalf("Failed to parse CIDR: %v", err)
				}
				testIP = ip.Mask(ipnet.Mask)
			} else {
				testIP = net.ParseIP(tc.ip)
				if testIP == nil {
					t.Fatalf("Failed to parse IP: %s", tc.ip)
				}
			}

			result := IPAdd(testIP, tc.offset)
			if result.String() != tc.expected {
				t.Errorf("IPAdd(%s, %d) = %s; want %s", tc.ip, tc.offset, result.String(), tc.expected)
			}
		})
	}
}

// TestRealWorldScenarios tests actual usage patterns from examples
func TestRealWorldScenarios(t *testing.T) {
	// These test the IPAdd function as it would be called by the operator

	t.Run("Gateway and DNS IPs", func(t *testing.T) {
		// Network: 10.0.0.0/24
		ip, ipnet, _ := net.ParseCIDR("10.0.0.0/24")
		baseIP := ip.Mask(ipnet.Mask)

		gateway := IPAdd(baseIP, 1)
		dns1 := IPAdd(baseIP, 2)
		dns2 := IPAdd(baseIP, 3)

		if gateway.String() != "10.0.0.1" {
			t.Errorf("Gateway = %s; want 10.0.0.1", gateway.String())
		}
		if dns1.String() != "10.0.0.2" {
			t.Errorf("DNS1 = %s; want 10.0.0.2", dns1.String())
		}
		if dns2.String() != "10.0.0.3" {
			t.Errorf("DNS2 = %s; want 10.0.0.3", dns2.String())
		}
	})

	t.Run("DHCP Pool", func(t *testing.T) {
		// Generate DHCP pool 192.168.10.100-200
		baseIP := net.ParseIP("192.168.10.0")

		start := IPAdd(baseIP, 100)
		end := IPAdd(baseIP, 200)

		if start.String() != "192.168.10.100" {
			t.Errorf("DHCP start = %s; want 192.168.10.100", start.String())
		}
		if end.String() != "192.168.10.200" {
			t.Errorf("DHCP end = %s; want 192.168.10.200", end.String())
		}
	})

	t.Run("Cross-octet scenarios", func(t *testing.T) {
		// Test crossing octet boundaries
		ip1 := net.ParseIP("10.0.0.250")
		result1 := IPAdd(ip1, 10)
		if result1.String() != "10.0.1.4" {
			t.Errorf("Cross octet = %s; want 10.0.1.4", result1.String())
		}

		ip2 := net.ParseIP("10.0.255.250")
		result2 := IPAdd(ip2, 10)
		if result2.String() != "10.1.0.4" {
			t.Errorf("Cross multiple octets = %s; want 10.1.0.4", result2.String())
		}
	})
}
