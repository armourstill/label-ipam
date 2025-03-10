package ipam

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
)

var one = big.NewInt(1)

type ipam struct {
	// mutex  sync.RWMutex
	prefix string
	zones  map[string]*zone
	labels LabelMap
}

func New(prefix string, labels LabelMap) IPAM {
	ipam := &ipam{prefix: prefix, zones: make(map[string]*zone)}
	if labels != nil {
		ipam.labels = labels.Copy()
	} else {
		ipam.labels = make(LabelMap)
	}
	return ipam
}

func (i *ipam) SetLabel(key, value string) {
	i.labels[key] = value
}

func (i *ipam) RemoveLabel(key string) (string, bool) {
	value, ok := i.labels[key]
	delete(i.labels, key)
	return value, ok
}

func (i *ipam) Labels() LabelMap {
	if i.labels == nil {
		return nil
	}
	return i.labels.Copy()
}

func (i *ipam) overlappedWith(zone *zone) bool {
	for _, z := range i.zones {
		if z.start.Cmp(zone.end) > 0 || z.end.Cmp(zone.start) < 0 {
			continue
		}
		return true
	}
	return false
}

func (i *ipam) createZoneSingle(ip net.IP, lazy bool) *zone {
	ipBigInt := IPToBigInt(ip)
	zone := &zone{
		start: ipBigInt,
		end:   ipBigInt,
		lazy:  lazy,
		storage: &Zone{
			Literal: ip.String(),
			Buckets: make(map[string]*Bucket),
			Labels:  make(map[string]string),
		},
	}
	if IsIPv4(ip) {
		zone.version = 4
	} else {
		zone.version = 6
	}
	return zone
}

func (i *ipam) createZoneCIDR(cidr *net.IPNet, lazy bool) *zone {
	// 一些准备工作
	zone := &zone{
		lazy: lazy,
		storage: &Zone{
			Literal: cidr.String(),
			Buckets: make(map[string]*Bucket),
			Labels:  make(map[string]string),
		},
	}
	ones, _ := cidr.Mask.Size()
	// 计算基址和偏移
	local := IPToBigInt(cidr.IP)
	var offset *big.Int
	if IsIPv4(cidr.IP) {
		zone.version = 4
		offset = new(big.Int).Sub(new(big.Int).Lsh(one, uint(32-ones)), one)
	} else {
		zone.version = 6
		offset = new(big.Int).Sub(new(big.Int).Lsh(one, uint(128-ones)), one)
	}
	zone.start = new(big.Int).Add(local, one)                         // start避开0地址
	zone.end = new(big.Int).Sub(new(big.Int).Add(local, offset), one) // end避开广播地址
	return zone
}

func (i *ipam) createZoneRange(low, high net.IP, lazy bool) *zone {
	start := IPToBigInt(low)
	end := IPToBigInt(high)
	zone := &zone{
		start: start,
		end:   end,
		lazy:  lazy,
		storage: &Zone{
			Literal: low.String() + "-" + high.String(),
			Buckets: make(map[string]*Bucket),
			Labels:  make(map[string]string),
		},
	}
	if IsIPv4(low) {
		zone.version = 4
	} else {
		zone.version = 6
	}
	return zone
}

func (i *ipam) AddZone(literal string, lazy bool) error {
	if _, ok := i.zones[literal]; ok {
		return fmt.Errorf("Zone literal %s already exitst", literal)
	}

	var zone *zone
	// 根据literal的格式不同有不同的zone生成方式
	if single := net.ParseIP(literal); single != nil {
		zone = i.createZoneSingle(single, lazy)
	} else if ip, ipnet, err := net.ParseCIDR(literal); err == nil {
		if !ip.Equal(ipnet.IP) {
			return errors.New("Invalid CIDR network value")
		}
		zone = i.createZoneCIDR(ipnet, lazy)
	} else if pair := strings.Split(literal, "-"); len(pair) == 2 {
		low := net.ParseIP(pair[0])
		high := net.ParseIP(pair[1])
		if low == nil || high == nil {
			return errors.New("Invalid IP range value")
		}
		if (IsIPv4(low) && !IsIPv4(high)) || (IsIPv6(low) && !IsIPv6(high)) {
			return errors.New("Invalid IP range value: IPs format are different")
		}
		if IPToBigInt(low).Cmp(IPToBigInt(high)) >= 0 {
			return errors.New("The left IP should be less than the right one")
		}
		zone = i.createZoneRange(low, high, lazy)
	} else {
		return errors.New("Invalid format")
	}
	if i.overlappedWith(zone) {
		return errors.New("Literal overlapped")
	}
	i.zones[zone.storage.Literal] = zone

	return nil
}

