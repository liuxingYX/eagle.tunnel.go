/*
 * @Author: EagleXiang
 * @Github: https://github.com/eaglexiang
 * @Date: 2018-12-13 18:54:13
 * @LastEditors: EagleXiang
 * @LastEditTime: 2019-03-17 17:00:52
 */

package cmd

import (
	"errors"
	"net"
	"strings"

	"github.com/eaglexiang/eagle.tunnel.go/src/core/protocols/et/comm"
	logger "github.com/eaglexiang/eagle.tunnel.go/src/logger"
	"github.com/eaglexiang/go-textcache"
	cache "github.com/eaglexiang/go-textcache"
	mytunnel "github.com/eaglexiang/go-tunnel"
)

// DNS ET-DNS子协议的实现
type DNS struct {
	DNSType        int
	dnsRemoteCache *cache.TextCache
	dnsLocalCache  *cache.TextCache
	DNSResolver    func(string) (string, error) `label:"DNS解析器"`
}

// Handle 处理ET-DNS请求
func (d DNS) Handle(req string, tunnel *mytunnel.Tunnel) error {
	reqs := strings.Split(req, " ")
	if len(reqs) < 2 {
		return errors.New("ETDNS.Handle -> req is too short")
	}
	e := comm.NetArg{NetConnArg: comm.NetConnArg{Domain: reqs[1]}}
	err := d.resolvDNSByLocal(&e)
	if err != nil {
		return err
	}
	_, err = tunnel.WriteLeft([]byte(e.IP))
	if err != nil {
		return err
	}
	return nil
}

// Send 发送ET-DNS请求
func (d DNS) Send(e *comm.NetArg) (err error) {
	ip, result := comm.HostsCache[e.Domain]
	if result {
		e.IP = ip
		return nil
	}
	switch comm.ETArg.ProxyStatus {
	case comm.ProxySMART:
		err = d.smartSend(e)
	case comm.ProxyENABLE:
		err = d.proxySend(e)
	default:
		logger.Error("dns.send invalid proxy-status")
		err = errors.New("invalid proxy-status")
	}
	return err
}

// smartSend 智能模式
// 智能模式会先检查域名是否存在于白名单
// 白名单内域名将转入强制代理模式
func (d DNS) smartSend(e *comm.NetArg) (err error) {
	white := IsWhiteDomain(e.Domain)
	if white {
		err = d.resolvDNSByProxy(e)
	} else {
		err = d.resolvDNSByLocal(e)
		// 判断IP所在位置是否适合代理
		comm.SubSenders[comm.EtLOCATION].Send(e)
		if !checkProxyByLocation(e.Location) {
			return nil
		}
		// 更新IP为Relay端的解析结果
		ne := &comm.NetArg{NetConnArg: comm.NetConnArg{Domain: e.Domain}}
		err = d.resolvDNSByProxy(ne)
		e.IP = ne.IP
	}
	return err
}

// proxySend 强制代理模式
func (d DNS) proxySend(e *comm.NetArg) error {
	return d.resolvDNSByProxy(e)
}

// Type ET子协议类型
func (d DNS) Type() int {
	return d.DNSType
}

// Name ET子协议的名字
func (d DNS) Name() string {
	return comm.FormatEtType(d.DNSType)
}

func (d *DNS) getCacheNodeOfRemote(domain string) (node *textcache.CacheNode, loaded bool) {
	if d.dnsRemoteCache == nil {
		d.dnsRemoteCache = cache.CreateTextCache()
	}
	return d.dnsRemoteCache.Get(domain)
}

func (d *DNS) getCacheNodeOfLocal(domain string) (node *cache.CacheNode, loaded bool) {
	if d.dnsLocalCache == nil {
		d.dnsLocalCache = cache.CreateTextCache()
	}
	return d.dnsLocalCache.Get(domain)
}

// resolvDNSByProxy 使用代理服务器进行DNS的解析
// 此函数主要完成缓存功能
// 当缓存不命中则调用 DNS._resolvDNSByProxy
func (d DNS) resolvDNSByProxy(e *comm.NetArg) (err error) {
	node, loaded := d.getCacheNodeOfRemote(e.Domain)
	if loaded {
		e.IP, err = node.Wait()
	} else {
		err = d._resolvDNSByProxy(e)
		if err != nil {
			d.dnsRemoteCache.Delete(e.Domain)
		} else {
			node.Update(e.IP)
		}
	}
	return err
}

// _resolvDNSByProxy 使用代理服务器进行DNS的解析
// 实际完成DNS查询操作
func (d DNS) _resolvDNSByProxy(e *comm.NetArg) (err error) {
	e.IP, err = sendQuery(d, e.Domain)
	ip := net.ParseIP(e.IP)
	if ip == nil {
		logger.Warning("fail to resolv dns by proxy: ", e.Domain, " -> ", e.IP)
		return errors.New("invalid reply")
	}
	return nil
}

// resolvDNSByLocal 本地解析DNS
// 此函数主要完成缓存功能
// 当缓存不命中则进一步调用 DNS._resolvDNSByLocalClient
func (d DNS) resolvDNSByLocal(e *comm.NetArg) (err error) {
	node, loaded := d.getCacheNodeOfLocal(e.Domain)
	if loaded {
		e.IP, err = node.Wait()
	} else {
		err = d._resolvDNSByLocal(e)
		if err != nil {
			d.dnsLocalCache.Delete(e.Domain)
		} else {
			node.Update(e.IP)
		}
	}
	return err
}

// _resolvDNSByLocalClient 本地解析DNS
// 实际完成DNS的解析动作
func (d DNS) _resolvDNSByLocal(e *comm.NetArg) (err error) {
	e.IP, err = d.DNSResolver(e.Domain)
	// 本地解析失败应该让用户察觉，手动添加DNS白名单
	if err != nil {
		logger.Warning("fail to resolv dns by local, ",
			"consider adding this domain to your whitelist_domain.txt: ",
			e.Domain)
	}
	return err
}

// IsWhiteDomain 判断域名是否是白名域名
func IsWhiteDomain(host string) (isWhite bool) {
	for _, line := range comm.WhitelistDomains {
		if strings.HasSuffix(host, line) {
			return true
		}
	}
	return false
}