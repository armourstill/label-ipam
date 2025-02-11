package ipam

import (
	"testing"
)

func TestSize(t *testing.T) {
	m := LabelMap{
		"ref_uuid1": "8b6509aa-b4eb-441e-a8a6-06badadf035a",
		"ref_uuid2": "8b6509aa-b4eb-441e-a8a6-06badadf035a",
		"ref_uuid3": "8b6509aa-b4eb-441e-a8a6-06badadf035a",
		"ref_uuid4": "8b6509aa-b4eb-441e-a8a6-06badadf035a",
	}
	ipam := New("test", m)
	ipam.AddZone("FE08::-FE20::", true)
	for i := 0; i < 8; i++ {
		ipam.AllocAddrNext(m)
	}
	fatBlock, _ := ipam.Dump(true)
	if len(fatBlock) > BucketSize {
		t.Logf("Block size is %d", len(fatBlock))
	}
	thinBlock, _ := ipam.Dump(false)
	if len(fatBlock) == len(thinBlock) {
		t.Fatal("Size of fatBlock and thinBlock should not be equal")
	}
}
