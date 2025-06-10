package netutil

import (
	"math/big"
	"net"
)

// IPAdd adds an integer offset to an IP address and returns the resulting IP.
// It works with both IPv4 and IPv6 addresses.
func IPAdd(ip net.IP, offset int) net.IP {
	// Convert IP to big.Int
	ipInt := new(big.Int)
	ipInt.SetBytes(ip)

	// Add the offset
	offsetInt := big.NewInt(int64(offset))
	ipInt.Add(ipInt, offsetInt)

	// Convert back to IP
	// Determine if it's IPv4 or IPv6 based on the original IP
	if ip.To4() != nil {
		// IPv4
		// Ensure we're working with a 4-byte representation
		ipBytes := ipInt.Bytes()
		if len(ipBytes) < 4 {
			// Pad with zeros at the beginning if necessary
			padded := make([]byte, 4)
			copy(padded[4-len(ipBytes):], ipBytes)
			ipBytes = padded
		} else if len(ipBytes) > 4 {
			// Truncate to last 4 bytes if overflow
			ipBytes = ipBytes[len(ipBytes)-4:]
		}
		return net.IP(ipBytes).To4()
	}

	// IPv6
	ipBytes := ipInt.Bytes()
	if len(ipBytes) < 16 {
		// Pad with zeros at the beginning if necessary
		padded := make([]byte, 16)
		copy(padded[16-len(ipBytes):], ipBytes)
		ipBytes = padded
	} else if len(ipBytes) > 16 {
		// Truncate to last 16 bytes if overflow
		ipBytes = ipBytes[len(ipBytes)-16:]
	}
	return net.IP(ipBytes)
}
