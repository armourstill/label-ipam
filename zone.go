package ipam

import (
	"math/big"
	"net"
	"strconv"
)

const BucketSize = (512 + 1024) * 1024

var AddrNumPerBucket = 4096

type zone struct {
	start   *big.Int
	end     *big.Int
	version uint8
	lazy    bool
	storage *Zone
}

func (z *zone) Contains(ip net.IP) bool {
	ipBigInt := IPToBigInt(ip)
	return z.start.Cmp(ipBigInt) <= 0 && z.end.Cmp(ipBigInt) >= 0
}

func (z *zone) IPUsed(ip net.IP) bool {
	for _, bucket := range z.storage.Buckets {
		if _, ok := bucket.Used[ip.String()]; ok {
			return ok
		}
	}
	return false
}

func (z *zone) IPReserved(ip net.IP) bool {
	if z.storage.Reserved == nil {
		return false
	}
	_, ok := z.storage.Reserved[ip.String()]
	return ok
}

func (z *zone) GetAddrDesc(ip net.IP) (*Descriptor, bool) {
	for _, bucket := range z.storage.Buckets {
		if desc, ok := bucket.Used[ip.String()]; ok {
			return desc, ok
		}
	}
	return nil, false
}

func (z *zone) SetAddrLabel(ip net.IP, key, value string) bool {
	for _, bucket := range z.storage.Buckets {
		if desc, ok := bucket.Used[ip.String()]; ok {
			if desc == nil {
				bucket.Used[ip.String()] = &Descriptor{Labels: map[string]string{key: value}}
			} else if desc.Labels == nil {
				desc.Labels = map[string]string{key: value}
			} else {
				desc.Labels[key] = value
			}
			return true
		}
	}
	return false
}

func (z *zone) RemoveAddrLabel(ip net.IP, key string) bool {
	for _, bucket := range z.storage.Buckets {
		if desc, ok := bucket.Used[ip.String()]; ok {
			delete(desc.Labels, key)
			return true
		}
	}
	return false
}

func (z *zone) ReserveAddr(ip net.IP, desc *Descriptor) {
	if z.storage.Reserved == nil {
		z.storage.Reserved = make(map[string]*Descriptor)
	}
	// TODO
}

func (z *zone) AlocAddrWithCreateBucket(prefix string, ip net.IP, labels LabelMap) {
	var bucket *Bucket
	if z.storage.Buckets == nil {
		z.storage.Buckets = make(map[string]*Bucket)
	}
	for _, b := range z.storage.Buckets {
		// if found, increase RefCount and update Labels
		if desc, ok := b.Used[ip.String()]; ok {
			desc.RefCount++
			for k, v := range labels {
				desc.Labels[k] = v
			}
			return
		}
		if len(b.Used) < AddrNumPerBucket {
			bucket = b
			break
		}
	}
	if bucket == nil {
		bucket = &Bucket{Used: make(map[string]*Descriptor)}
		key := prefix + "/" + z.storage.Literal + "/" + strconv.Itoa(len(z.storage.Buckets))
		z.storage.Buckets[key] = bucket
	}
	desc := &Descriptor{RefCount: 1}
	if labels != nil {
		desc.Labels = labels.Copy()
	}
	bucket.Used[ip.String()] = desc
}

func (z *zone) ReleaseAddrWithDeleteBucket(ip net.IP) {
	// query from Reserved at first
	if _, reserved := z.storage.Reserved[ip.String()]; reserved {
		delete(z.storage.Reserved, ip.String())
		return
	}
	for key, bucket := range z.storage.Buckets {
		desc, ok := bucket.Used[ip.String()]
		if !ok {
			continue
		}
		desc.RefCount--
		if desc.RefCount > 0 {
			return
		}
		delete(bucket.Used, ip.String())
		if len(bucket.Used) <= 0 {
			delete(z.storage.Buckets, key)
		}
		return
	}
}