func (i *ipam) SetZoneLabel(literal, key, value string) error {
	zone, ok := i.zones[strings.ToLower(literal)]
	if !ok {
		return fmt.Errorf("IP literal %s not exists", literal)
	}
	zone.storage.Labels[key] = value
	return nil
}

func (i *ipam) RemoveZone(literal string) error {
	if single := net.ParseIP(literal); single != nil {
		goto del
	} else if ip, ipnet, err := net.ParseCIDR(literal); err == nil {
		if !ip.Equal(ipnet.IP) {
			return errors.New("Invalid CIDR network value")
		}
		goto del
	} else if pair := strings.Split(literal, "-"); len(pair) == 2 {
		low := net.ParseIP(pair[0])
		high := net.ParseIP(pair[1])
		if low == nil || high == nil {
			return errors.New("Invalid IP range literal")
		}
		if (IsIPv4(low) && !IsIPv4(high)) || (IsIPv6(low) && !IsIPv6(high)) {
			return errors.New("Invalid IP range literal: IPs format are different")
		}
		if IPToBigInt(low).Cmp(IPToBigInt(high)) >= 0 {
			return errors.New("The left IP should be less than the right one")
		}
		goto del
	}
	return errors.New("Invalid literal format")

del:
	delete(i.zones, literal)

	return nil
}

func (i *ipam) RemoveZoneLabel(literal, key string) (string, bool) {
	zone, zoneOk := i.zones[strings.ToLower(literal)]
	if !zoneOk {
		return "", zoneOk
	}
	value, keyOk := zone.storage.Labels[key]
	delete(zone.storage.Labels, key)
	return value, keyOk
}

func (i *ipam) ZoneLabels(literal string) (LabelMap, bool) {
	zone, zoneOk := i.zones[strings.ToLower(literal)]
	if !zoneOk {
		return nil, zoneOk
	}
	if zone.storage.Labels == nil {
		return nil, zoneOk
	}
	return LabelMap(zone.storage.Labels).Copy(), zoneOk
}

func (i *ipam) IdleCount() string {
	usedCount := big.NewInt(int64(len(i.usedAddrs())))
	reservedCount := big.NewInt(int64(len(i.reservedAddrs())))
	totalCount := big.NewInt(0)
	for _, zone := range i.zones {
		zoneTotal := big.NewInt(0).Sub(zone.end, zone.start)
		zoneTotal.Add(zoneTotal, big.NewInt(1))
		totalCount.Add(totalCount, zoneTotal)
	}
	return totalCount.Sub(totalCount, usedCount).Sub(totalCount, reservedCount).String()
}

// FIXME: For IPv6 zone, there is a risk of reaching slice capacity
func (i *ipam) usedAddrs() []string {
	result := make([]string, 0)
	for _, zone := range i.zones {
		for _, b := range zone.storage.Buckets {
			for addr := range b.Used {
				result = append(result, addr)
			}
		}
	}
	return result
}

func (i *ipam) UsedAddrs() []string {
	return i.usedAddrs()
}

// FIXME: For IPv6 zone, there is a risk of reaching slice capacity
func (i *ipam) reservedAddrs() []string {
	result := make([]string, 0)
	for _, zone := range i.zones {
		for addr := range zone.storage.Reserved {
			result = append(result, addr)
		}
	}
	return result
}

func (i *ipam) ReservedAddrs() []string {
	return i.reservedAddrs()
}

func (i *ipam) AllocAddrSpecific(specific string, labels LabelMap) error {
	ip := net.ParseIP(specific)
	if ip == nil {
		return fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if (IsIPv4(ip) && zone.version == 6) || (!IsIPv4(ip) && zone.version == 4) {
			continue
		}
		if !zone.Contains(ip) {
			continue
		}
		if zone.IPReserved(ip) {
			return fmt.Errorf("IP %s already reserved", specific)
		}
		zone.AlocAddrWithCreateBucket(i.prefix, ip, labels)
		return nil
	}
	return fmt.Errorf("IP %s is not handled", specific)
}

func (i *ipam) AllocAddrNext(labels LabelMap) (net.IP, error) {
	for _, zone := range i.zones {
		for tmp := new(big.Int).Add(zone.start, big.NewInt(0)); tmp.Cmp(zone.end) <= 0; tmp.Add(tmp, one) {
			ip := BigIntToIP(tmp, zone.version)
			if zone.IPUsed(ip) || zone.IPReserved(ip) {
				continue
			}
			zone.AlocAddrWithCreateBucket(i.prefix, ip, labels)
			return ip, nil
		}
	}
	return nil, errors.New("No remained IP to allocate")
}

