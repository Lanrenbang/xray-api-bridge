package apiserver

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/blackhole"
	"github.com/xtls/xray-core/proxy/dokodemo"
	"github.com/xtls/xray-core/proxy/dns"
	"github.com/xtls/xray-core/proxy/freedom"
	"github.com/xtls/xray-core/proxy/http"
	"github.com/xtls/xray-core/proxy/loopback"
	"github.com/xtls/xray-core/proxy/socks"
	"github.com/xtls/xray-core/proxy/wireguard"
)

// ReverseWireguardInbound reverse-maps a wireguard.DeviceConfig to a conf.WireGuardConfig
func ReverseWireguardInbound(config *wireguard.DeviceConfig) (json.RawMessage, error) {
	peers := make([]*conf.WireGuardPeerConfig, len(config.Peers))
	for i, p := range config.Peers {
		peers[i] = &conf.WireGuardPeerConfig{
			PublicKey:    p.PublicKey,
			PreSharedKey: string(p.PreSharedKey),
			Endpoint:     p.Endpoint,
			KeepAlive:    p.KeepAlive,
			AllowedIPs:   p.AllowedIps,
		}
	}

	wgConfig := &conf.WireGuardConfig{
		SecretKey:  config.SecretKey,
		Address:    config.Endpoint,
		Peers:      peers,
		MTU:        config.Mtu,
		NumWorkers: config.NumWorkers,
		Reserved:   config.Reserved,
	}
	return json.Marshal(wgConfig)
}

// ReverseDokodemoInbound reverse-maps a dokodemo.Config to a conf.DokodemoConfig
func ReverseDokodemoInbound(config *dokodemo.Config) (json.RawMessage, error) {
	networks := make([]conf.Network, len(config.Networks))
	for i, n := range config.Networks {
		networks[i] = conf.Network(n.String())
	}
	dkConfig := &conf.DokodemoConfig{
		Address:        &conf.Address{Address: config.Address.AsAddress()},
		Port:           uint16(config.Port),
		Network:        (*conf.NetworkList)(&networks),
		FollowRedirect: config.FollowRedirect,
		UserLevel:      config.UserLevel,
	}
	return json.Marshal(dkConfig)
}

// ReverseHTTPInbound reverse-maps a http.ServerConfig to a conf.HTTPConfig
func ReverseHTTPInbound(config *http.ServerConfig) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseSocksInbound reverse-maps a socks.ServerConfig to a conf.SocksConfig
func ReverseSocksInbound(config *socks.ServerConfig) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseDNSOutbound reverse-maps a dns.Config to a conf.DNSOutboundConfig
func ReverseDNSOutbound(config *dns.Config) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseBlackholeOutbound reverse-maps a blackhole.Config to a conf.BlackholeConfig
func ReverseBlackholeOutbound(config *blackhole.Config) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseFreedomOutbound reverse-maps a freedom.Config to a conf.FreedomConfig
func ReverseFreedomOutbound(config *freedom.Config) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseLoopbackOutbound reverse-maps a loopback.Config to a conf.LoopbackConfig
func ReverseLoopbackOutbound(config *loopback.Config) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseHTTPOutbound reverse-maps a http.ClientConfig to a conf.HTTPOutboundConfig
func ReverseHTTPOutbound(config *http.ClientConfig) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseSocksOutbound reverse-maps a socks.ClientConfig to a conf.SocksOutboundConfig
func ReverseSocksOutbound(config *socks.ClientConfig) (json.RawMessage, error) {
	return json.Marshal(config)
}

// ReverseWireguardOutbound reverse-maps a wireguard.DeviceConfig to a conf.WireGuardConfig
func ReverseWireguardOutbound(config *wireguard.DeviceConfig) (json.RawMessage, error) {
	return ReverseWireguardInbound(config)
}
