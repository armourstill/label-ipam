package ipam

//go:generate sh codegen.sh

import (
	"context"
	"net"
)

// 支持IPv4与IPv6的地址管理与持久化
type IPAM interface {
	// 设置IPAM自己的标签
	SetLabel(ctx context.Context, key, value string)
	// 移除IPAM的标签，返回被移除的key的value与key是否存在，若key不存在则默认value为空字符串
	RemoveLabel(ctx context.Context, key string) (string, bool)
	// 列出指定IPAM的所有标签
	Labels(ctx context.Context) LabelMap

	// 添加Zone，即IP地址段
	//
	// literal支持以下几种IP地址格式：
	//
	// 1. 单个IPv4/IPv6地址(如192.168.0.1，或FE80::12)
	//
	// 2. 连字符(闭)区间(如192.168.0.1-192.168.3.2，或FE80::12-FE80::1:12)
	//
	// 3. CIDR网络地址(如192.168.0..0/24，或FE80::/64)
	//
	// 注意：当使用CIDR网络地址时，该网络的0地址和广播地址不会被计入可用地址内，若需要使用这些地址，请使用连字符区间格式
	//
	// lazy: 目前无效
	AddZone(ctx context.Context, literal string, lazy bool) error
	// 设置Zone的标签
	SetZoneLabel(ctx context.Context, literal, key, value string) error
	// 移除由literal指定的Zone
	RemoveZone(ctx context.Context, literal string) error
	// 移除Zone的标签，返回被移除的key的value与zone/key是否存在，若zone/key不存在则默认value为空字符串
	RemoveZoneLabel(ctx context.Context, literal, key string) (string, bool)
	// 列出指定Zone的所有标签并返回Zone是否存在
	ZoneLabels(ctx context.Context, literal string) (LabelMap, bool)

	// 全部已使用的IP地址
	UsedAddrs(ctx context.Context) []string
	// 全部已保留的IP地址
	ReservedAddrs(ctx context.Context) []string
	// 分配一个指定的IP，允许指定已使用的IP（内部引用计数加1），若有标签，则一并填写（覆盖）标签
	AllocAddrSpecific(ctx context.Context, specific string, labels LabelMap) error
	// 从随机Zone的未使用IP中分配一个，若有标签，则一并填写标签
	AllocAddrNext(ctx context.Context, labels LabelMap) (net.IP, error)
	// 保留一个IP，该IP必须未分配
	ReserveAddr(ctx context.Context, specific string, labels LabelMap) error
	// 释放一个IP，该IP应当已分配或已保留，允许多次释放一个被多次申请的IP（内部引用计数减1）
	ReleaseAddr(ctx context.Context, specific string) error
	// 设置IP地址的标签，若IP格式错误或地址未分配/未保留则报错
	SetAddrLabel(ctx context.Context, specific, key, value string) error
	// 移除IP地址的标签，若IP格式错误或地址未分配/未保留则报错
	RemoveAddrLabel(ctx context.Context, specific, key string) error
	// 列出指定IP地址的所有标签，若IP格式错误或地址未分配/未保留则报错
	AddrLabels(ctx context.Context, specific string) (LabelMap, error)
	// 从IP地址查找所属Zone的literal，由于zone的literal不可能为空字符串，因此无需返回布尔值
	FindLiteral(ctx context.Context, specific string) string
	// 列出当前所有Zone的literal
	Literals(ctx context.Context) []string

	// 导出所有Zone为字节码，默认情况下不导出已分配的IP信息
	//
	// fat为true时，将会包含已分配的IP信息
	Dump(ctx context.Context, fat bool) ([]byte, error)
	// 将Zone中已分配的IP导出为分散的字节码映射
	DumpZoneAddrs(ctx context.Context, literal string, onlyKeys bool) (map[string][]byte, error)
	// 从Dump导出的字节码中加载全部Zone，IPAM中已有的同名Zone将会被覆盖
	Load(ctx context.Context, raw []byte) error
	// 从DumpZoneAddrs导出的字节码映射中加载Zone的已分配地址
	//
	// 若addrs的key不包含在该Zone中，则该key与其对应的地址将被忽略
	LoadZoneAddrs(ctx context.Context, literal string, addrs map[string][]byte) error
}