func (i *ipam) ReserveAddr(specific string, labels LabelMap) error {
	ip := net.ParseIP(specific)
	if ip == nil {
		return fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if (IsIPv4(ip) && zone.version == 6) || (!IsIPv4(ip) && zone.version == 4) {
			continue
		}
		if !zone.Contains(ip) {
			continue
		}
		if zone.IPUsed(ip) {
			return fmt.Errorf("IP %s is in use", specific)
		}
		if zone.IPReserved(ip) {
			return fmt.Errorf("IP %s already reserved", specific)
		}
		if zone.storage.Reserved == nil {
			zone.storage.Reserved = make(map[string]*Descriptor)
		}
		zone.storage.Reserved[ip.String()] = &Descriptor{Labels: labels.Copy()}
		return nil
	}
	return fmt.Errorf("IP %s is not handled", specific)
}

func (i *ipam) ReleaseAddr(specific string) error {
	ip := net.ParseIP(specific)
	if ip == nil {
		return fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if !zone.Contains(ip) {
			continue
		}
		// 无差别尝试移除
		zone.ReleaseAddrWithDeleteBucket(ip)
		return nil
	}
	return fmt.Errorf("IP %s is not handled", specific)
}

func (i *ipam) SetAddrLabel(specific, key, value string) error {
	ip := net.ParseIP(specific)
	if ip == nil {
		return fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if zone.SetAddrLabel(ip, key, value) {
			return nil
		}
	}
	return fmt.Errorf("IP %s not allocated", specific)
}

func (i *ipam) RemoveAddrLabel(specific, key string) error {
	ip := net.ParseIP(specific)
	if ip == nil {
		return fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if zone.RemoveAddrLabel(ip, key) {
			return nil
		}
	}
	return fmt.Errorf("IP %s not allocated", specific)
}

func (i *ipam) AddrLabels(specific string) (LabelMap, error) {
	ip := net.ParseIP(specific)
	if ip == nil {
		return nil, fmt.Errorf("Invalid IP format %s", specific)
	}
	for _, zone := range i.zones {
		if desc, ok := zone.GetAddrDesc(ip); ok {
			return LabelMap(desc.Labels).Copy(), nil
		}
	}
	return nil, fmt.Errorf("IP %s not allocated", specific)
}

func (i *ipam) FindLiteral(specific string) string {
	ip := net.ParseIP(specific)
	if ip == nil {
		return ""
	}
	for _, zone := range i.zones {
		if zone.Contains(ip) {
			return zone.storage.Literal
		}
	}
	return ""
}

func (i *ipam) Literals() []string {
	results := make([]string, 0)
	for _, zone := range i.zones {
		results = append(results, zone.storage.Literal)
	}
	return results
}

func (i *ipam) Dump(fat bool) ([]byte, error) {
	var resize func(*zone) *Zone
	if fat {
		resize = func(zone *zone) *Zone {
			storage := zone.storage
			return &Zone{
				Literal:  storage.Literal,
				Labels:   storage.Labels,
				Buckets:  storage.Buckets,
				Reserved: storage.Reserved,
			}
		}
	} else {
		resize = func(zone *zone) *Zone {
			storage := zone.storage
			emptyBuckets := make(map[string]*Bucket)
			// 只保留key，用于以后索引
			for key := range storage.Buckets {
				emptyBuckets[key] = nil
			}
			return &Zone{
				Literal:  storage.Literal,
				Labels:   storage.Labels,
				Buckets:  emptyBuckets,
				Reserved: storage.Reserved,
			}
		}
	}
	// 生成一个Block，把所有zone对应的Zone放入Block，最后做Marshal
	block := &Block{
		Labels: i.labels.Copy(),
		Zones:  make([]*Zone, 0),
	}
	for _, zone := range i.zones {
		block.Zones = append(block.Zones, resize(zone))
	}
	return block.Marshal()
}

func (i *ipam) DumpZoneAddrs(literal string, onlyKeys bool) (map[string][]byte, error) {
	zone, zoneOk := i.zones[strings.ToLower(literal)]
	if !zoneOk {
		return nil, fmt.Errorf("IP Lliteral %s not exists", literal)
	}
	result := make(map[string][]byte)
	if onlyKeys {
		for key := range zone.storage.Buckets {
			result[key] = nil
		}
	} else {
		for key, b := range zone.storage.Buckets {
			raw, err := b.Marshal()
			if err != nil {
				return nil, fmt.Errorf("Marshal IP failed: %s", err)
			}
			result[key] = raw
		}
	}
	return result, nil
}

