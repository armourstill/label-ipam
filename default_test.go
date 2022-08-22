package ipam

import (
	"context"
	"strings"
	"testing"
)

func TestBasic(t *testing.T) {
	ctx := context.TODO()

	// 创建ipam实例
	ipam := New("test", nil)

	// 添加zone，相当于添加IP段
	if err := ipam.AddZone(ctx, "192.168.1.0/24", true); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AddZone(ctx, "FE80::12", true); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AddZone(ctx, "FE80::30-FE80::1:30", true); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AddZone(ctx, "192.168.3.0-192.168.3.10", true); err != nil {
		t.Fatal(err)
	}

	// 申请/保留IP地址
	if err := ipam.AllocAddrSpecific(ctx, "192.168.1.1", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AllocAddrSpecific(ctx, "FE80::12", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AllocAddrSpecific(ctx, "FE80::1:12", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AllocAddrSpecific(ctx, "192.168.3.3", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AllocAddrSpecific(ctx, "192.168.1.0", nil); err == nil {
		t.Fatal("Target IP should not be acquired")
	}
	result, err := ipam.AllocAddrNext(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ipam.ReserveAddr(ctx, "192.168.1.100", nil); err != nil {
		t.Fatal(err)
	}
	if err := ipam.AllocAddrSpecific(ctx, "192.168.1.100", nil); err == nil {
		t.Fatal("Target IP should be resversed")
	}

	// 释放IP地址
	if err := ipam.ReleaseAddr(ctx, result.String()); err != nil {
		t.Fatal(err)
	}
	// 从IP地址查询所属zone，即IP段
	if literal := ipam.FindLiteral(ctx, "192.168.3.0"); len(literal) <= 0 {
		t.Fatal("Target IP literal should be found")
	}
	// 列出所有zone，即所有IP段
	if len(ipam.Literals(ctx)) != 4 {
		t.Fatal("IPAM should have 4 zones")
	}

	// 移除zone
	if err := ipam.RemoveZone(ctx, "192.168.1.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := ipam.ReleaseAddr(ctx, "192.168.1.1"); err == nil {
		t.Fatal("Target IP should not be handled")
	}
}

func TestLabel(t *testing.T) {
	ctx := context.TODO()
	foo, bar, bar2 := "foo", "bar", "bar2"
	m := map[string]string{foo: bar}

	// 创建ipam实例
	ipam := New("test", m)
	if ipam.Labels(ctx)[foo] != bar {
		t.Fatal("Wrong IPAM label value")
	}
	ipam.SetLabel(ctx, foo, bar2)
	if value, _ := ipam.RemoveLabel(ctx, foo); value != bar2 {
		t.Fatal("Wrong IPAM label value from removing a key")
	}

	literal := "192.168.1.0/24"
	// 添加zone，相当于添加IP段
	if err := ipam.AddZone(ctx, literal, true); err != nil {
		t.Fatal(err)
	}
	if err := ipam.SetZoneLabel(ctx, literal, foo, bar); err != nil {
		t.Fatal(err)
	}
	if value, _ := ipam.RemoveZoneLabel(ctx, literal, foo); value != bar {
		t.Fatal("Wrong zone label value from removing a key")
	}

	specific := "192.168.1.1"
	// 申请IP地址，并携带标签
	if err := ipam.AllocAddrSpecific(ctx, specific, m); err != nil {
		t.Fatal(err)
	}
	// 设置地址标签并覆盖key
	ipam.SetAddrLabel(ctx, specific, foo, bar2)
	labels, _ := ipam.AddrLabels(ctx, specific)
	if len(labels) <= 0 {
		t.Fatalf("IP %s has no label", specific)
	}
	if value := labels[foo]; value != bar2 {
		t.Fatal("Wrong IP label value")
	}
	if err := ipam.RemoveAddrLabel(ctx, specific, foo); err != nil {
		t.Fatal(err)
	}
}

func TestDumpLoad(t *testing.T) {
	ctx := context.TODO()
	literal := "FE08::-FE20::"
	AddrNumPerBucket = 64
	defer func() {
		AddrNumPerBucket = 4096
	}()

	// Dump
	ipam1 := New("test", nil)
	ipam1.AddZone(ctx, literal, true)
	for i := 0; i < AddrNumPerBucket*2; i++ {
		ipam1.AllocAddrNext(ctx, nil)
	}
	rawBlock, err := ipam1.Dump(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	dumpedAddrs, err := ipam1.DumpZoneAddrs(ctx, literal)
	if err != nil {
		t.Fatal(err)
	}
	if len(dumpedAddrs) != 2 {
		t.Fatal("2 buckets should be dumped")
	}

	// Load
	ipam2 := New("test", nil)
	if err := ipam2.Load(ctx, rawBlock); err != nil {
		t.Fatal(err)
	}
	if err := ipam2.LoadZoneAddrs(ctx, literal, dumpedAddrs); err != nil {
		t.Fatal(err)
	}
	for _, b := range ipam2.(*ipam).zones[strings.ToLower(literal)].storage.Buckets {
		if len(b.Used) != AddrNumPerBucket {
			t.Fatal("Loaded addrs have fault")
		}
	}
}

func BenchmarkAllocNext(b *testing.B) {
	ctx := context.TODO()
	ipam := New("test", nil)
	ipam.AddZone(ctx, "0.0.0.0-255.0.0.0", true)
	for n := 0; n < b.N; n++ {
		ipam.AllocAddrNext(ctx, nil)
	}
}