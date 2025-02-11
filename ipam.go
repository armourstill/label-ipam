package ipam

//go:generate sh codegen.sh

import (
	"net"
)

// IPAM manage IPv4 and IPv6 addresses and serialize them
type IPAM interface {
	// Set label of IPAM itself
	SetLabel(key, value string)
	// Remove label of IPAM, return the value and the key exists or not
	RemoveLabel(key string) (string, bool)
	// List all labels
	Labels() LabelMap
	// Add address segment, or called zone
	//
	// Field literal supports IP format as follows:
	//
	// 1. Single IPv4/IPv6, such as 192.168.0.1 or FE80::12
	//
	// 2. Interval with a dash, such as 192.168.0.1-192.168.3.2 or FE80::12-FE80::1:12
	//
	// 3. CIDR network address, such as 192.168.0..0/24 or FE80::/64
	//
	// Warning: the zero address and broadcast address will be unavailable when using a CIDR network address.
	// If they must be used, please change format to interval.
	//
	// Field lazy is invalid for now
	AddZone(literal string, lazy bool) error
	// Set label of zone
	SetZoneLabel(literal, key, value string) error
	// Remove a zone
	RemoveZone(literal string) error
	// Remove label of zone, return the value and the key exists or not
	RemoveZoneLabel(literal, key string) (string, bool)
	// List all labels of a zone
	ZoneLabels(literal string) (LabelMap, bool)
	// Return available address count as a string, the value is 'all - used - reserved'
	IdleCount() string
	// Return all used addresses
	UsedAddrs() []string
	// Return all reserved addresses
	ReservedAddrs() []string
	// Allocate a specified addr and add/update it's labels, an used addr can be allocated again
	AllocAddrSpecific(specific string, labels LabelMap) error
	// Allocate a new random addr and add it's labels
	AllocAddrNext(labels LabelMap) (net.IP, error)
	// Reserve an unused addr and add it's labels
	ReserveAddr(specific string, labels LabelMap) error
	// Release an used or reserved addr, some used addrs could be released more than one time
	ReleaseAddr(specific string) error
	// Set label of an used or reserved addr
	SetAddrLabel(specific, key, value string) error
	// Remove label of an used or reserved addr
	RemoveAddrLabel(specific, key string) error
	// List all labels of an used or reserved addr
	AddrLabels(specific string) (LabelMap, error)
	// Find the zone literal from an addr
	FindLiteral(specific string) string
	// List all zone literals
	Literals() []string
	// Export all zones with allocated addrs as bytes
	//
	// If fat is true, then descriptor info contained
	Dump(fat bool) ([]byte, error)
	// Export specified zone with allocated addrs as bytes
	//
	// If onlyKeys is true, then omits any descriptor info
	DumpZoneAddrs(literal string, onlyKeys bool) (map[string][]byte, error)
	// Load all zones from bytes, cover the zone with same literal
	Load(raw []byte) error
	// Load specified zone from bytes. If a key in addrs is not contained in zone, then it will be ignored.
	//
	// If force is true, then load all keys in addrs
	LoadZoneAddrs(literal string, addrs map[string][]byte, force bool) error
}
