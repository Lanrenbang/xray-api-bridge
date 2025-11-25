package apiserver

import (
	"encoding/json"
	"fmt"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"

	"github.com/xtls/xray-core/proxy/blackhole"
	"github.com/xtls/xray-core/proxy/dokodemo"
	"github.com/xtls/xray-core/proxy/dns"
	"github.com/xtls/xray-core/proxy/freedom"
	"github.com/xtls/xray-core/proxy/http"
	"github.com/xtls/xray-core/proxy/loopback"
	"github.com/xtls/xray-core/proxy/socks"
	vless_inbound "github.com/xtls/xray-core/proxy/vless/inbound"
	vless_outbound "github.com/xtls/xray-core/proxy/vless/outbound"
	vmess_inbound "github.com/xtls/xray-core/proxy/vmess/inbound"
	vmess_outbound "github.com/xtls/xray-core/proxy/vmess/outbound"
	"github.com/xtls/xray-core/proxy/wireguard"
)

// ReverseInbound is the main factory function to reverse-map an inbound handler config.
func ReverseInbound(inbound *core.InboundHandlerConfig) (*conf.InboundDetourConfig, error) {
	if inbound.ReceiverSettings == nil {
		return nil, fmt.Errorf("receiver settings for inbound %s is nil", inbound.Tag)
	}
	instance, err := inbound.ReceiverSettings.GetInstance()
	if err != nil {
		return nil, err
	}
	receiverSettings := instance.(*proxyman.ReceiverConfig)

	portList := &conf.PortList{}
	if receiverSettings.PortList != nil {
		for _, p := range receiverSettings.PortList.Range {
			portList.Range = append(portList.Range, conf.PortRange{From: p.From, To: p.To})
		}
	}

	var listenOn *conf.Address
	if receiverSettings.Listen != nil {
		listenOn = &conf.Address{Address: receiverSettings.Listen.AsAddress()}
	}

	var sniffingConfig *conf.SniffingConfig
	if receiverSettings.SniffingSettings != nil {
		sniffingConfig = ReverseSniffing(receiverSettings.SniffingSettings)
	}

	var streamSetting *conf.StreamConfig
	if receiverSettings.StreamSettings != nil {
		streamSetting = ReverseStreamSettings(receiverSettings.StreamSettings)
	}

	confInbound := &conf.InboundDetourConfig{
		Tag:            inbound.Tag,
		ListenOn:       listenOn,
		PortList:       portList,
		SniffingConfig: sniffingConfig,
		StreamSetting:  streamSetting,
	}

	proxySettings := inbound.GetProxySettings()
	var protocolName string
	var settingsData json.RawMessage

	switch proxySettings.Type {
	case "xray.proxy.vless.inbound.Config":
		protocolName = "vless"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*vless_inbound.Config)
		settingsData, err = ReverseVlessInbound(config)
	case "xray.proxy.vmess.inbound.Config":
		protocolName = "vmess"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*vmess_inbound.Config)
		settingsData, err = ReverseVmessInbound(config)
	case "xray.proxy.wireguard.Config":
		protocolName = "wireguard"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*wireguard.DeviceConfig)
		settingsData, err = ReverseWireguardInbound(config)
	case "xray.proxy.dokodemo.Config":
		protocolName = "dokodemo-door"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*dokodemo.Config)
		settingsData, err = ReverseDokodemoInbound(config)
	case "xray.proxy.http.Config":
		protocolName = "http"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*http.ServerConfig)
		settingsData, err = ReverseHTTPInbound(config)
	case "xray.proxy.socks.Config":
		protocolName = "socks"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*socks.ServerConfig)
		settingsData, err = ReverseSocksInbound(config)
	default:
		return nil, fmt.Errorf("unsupported inbound protocol type: %s", proxySettings.Type)
	}

	if err != nil {
		return nil, err
	}

	confInbound.Protocol = protocolName
	confInbound.Settings = &settingsData

	return confInbound, nil
}

// ReverseOutbound is the main factory function to reverse-map an outbound handler config.
func ReverseOutbound(outbound *core.OutboundHandlerConfig) (*conf.OutboundDetourConfig, error) {
	instance, err := outbound.SenderSettings.GetInstance()
	if err != nil {
		return nil, err
	}
	senderSettings := instance.(*proxyman.SenderConfig)

	confOutbound := &conf.OutboundDetourConfig{
		Tag:           outbound.Tag,
		MuxSettings:   ReverseMux(senderSettings.MultiplexSettings),
		StreamSetting: ReverseStreamSettings(senderSettings.StreamSettings),
		ProxySettings: ReverseProxySettings(senderSettings.ProxySettings),
	}

	proxySettings := outbound.GetProxySettings()
	var protocolName string
	var settingsData json.RawMessage

	switch proxySettings.Type {
	case "xray.proxy.vless.outbound.Config":
		protocolName = "vless"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*vless_outbound.Config)
		settingsData, err = ReverseVlessOutbound(config)
	case "xray.proxy.vmess.outbound.Config":
		protocolName = "vmess"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*vmess_outbound.Config)
		settingsData, err = ReverseVmessOutbound(config)
	case "xray.proxy.wireguard.Config":
		protocolName = "wireguard"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*wireguard.DeviceConfig)
		settingsData, err = ReverseWireguardOutbound(config)
	case "xray.proxy.dns.Config":
		protocolName = "dns"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*dns.Config)
		settingsData, err = ReverseDNSOutbound(config)
	case "xray.proxy.blackhole.Config":
		protocolName = "blackhole"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*blackhole.Config)
		settingsData, err = ReverseBlackholeOutbound(config)
	case "xray.proxy.freedom.Config":
		protocolName = "freedom"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*freedom.Config)
		settingsData, err = ReverseFreedomOutbound(config)
	case "xray.proxy.loopback.Config":
		protocolName = "loopback"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*loopback.Config)
		settingsData, err = ReverseLoopbackOutbound(config)
	case "xray.proxy.http.Config":
		protocolName = "http"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*http.ClientConfig)
		settingsData, err = ReverseHTTPOutbound(config)
	case "xray.proxy.socks.Config":
		protocolName = "socks"
		instance, err := proxySettings.GetInstance()
		if err != nil {
			return nil, err
		}
		config := instance.(*socks.ClientConfig)
		settingsData, err = ReverseSocksOutbound(config)
	default:
		return nil, fmt.Errorf("unsupported outbound protocol type: %s", proxySettings.Type)
	}

	if err != nil {
		return nil, err
	}

	confOutbound.Protocol = protocolName
	confOutbound.Settings = &settingsData

	return confOutbound, nil
}
