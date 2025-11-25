package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/vless/inbound"
	"github.com/xtls/xray-core/proxy/vless"
	proxyman "github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/reality"
	"github.com/xtls/xray-core/transport/internet/splithttp"
)

// JSONSuccessResponse defines the structure for a successful API response.
type JSONSuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// JSONErrorResponse defines the structure for an error API response.
type JSONErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// RespondWithJSON writes a JSON response with a given status code and payload.
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// RespondWithError writes a JSON error response.
func RespondWithError(w http.ResponseWriter, statusCode int, errorMessage string) {
	payload := JSONErrorResponse{
		Success: false,
		Error:   errorMessage,
	}
	RespondWithJSON(w, statusCode, payload)
}

// SimplifiedInboundResponse defines a user-friendly JSON structure for an inbound handler.
type SimplifiedInboundResponse struct {
	Tag              string      `json:"tag"`
	ReceiverSettings interface{} `json:"receiver_settings,omitempty"`
	ProxySettings    interface{} `json:"proxy_settings,omitempty"`
}

// SimplifiedOutboundResponse defines a user-friendly JSON structure for an outbound handler.
type SimplifiedOutboundResponse struct {
	Tag            string      `json:"tag"`
	SenderSettings interface{} `json:"sender_settings,omitempty"`
	ProxySettings  interface{} `json:"proxy_settings,omitempty"`
}

// JSONVlessUser is a struct for marshaling VLESS user info into a more readable JSON format.
type JSONVlessUser struct {
	Level   uint32      `json:"level"`
	Email   string      `json:"email"`
	Account interface{} `json:"account"`
}

// DecodeTypedMessage unmarshals a serial.TypedMessage into a concrete Go struct and then
// performs further decoding on nested fields if necessary.
func DecodeTypedMessage(msg *serial.TypedMessage) (interface{}, error) {
	if msg == nil {
		return nil, nil
	}

	// 直接尝试获取实例
	instance, err := msg.GetInstance()
	if err != nil {
		// 如果无法解码，返回原始数据但格式更友好
		return map[string]interface{}{
			"type":  msg.Type,
			"value": fmt.Sprintf("base64:%s", string(msg.Value)), // 简化显示
		}, nil
	}

		switch v := instance.(type) {
	case *inbound.Config:
		return decodeVlessInbound(v)
	case *proxyman.ReceiverConfig:
		return decodeReceiverConfig(v)
	case *internet.StreamConfig:
		return decodeStreamConfig(v)
	case *reality.Config:
		return decodeRealityConfig(v)
	case *splithttp.Config:
		result, err := decodeSplitHttpConfig(v)
		if err != nil {
			return map[string]interface{}{
				"type":  "splithttp.Config",
				"error": fmt.Sprintf("failed to decode splithttp: %v", err),
			}, nil
		}
		return result, nil
	case *vless.Account:
		return map[string]interface{}{
			"id":   v.Id,
			"flow": v.Flow,
		}, nil
	default:
		// 对于其他类型，尝试 JSON 序列化以获得更好的可读性
		jsonBytes, err := json.Marshal(instance)
		if err != nil {
			return map[string]interface{}{
				"type":  msg.Type,
				"error": fmt.Sprintf("failed to marshal: %v", err),
			}, nil
		}

		var result interface{}
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return map[string]interface{}{
				"type":  msg.Type,
				"value": string(jsonBytes),
			}, nil
		}

		return result, nil
	}
}

// decodeVlessInbound decodes the user accounts within a VLESS inbound config.
func decodeVlessInbound(config *inbound.Config) (interface{}, error) {
	// Protocol type for consumer-friendly format
	result := map[string]interface{}{
		"protocol":    "vless",
		"decryption":  config.Decryption,
		"fallbacks":   config.Fallbacks,
		"clients":     make([]map[string]interface{}, 0, len(config.Clients)),
	}

	for _, client := range config.Clients {
		decodedAccount, err := DecodeTypedMessage(client.Account)
		if err != nil {
			return nil, fmt.Errorf("failed to decode user account for email %s: %v", client.Email, err)
		}

		userData := map[string]interface{}{
			"level":  client.Level,
			"email":  client.Email,
		}

		// Merge account data
		if accountMap, ok := decodedAccount.(map[string]interface{}); ok {
			for k, v := range accountMap {
				userData[k] = v
			}
		}

		result["clients"] = append(result["clients"].([]map[string]interface{}), userData)
	}

	return result, nil
}

