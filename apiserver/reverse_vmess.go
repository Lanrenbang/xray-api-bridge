package apiserver

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vmess"
	"github.com/xtls/xray-core/proxy/vmess/inbound"
	"github.com/xtls/xray-core/proxy/vmess/outbound"
)

// VMessUserConfig is a user-facing struct for a VMess user.
type VMessUserConfig struct {
	ID       string `json:"id"`
	Level    uint32 `json:"level"`
	Email    string `json:"email"`
	Security string `json:"security,omitempty"`
}

// ReverseVmessInbound converts a vmess.Config to a conf.VMessInboundConfig's settings
func ReverseVmessInbound(config *inbound.Config) (json.RawMessage, error) {
	var userMessages []json.RawMessage
	for _, u := range config.User {
		account, err := u.GetTypedAccount()
		if err != nil {
			return nil, err
		}
		vmessAccount := account.(*vmess.MemoryAccount)
		user := &VMessUserConfig{
			ID:       vmessAccount.ID.String(),
			Level:    u.Level,
			Email:    u.Email,
			Security: vmessAccount.Security.String(),
		}
		userBytes, err := json.Marshal(user)
		if err != nil {
			return nil, err
		}
		userMessages = append(userMessages, userBytes)
	}

	settings := struct {
		Users        []json.RawMessage `json:"clients"`
	}{
		Users: userMessages,
	}

	return json.Marshal(settings)
}

// ReverseVmessOutbound converts a vmess.Config to a conf.VMessOutboundConfig's settings
func ReverseVmessOutbound(config *outbound.Config) (json.RawMessage, error) {
	var vnext []*conf.VMessOutboundTarget
	s := config.Receiver
	var users []json.RawMessage
	account, err := s.User.GetTypedAccount()
	if err != nil {
		return nil, err
	}
	vmessAccount := account.(*vmess.MemoryAccount)
	user := &VMessUserConfig{
		ID:       vmessAccount.ID.String(),
		Level:    s.User.Level,
		Email:    s.User.Email,
		Security: vmessAccount.Security.String(),
	}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	users = append(users, userBytes)

	vnext = append(vnext, &conf.VMessOutboundTarget{
		Address: &conf.Address{Address: s.Address.AsAddress()},
		Port:    uint16(s.Port),
		Users:   users,
	})

	settings := &conf.VMessOutboundConfig{
		Receivers: vnext,
	}

	return json.Marshal(settings)
}
