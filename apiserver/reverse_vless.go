package apiserver

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vless/inbound"
	"github.com/xtls/xray-core/proxy/vless/outbound"
)

// VLessUserConfig is a user-facing struct for a VLESS user.
type VLessUserConfig struct {
	ID         string `json:"id"`
	Level      uint32 `json:"level"`
	Email      string `json:"email"`
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

// VLessInboundFallbackConfig is a user-facing struct for VLESS fallbacks.
type VLessInboundFallbackConfig struct {
	Name string `json:"name,omitempty"`
	Alpn string `json:"alpn,omitempty"`
	Path string `json:"path,omitempty"`
	Dest string `json:"dest,omitempty"`
	Xver uint64 `json:"xver,omitempty"`
}

// ReverseVlessInbound converts a vless.Config to a conf.VLessInboundConfig's settings
func ReverseVlessInbound(config *inbound.Config) (json.RawMessage, error) {
	var clientMessages []json.RawMessage
	for _, c := range config.Clients {
		account, err := c.GetTypedAccount()
		if err != nil {
			return nil, err
		}
		vlessAccount := account.(*vless.MemoryAccount)
		user := &VLessUserConfig{
			ID:    vlessAccount.ID.String(),
			Level: c.Level,
			Email: c.Email,
			Flow:  vlessAccount.Flow,
		}
		userBytes, err := json.Marshal(user)
		if err != nil {
			return nil, err
		}
		clientMessages = append(clientMessages, userBytes)
	}

	var fallbacks []*VLessInboundFallbackConfig
	for _, f := range config.Fallbacks {
		fallbacks = append(fallbacks, &VLessInboundFallbackConfig{
			Name: f.Name,
			Alpn: f.Alpn,
			Path: f.Path,
			Dest: f.Dest,
			Xver: f.Xver,
		})
	}

	settings := struct {
		Clients    []json.RawMessage           `json:"clients"`
		Decryption string                      `json:"decryption"`
		Fallbacks  []*VLessInboundFallbackConfig `json:"fallbacks"`
	}{
		Clients:    clientMessages,
		Decryption: config.Decryption,
		Fallbacks:  fallbacks,
	}

	return json.Marshal(settings)
}

// ReverseVlessOutbound converts a vless.Config to a conf.VLessOutboundConfig's settings
func ReverseVlessOutbound(config *outbound.Config) (json.RawMessage, error) {
	var vnext []*conf.VLessOutboundVnext
	s := config.Vnext
	var users []json.RawMessage
	u := s.User
	account, err := u.GetTypedAccount()
	if err != nil {
		return nil, err
	}
	vlessAccount := account.(*vless.MemoryAccount)
	user := &VLessUserConfig{
		ID:         vlessAccount.ID.String(),
		Level:      u.Level,
		Email:      u.Email,
		Flow:       vlessAccount.Flow,
		Encryption: vlessAccount.Encryption,
	}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	users = append(users, userBytes)
	vnext = append(vnext, &conf.VLessOutboundVnext{
		Address: &conf.Address{Address: s.Address.AsAddress()},
		Port:    uint16(s.Port),
		Users:   users,
	})

	settings := &conf.VLessOutboundConfig{
		Vnext: vnext,
	}

	return json.Marshal(settings)
}
