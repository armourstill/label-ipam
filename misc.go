package ipam

import (
	"math/big"
	"net"
)

// IsIPv4 return true if addr is an IPv4 address
func IsIPv4(addr net.IP) bool {
	if addr == nil {
		return false
	}
	return addr.To4() != nil
}

// IsIPv6 return true if addr is an IPv6 address
func IsIPv6(addr net.IP) bool {
	if addr == nil {
		return false
	}
	return addr.To4() == nil
}

func IPToBigInt(ip net.IP) *big.Int {
	if IsIPv4(ip) {
		return big.NewInt(0).SetBytes(ip.To4())
	}
	return big.NewInt(0).SetBytes(ip.To16())
}

func BigIntToIP(ipInt *big.Int, version uint8) net.IP {
	if version == 4 {
		return net.IP(ipInt.FillBytes(make([]byte, 4)))
	}
	return net.IP(ipInt.FillBytes(make([]byte, 16)))
}
