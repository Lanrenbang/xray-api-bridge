package apiserver

import "encoding/json"

// SimplifiedUser defines a simplified, flat user structure for VLESS user configuration.
type SimplifiedUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow,omitempty"`
	Level uint32 `json:"level,omitempty"`
}

// SubscriptionProfile defines the structure for a single subscription generation profile.
// This maps to an entry in the subscription.jsonc array.
type SubscriptionProfile struct {
	Level         int64           `json:"level"`
	Address       string          `json:"address"`
	Port          uint16          `json:"port"`
	Protocol      string          `json:"protocol"`
	Network       string          `json:"network"`
	Security      string          `json:"security"`
	Description   string          `json:"description"`
	Encryption    string          `json:"encryption"`
	Fingerprint   string          `json:"fingerprint"`
	ServerName    string          `json:"serverName"`
	Flow          string          `json_"flow"`
	Password      string          `json:"password"`
	Mldsa65Verify string          `json:"mldsa65Verify"`
	Alpn          []string        `json:"alpn"`
	EchConfigList string          `json:"echConfigList"`
	Host          string          `json:"host"`
	Mode          string          `json:"mode"`
	Extra         json.RawMessage `json:"extra"`
	Path          string          `json:"path"`
}