// decodeReceiverConfig decodes proxyman.ReceiverConfig into a more readable format
func decodeReceiverConfig(config *proxyman.ReceiverConfig) (interface{}, error) {
	result := map[string]interface{}{
		"listen":           config.Listen,
		"port_range":       config.PortList,
		"sniffing_enabled": config.SniffingSettings != nil,
	}

	if config.SniffingSettings != nil {
		result["sniffing_override"] = config.SniffingSettings.DestinationOverride
		result["sniffing_metadata_only"] = config.SniffingSettings.MetadataOnly
	}

	// Decode stream settings
	if config.StreamSettings != nil {
		decodedStream, err := DecodeTypedMessage(serial.ToTypedMessage(config.StreamSettings))
		if err != nil {
			// If decoding fails, return basic stream settings info
			result["stream_settings"] = map[string]interface{}{
				"protocol_name": config.StreamSettings.ProtocolName,
				"error": fmt.Sprintf("failed to decode stream settings: %v", err),
			}
		} else {
			result["stream_settings"] = decodedStream
		}
	}

	return result, nil
}

// decodeStreamConfig decodes internet.StreamConfig into a more readable format
func decodeStreamConfig(config *internet.StreamConfig) (interface{}, error) {
	result := map[string]interface{}{
		"protocol":  config.ProtocolName,
		"transport": config.TransportSettings,
	}

	// Determine security type from protobuf type
	securityType := "none"
	if config.SecurityType != "" {
		// Extract human-readable security type from protobuf type
		switch {
		case config.SecurityType == "xray.transport.internet.tls.Config":
			securityType = "tls"
		case config.SecurityType == "xray.transport.internet.reality.Config":
			securityType = "reality"
		default:
			securityType = config.SecurityType
		}
	}

	if securityType != "none" {
		result["security"] = securityType
	}

	if config.SocketSettings != nil {
		result["socket"] = map[string]interface{}{
			"accept_proxy_protocol": config.SocketSettings.AcceptProxyProtocol,
			"tcp_fast_open":         config.SocketSettings.Tfo,
			"tcp_congestion":        config.SocketSettings.TcpCongestion,
		}
	}

	// Decode security settings for better access
	if config.SecuritySettings != nil && len(config.SecuritySettings) > 0 {
		securitySettings := config.SecuritySettings[0]
		decodedSecurity, err := DecodeTypedMessage(securitySettings)
		if err == nil {
			result["security_settings"] = decodedSecurity
		}
	}

	// Decode transport settings for better access
	if config.TransportSettings != nil && len(config.TransportSettings) > 0 {
		decodedTransport := make([]map[string]interface{}, 0, len(config.TransportSettings))
		for _, ts := range config.TransportSettings {
			decodedTS, err := DecodeTypedMessage(ts.Settings)
			if err == nil {
				if transportMap, ok := decodedTS.(map[string]interface{}); ok {
					decodedTSMap := map[string]interface{}{
						"protocol_name": ts.ProtocolName,
						"settings": transportMap,
					}
					decodedTransport = append(decodedTransport, decodedTSMap)
				} else {
					// If decoding fails, provide fallback info
					decodedTransport = append(decodedTransport, map[string]interface{}{
						"protocol_name": ts.ProtocolName,
						"settings": map[string]interface{}{
							"type":  ts.Settings.Type,
							"value": fmt.Sprintf("base64:%s", string(ts.Settings.Value)),
						},
					})
				}
			} else {
				// If decoding fails, provide fallback info
				decodedTransport = append(decodedTransport, map[string]interface{}{
					"protocol_name": ts.ProtocolName,
					"settings": map[string]interface{}{
						"type":  ts.Settings.Type,
						"value": fmt.Sprintf("base64:%s", string(ts.Settings.Value)),
					},
				})
			}
		}
		result["transport_settings"] = decodedTransport
	}

	return result, nil
}

// decodeRealityConfig decodes reality.Config into a more readable format
func decodeRealityConfig(config *reality.Config) (interface{}, error) {
	result := map[string]interface{}{
		"show":       config.Show,
		"xver":       config.Xver,
		"serverNames": config.ServerNames,
		"privateKey":  config.PrivateKey,
		"maxTimeDiff": config.MaxTimeDiff,
		"shortIds":    config.ShortIds,
	}

	// Try to access additional fields if they exist
	// These might not exist in all versions, so we check them safely
	if config.Fingerprint != "" {
		result["fingerprint"] = config.Fingerprint
	}
	if config.ServerName != "" {
		result["serverName"] = config.ServerName
	}

	return result, nil
}

// decodeSplitHttpConfig decodes splithttp.Config into a more readable format
func decodeSplitHttpConfig(config *splithttp.Config) (interface{}, error) {
	result := map[string]interface{}{
		"path":         config.Path,
		"host":         config.Host,
		"mode":         config.Mode,
		"headers":      config.Headers,
	}

	// Add Xmux settings if they exist
	if config.Xmux != nil {
		result["xmux"] = map[string]interface{}{
			"maxConcurrency":     config.Xmux.MaxConcurrency,
			"maxConnections":     config.Xmux.MaxConnections,
			"cMaxReuseTimes":     config.Xmux.CMaxReuseTimes,
			"hMaxRequestTimes":   config.Xmux.HMaxRequestTimes,
			"hMaxReusableSecs":   config.Xmux.HMaxReusableSecs,
		}
	}

	return result, nil
}
