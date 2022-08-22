package ipam

import (
	"math/big"
	"net"
	"testing"
)

func TestBigIntAndIP(t *testing.T) {
	start := IPToBigInt(net.ParseIP("192.168.3.0"))
	end := IPToBigInt(net.ParseIP("192.168.3.10"))
	result := big.NewInt(0)
	if result.Sub(end, start); result.Uint64() != 10 {
		t.Fatal("result should be 10")
	}
	mid := IPToBigInt(net.ParseIP("192.168.3.3"))
	if !(start.Cmp(mid) <= 0 && end.Cmp(mid) >= 0) {
		t.Fatal("mid IP should be between start and end")
	}
}
