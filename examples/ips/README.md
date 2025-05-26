# IPs Operator Examples

The `ips` operator performs IP address arithmetic, allowing you to calculate IP addresses based on an offset from a base IP or CIDR block. It's particularly useful for network planning and IP allocation.

## Files in this directory:

1. **basic.yml** - Basic IP arithmetic operations
2. **network-planning.yml** - Network and subnet planning
3. **service-allocation.yml** - Allocating IPs to services
4. **multi-subnet.yml** - Working with multiple subnets

## Usage Pattern:

```yaml
# Single IP
single_ip: (( ips "10.0.0.10" 5 ))  # Returns "10.0.0.15"

# Multiple IPs
ip_range: (( ips "10.0.0.10" 0 5 )) # Returns ["10.0.0.10", "10.0.0.11", ..., "10.0.0.14"]

# CIDR notation (starts from network address)
from_cidr: (( ips "10.0.0.0/24" 5 ))  # Returns "10.0.0.5"

# Negative offset (from end of network)
from_end: (( ips "10.0.0.0/24" -1 ))  # Returns "10.0.0.254"
```

## Key Features:

- Add offset to IP addresses
- Start from network address with CIDR
- Generate ranges with count parameter
- Negative offsets count from end
- Works with both IPv4 addresses

## Common Use Cases:

- Gateway and DNS server allocation
- DHCP range configuration
- Static IP assignments
- Service endpoint planning
- Network documentation

## Running Examples:

```bash
# Basic IP calculations
spruce merge basic.yml

# Network planning
spruce merge network-planning.yml

# Service allocations
spruce merge service-allocation.yml
```