func (i *ipam) loadZoneSingle(z *Zone, lazy bool) *ipam {
	ip := net.ParseIP(z.Literal)
	ipBigInt := IPToBigInt(ip)
	zone := &zone{start: ipBigInt, end: ipBigInt, lazy: lazy, storage: z, version: 6}
	if IsIPv4(ip) {
		zone.version = 4
	}
	i.zones[z.Literal] = zone
	return i
}

func (i *ipam) loadZoneCIDR(z *Zone, lazy bool) *ipam {
	ip, cidr, _ := net.ParseCIDR(z.Literal)
	ones, _ := cidr.Mask.Size()
	local := IPToBigInt(cidr.IP)
	lsh := uint(128 - ones)
	version := uint8(6)
	if IsIPv4(ip) {
		lsh = uint(32 - ones)
		version = 4
	}
	offset := new(big.Int).Sub(new(big.Int).Lsh(one, lsh), one)
	start := new(big.Int).Add(local, one)
	end := new(big.Int).Sub(new(big.Int).Add(local, offset), one)
	zone := &zone{start: start, end: end, lazy: lazy, storage: z, version: version}
	i.zones[z.Literal] = zone
	return i
}

func (i *ipam) loadZoneRange(z *Zone, lazy bool) *ipam {
	pair := strings.Split(z.Literal, "-")
	ip0, ip1 := net.ParseIP(pair[0]), net.ParseIP(pair[1])
	start := IPToBigInt(ip0)
	end := IPToBigInt(ip1)
	zone := &zone{start: start, end: end, lazy: lazy, storage: z}
	if IsIPv4(ip0) {
		zone.version = 4
	}
	i.zones[z.Literal] = zone
	return i
}

func (i *ipam) loadZone(z *Zone, lazy bool) error {
	if single := net.ParseIP(z.Literal); single != nil {
		i.loadZoneSingle(z, lazy)
	} else if ip, ipnet, err := net.ParseCIDR(z.Literal); err == nil {
		if !ip.Equal(ipnet.IP) {
			return errors.New("Invalid CIDR network value")
		}
		i.loadZoneCIDR(z, lazy)
	} else if pair := strings.Split(z.Literal, "-"); len(pair) == 2 {
		low := net.ParseIP(pair[0])
		high := net.ParseIP(pair[1])
		if low == nil || high == nil {
			return errors.New("Invalid IP range value")
		}
		if (IsIPv4(low) && !IsIPv4(high)) || (IsIPv6(low) && !IsIPv6(high)) {
			return errors.New("Invalid IP range value: IPs format are different")
		}
		if IPToBigInt(low).Cmp(IPToBigInt(high)) >= 0 {
			return errors.New("The left IP should be less than the right one")
		}
		i.loadZoneRange(z, lazy)
	} else {
		return errors.New("Invalid format")
	}

	return nil
}

func (i *ipam) Load(raw []byte) error {
	block := &Block{}
	if err := block.Unmarshal(raw); err != nil {
		return err
	}
	for _, z := range block.Zones {
		if z.Buckets == nil {
			z.Buckets = make(map[string]*Bucket)
		}
		if z.Reserved == nil {
			z.Reserved = make(map[string]*Descriptor)
		}
		if err := i.loadZone(z, false); err != nil {
			return err
		}
	}
	return nil
}

func (i *ipam) LoadZoneAddrs(literal string, addrs map[string][]byte, force bool) error {
	zone, zoneOk := i.zones[strings.ToLower(literal)]
	if !zoneOk {
		return fmt.Errorf("IP Lliteral %s not exists", literal)
	}
	temp := make(map[string]*Bucket)
	for key, raw := range addrs {
		if !strings.HasPrefix(key, i.prefix) {
			continue
		}
		if _, ok := zone.storage.Buckets[key]; !ok && !force {
			// 如果key不属于该zone，则跳过
			continue
		}
		bucket := &Bucket{}
		if err := bucket.Unmarshal(raw); err != nil {
			return fmt.Errorf("Unmarshal IP failed: %s", err)
		}
		if bucket.Used == nil {
			bucket.Used = make(map[string]*Descriptor)
		}
		temp[key] = bucket
	}
	for key, b := range temp {
		zone.storage.Buckets[key] = b
	}
	return nil
}
