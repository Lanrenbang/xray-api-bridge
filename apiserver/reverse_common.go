package apiserver

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/reality"
	"github.com/xtls/xray-core/transport/internet/splithttp"
	"github.com/xtls/xray-core/transport/internet/tcp"
	"github.com/xtls/xray-core/transport/internet/tls"
)

// ReverseProxySettings converts *internet.ProxyConfig to *conf.ProxyConfig
func ReverseProxySettings(p *internet.ProxyConfig) *conf.ProxyConfig {
	if p == nil {
		return nil
	}
	return &conf.ProxyConfig{
		Tag:                 p.Tag,
		TransportLayerProxy: p.TransportLayerProxy,
	}
}

// ReverseMux converts *proxyman.MultiplexingConfig to *conf.MuxConfig
func ReverseMux(m *proxyman.MultiplexingConfig) *conf.MuxConfig {
	if m == nil {
		return nil
	}
	return &conf.MuxConfig{
		Enabled:         m.Enabled,
		Concurrency:     int16(m.Concurrency),
		XudpConcurrency: int16(m.XudpConcurrency),
		XudpProxyUDP443: m.XudpProxyUDP443,
	}
}

// ReverseSniffing converts *proxyman.SniffingConfig to *conf.SniffingConfig
func ReverseSniffing(s *proxyman.SniffingConfig) *conf.SniffingConfig {
	if s == nil {
		return nil
	}

	var destOverride conf.StringList = s.DestinationOverride
	var domainsExcluded conf.StringList = s.DomainsExcluded

	return &conf.SniffingConfig{
		Enabled:         s.Enabled,
		DestOverride:    &destOverride,
		DomainsExcluded: &domainsExcluded,
		MetadataOnly:    s.MetadataOnly,
		RouteOnly:       s.RouteOnly,
	}
}

// ReverseStreamSettings converts *internet.StreamConfig to *conf.StreamConfig
func ReverseStreamSettings(s *internet.StreamConfig) *conf.StreamConfig {
	if s == nil {
		return nil
	}

	// Reverse mapping for network protocol names
	var protocolName string
	switch s.ProtocolName {
	case "tcp":
		protocolName = "raw"
	case "splithttp":
		protocolName = "xhttp"
	default:
		protocolName = s.ProtocolName
	}
	protocol := conf.TransportProtocol(protocolName)

	// Reverse mapping for security types
	var securityName string
	switch s.SecurityType {
	case "xray.transport.internet.tls.Config":
		securityName = "tls"
	case "xray.transport.internet.reality.Config":
		securityName = "reality"
	case "":
		securityName = "none"
	default:
		securityName = s.SecurityType
	}

	cs := &conf.StreamConfig{
		Network:        &protocol,
		Security:       securityName,
		SocketSettings: ReverseSocketSettings(s.SocketSettings),
	}

	// Reverse mapping for transport settings
	for _, ts := range s.TransportSettings {
		instance, err := ts.Settings.GetInstance()
		if err != nil {
			continue
		}
		switch ts.ProtocolName {
		case "tcp":
			if config, ok := instance.(*tcp.Config); ok {
				cs.RAWSettings = &conf.TCPConfig{
					AcceptProxyProtocol: config.AcceptProxyProtocol,
					// HeaderConfig is not directly available in internet.TCPConfig, it's built from HeaderSettings
					// For now, we'll leave it as default or try to reverse map if needed later.
				}
			}
		case "splithttp":
			if config, ok := instance.(*splithttp.Config); ok {
				cs.XHTTPSettings = &conf.SplitHTTPConfig{
					Host: config.Host,
					Path: config.Path,
					Mode: config.Mode,
					Headers: config.Headers,
					// Populate other fields as needed from splithttp.Config
				}
			}
		}
	}

	if s.SecuritySettings != nil {
		for _, secSetting := range s.SecuritySettings {
			instance, err := secSetting.GetInstance()
			if err != nil {
				continue
			}
			if tlsSettings, ok := instance.(*tls.Config); ok {
				cs.TLSSettings = ReverseTLSSettings(tlsSettings)
			} else if realitySettings, ok := instance.(*reality.Config); ok {
				cs.REALITYSettings = ReverseREALITYSettings(realitySettings)
			}
		}
	}

	return cs
}

// ReverseTLSSettings converts *tls.Config to *conf.TLSConfig
func ReverseTLSSettings(t *tls.Config) *conf.TLSConfig {
	if t == nil {
		return nil
	}
	var certs []*conf.TLSCertConfig
	for _, cert := range t.Certificate {
		certs = append(certs, &conf.TLSCertConfig{
			Usage: cert.Usage.String(),
		})
	}
	return &conf.TLSConfig{
		ServerName:      t.ServerName,
		Insecure:        t.AllowInsecure,
		DisableSystemRoot: t.DisableSystemRoot,
		Certs:             certs,
	}
}

// ReverseREALITYSettings converts *reality.Config to *conf.REALITYConfig
func ReverseREALITYSettings(r *reality.Config) *conf.REALITYConfig {
	if r == nil {
		return nil
	}

	shortIDsStr := make([]string, len(r.ShortIds))
	for i, id := range r.ShortIds {
		shortIDsStr[i] = hex.EncodeToString(id)
	}

	var minClientVer string
	if len(r.MinClientVer) == 3 {
		minClientVer = fmt.Sprintf("%d.%d.%d", r.MinClientVer[0], r.MinClientVer[1], r.MinClientVer[2])
	}

	return &conf.REALITYConfig{
		Show:         r.Show,
		Dest:         json.RawMessage("\"" + r.Dest + "\""),
		Type:         r.Type,
		Xver:         r.Xver,
		ServerNames:  r.ServerNames,
		PrivateKey:   base64.RawURLEncoding.EncodeToString(r.PrivateKey),
		MinClientVer: minClientVer,
		ShortIds:     shortIDsStr,
		Fingerprint:  r.Fingerprint,
		ServerName:   r.ServerName,
		PublicKey:    base64.RawURLEncoding.EncodeToString(r.PublicKey),
		ShortId:      hex.EncodeToString(r.ShortId),
		SpiderX:      r.SpiderX,
	}
}

// ReverseSocketSettings converts *internet.SocketConfig to *conf.SocketConfig
func ReverseSocketSettings(s *internet.SocketConfig) *conf.SocketConfig {
	if s == nil {
		return nil
	}
	return &conf.SocketConfig{
		Mark:                 s.Mark,
		TFO:                  s.Tfo,
		TProxy:               s.Tproxy.String(),
		AcceptProxyProtocol:  s.AcceptProxyProtocol,
		DomainStrategy:       s.DomainStrategy.String(),
		DialerProxy:          s.DialerProxy,
		TCPKeepAliveInterval: s.TcpKeepAliveInterval,
		TCPKeepAliveIdle:     s.TcpKeepAliveIdle,
		TCPCongestion:        s.TcpCongestion,
		TCPWindowClamp:       s.TcpWindowClamp,
		TCPMaxSeg:            s.TcpMaxSeg,
		Penetrate:            s.Penetrate,
		TCPUserTimeout:       s.TcpUserTimeout,
		V6only:               s.V6Only,
		Interface:            s.Interface,
		TcpMptcp:             s.TcpMptcp,
		// CustomSockopt:        ReverseCustomSockopt(s.CustomSockopt), // Assuming ReverseCustomSockopt exists if needed
		AddressPortStrategy:  s.AddressPortStrategy.String(),
		// HappyEyeballsSettings: ReverseHappyEyeballsSettings(s.HappyEyeballs), // Assuming ReverseHappyEyeballsSettings exists if needed
	}
}
