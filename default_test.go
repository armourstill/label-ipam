package ipam

import (
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func TestBasic(t *testing.T) {
	ipm := New("test", nil)

	if err := ipm.AddZone("192.168.1.0/24", true); err != nil {
		t.Fatal(err)
	}
	if idleCount := ipm.IdleCount(); idleCount != "254" {
		t.Fatalf("Wrong idle count %s", idleCount)
	}
	if err := ipm.AddZone("192.168.1.0/28", true); err == nil {
		t.Fatal("Literal should be overlapped")
	}
	if err := ipm.AddZone("FE80::12", true); err != nil {
		t.Fatal(err)
	}
	if idleCount := ipm.IdleCount(); idleCount != "255" {
		t.Fatalf("Wrong idle count %s", idleCount)
	}
	if err := ipm.AddZone("FE80::30-FE80::1:30", true); err != nil {
		t.Fatal(err)
	}
	if err := ipm.AddZone("192.168.3.0-192.168.3.10", true); err != nil {
		t.Fatal(err)
	}

	ipv4A := "192.168.1.1"
	ipv4B := "192.168.3.3"
	ipv4C := "192.168.1.0"
	ipv6A := "FE80::12"
	ipv6B := "FE80::1:12"
	if err := ipm.AllocAddrSpecific(ipv4A, nil); err != nil {
		t.Fatal(err)
	}
	if err := ipm.AllocAddrSpecific(ipv4B, nil); err != nil {
		t.Fatal(err)
	}
	if err := ipm.AllocAddrSpecific(ipv4C, nil); err == nil {
		t.Fatal("Target IP should not be acquired")
	}
	if err := ipm.AllocAddrSpecific(ipv6A, nil); err != nil {
		t.Fatal(err)
	}
	if err := ipm.AllocAddrSpecific(ipv6B, nil); err != nil {
		t.Fatal(err)
	}
	result, err := ipm.AllocAddrNext(nil)
	if err != nil {
		t.Fatal(err)
	}

	// allocate twice
	if err := ipm.AllocAddrSpecific(ipv4A, nil); err != nil {
		t.Fatal(err)
	}
	var desc *Descriptor
	for _, bucket := range ipm.(*ipam).zones["192.168.1.0/24"].storage.Buckets {
		if d, ok := bucket.Used[ipv4A]; ok {
			desc = d
			break
		}
	}
	if desc == nil {
		t.Fatalf("%s should be allocated", ipv4A)
	}
	if desc.RefCount != 2 {
		t.Fatalf("%s should be allocated 2 times", ipv4A)
	}

	if err := ipm.ReserveAddr("192.168.1.100", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipm.AllocAddrSpecific("192.168.1.100", nil); err == nil {
		t.Fatal("Target IP should be resversed")
	}

	if err := ipm.ReleaseAddr(result.String()); err != nil {
		t.Fatal(err)
	}
	if literal := ipm.FindLiteral("192.168.3.0"); len(literal) <= 0 {
		t.Fatal("Target IP literal should be found")
	}
	if len(ipm.Literals()) != 4 {
		t.Fatal("IPAM should have 4 zones")
	}

	if err := ipm.RemoveZone("192.168.1.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := ipm.ReleaseAddr("192.168.1.1"); err == nil {
		t.Fatal("Target IP should not be handled")
	}

	if len(ipm.UsedAddrs()) != 3 {
		t.Fatal("There should be 3 IPs remained")
	}
}

func TestLabel(t *testing.T) {
	foo, bar, bar2 := "foo", "bar", "bar2"
	m := map[string]string{foo: bar}

	ipam := New("test", m)
	if ipam.Labels()[foo] != bar {
		t.Fatal("Wrong IPAM label value")
	}
	ipam.SetLabel(foo, bar2)
	if value, _ := ipam.RemoveLabel(foo); value != bar2 {
		t.Fatal("Wrong IPAM label value from removing a key")
	}

	literal := "192.168.1.0/24"
	if err := ipam.AddZone(literal, true); err != nil {
		t.Fatal(err)
	}
	if err := ipam.SetZoneLabel(literal, foo, bar); err != nil {
		t.Fatal(err)
	}
	if value, _ := ipam.RemoveZoneLabel(literal, foo); value != bar {
		t.Fatal("Wrong zone label value from removing a key")
	}

	specific := "192.168.1.1"
	if err := ipam.AllocAddrSpecific(specific, m); err != nil {
		t.Fatal(err)
	}
	ipam.SetAddrLabel(specific, foo, bar2)
	labels, _ := ipam.AddrLabels(specific)
	if len(labels) <= 0 {
		t.Fatalf("IP %s has no label", specific)
	}
	if value := labels[foo]; value != bar2 {
		t.Fatal("Wrong IP label value")
	}
	if err := ipam.RemoveAddrLabel(specific, foo); err != nil {
		t.Fatal(err)
	}
}

func TestDumpAndLoad(t *testing.T) {
	literal := "FE08::-FE09::"
	AddrNumPerBucket = 64
	defer func() {
		AddrNumPerBucket = 4096
	}()

	ipam1 := New("test", nil)
	ipam1.AddZone(literal, true)
	bigIdle1, _ := big.NewInt(0).SetString(ipam1.IdleCount(), 10)
	allocNum := AddrNumPerBucket * 2
	for i := 0; i < allocNum; i++ {
		ipam1.AllocAddrNext(nil)
	}
	rawBlock, err := ipam1.Dump(false)
	if err != nil {
		t.Fatal(err)
	}
	dumpedAddrs, err := ipam1.DumpZoneAddrs(literal, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(dumpedAddrs) != 2 {
		t.Fatal("2 buckets should be dumped")
	}

	ipam2 := New("test", nil)
	if err := ipam2.Load(rawBlock); err != nil {
		t.Fatal(err)
	}
	for _, literal := range ipam2.Literals() {
		onlyKeys, err := ipam2.DumpZoneAddrs(literal, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(onlyKeys) != 2 {
			t.Fatal("2 buckets should be dumped")
		}
	}
	if err := ipam2.LoadZoneAddrs(literal, dumpedAddrs, false); err != nil {
		t.Fatal(err)
	}
	bigIdle2, _ := big.NewInt(0).SetString(ipam2.IdleCount(), 10)
	if bigIdle1.Sub(bigIdle1, bigIdle2).String() != strconv.Itoa(allocNum) {
		t.Fatalf("Wrong idle count %s", bigIdle2.String())
	}
	for _, b := range ipam2.(*ipam).zones[strings.ToLower(literal)].storage.Buckets {
		if len(b.Used) != AddrNumPerBucket {
			t.Fatal("Loaded addrs have fault")
		}
	}
}

func BenchmarkAllocNext(b *testing.B) {
	ipam := New("test", nil)
	ipam.AddZone("0.0.0.0-255.0.0.0", true)
	for n := 0; n < b.N; n++ {
		ipam.AllocAddrNext(nil)
	}
}
