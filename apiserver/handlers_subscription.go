package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/infra/conf"
	jsonconf "github.com/xtls/xray-core/infra/conf/json"
)

// HandleSubscription generates subscription links based on the bridge's configuration.
func (s *APIServer) HandleSubscription(w http.ResponseWriter, r *http.Request) {
	// Check for mandatory gRPC client
	if s.xrayClient == nil || s.xrayClient.HandlerClient == nil {
		RespondWithError(w, http.StatusInternalServerError, "Xray gRPC client is not available. This feature requires a running Xray-core instance.")
		return
	}

	// (#B1) Mandate 'uuid' query parameter
	uuidQuery := r.URL.Query().Get("uuid")
	if uuidQuery == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing required 'uuid' query parameter.")
		return
	}

	// Load subscription profiles from the specified file
	subscriptionProfiles, err := loadSubscriptionProfiles(s.subsConfigPath)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load subscription config: %v", err))
		return
	}

	// --- Data Fetching (gRPC only) ---
	listInboundsResp, errGrpc := s.xrayClient.HandlerClient.ListInbounds(r.Context(), &command.ListInboundsRequest{})
	if errGrpc != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list inbounds via gRPC: %v", errGrpc))
		return
	}

	var inbounds []conf.InboundDetourConfig
	for _, coreInbound := range listInboundsResp.Inbounds {
		confInbound, reverseErr := ReverseInbound(coreInbound)
		if reverseErr != nil {
			fmt.Printf("Warning: failed to reverse map inbound %s from gRPC: %v\n", coreInbound.Tag, reverseErr)
			continue
		}
		inbounds = append(inbounds, *confInbound)
	}

	if len(inbounds) == 0 {
		RespondWithError(w, http.StatusNotFound, "No inbounds found in Xray-core.")
		return
	}

	// --- Link Generation ---
	links, err := s.generateSubscriptionLinks(inbounds, subscriptionProfiles, uuidQuery)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate subscription links: %v", err))
		return
	}

	if len(links) == 0 {
		// (#A2) If no links are generated, it might be because no matching protocols were found.
		RespondWithError(w, http.StatusNotFound, "No matching subscription links could be generated. Ensure the Xray-core instance is configured as a server with 'vless' or 'vmess' inbounds, and the provided 'uuid' is correct.")
		return
	}

	// --- Response ---
	RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: links})
}

// generateSubscriptionLinks creates share links from a slice of user-friendly InboundDetourConfig.
func (s *APIServer) generateSubscriptionLinks(inbounds []conf.InboundDetourConfig, profiles []SubscriptionProfile, uuidQuery string) ([]string, error) {
	type clientInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Flow  string `json:"flow"`
		Level int64  `json:"level"`
	}

	// 1. Segregate all clients by protocol and network
	categorizedClients := make(map[string][]clientInfo)
	foundSpecialProtocol := false
	for _, inbound := range inbounds {
		if inbound.Protocol != "vless" && inbound.Protocol != "vmess" {
			continue
		}
		foundSpecialProtocol = true

		inbNetwork := "tcp"
		if inbound.StreamSetting != nil && inbound.StreamSetting.Network != nil {
			inbNetwork = string(*inbound.StreamSetting.Network)
		}
		if inbNetwork == "tcp" {
			inbNetwork = "raw" // Alias
		}

		key := inbound.Protocol + "_" + inbNetwork

		var clients []clientInfo
		if inbound.Settings != nil {
			var settings struct {
				Clients json.RawMessage `json:"clients"`
			}
			if err := json.Unmarshal(*inbound.Settings, &settings); err == nil {
				json.Unmarshal(settings.Clients, &clients)
			}
		}
		if len(clients) > 0 {
			categorizedClients[key] = append(categorizedClients[key], clients...)
		}
	}

	if !foundSpecialProtocol {
		return nil, fmt.Errorf("subscription feature is only available on server-side configurations with 'vless' or 'vmess' inbounds")
	}

	// 2. (#B2, #B3) Filter clients based on uuidQuery
	superKey := os.Getenv("XRAY_API_BRIDGE_SUBS_SUPERKEY")
	useAllClients := (superKey != "" && uuidQuery == superKey)

	var targetIDs map[string]struct{}
	if !useAllClients {
		targetIDs = make(map[string]struct{})
		for _, id := range strings.Split(uuidQuery, ",") {
			targetIDs[strings.TrimSpace(id)] = struct{}{}
		}
	}

	// This map will hold all clients that match the query, categorized by their protocol+network key.
	filteredClients := make(map[string][]clientInfo)
	// This list preserves the original order of matched clients, which is crucial for device-based ordering.
	var orderedMatchedClients []clientInfo

	// To avoid duplicates in orderedMatchedClients when a user ID appears in multiple inbounds
	seenClients := make(map[string]bool)

	// Find the maximum number of clients in any category to determine the loop count
	maxClients := 0
	for _, clientList := range categorizedClients {
		if len(clientList) > maxClients {
			maxClients = len(clientList)
		}
	}

	// Iterate based on the maximum possible client index (like iterating through devices)
	for i := 0; i < maxClients; i++ {
		for key, clientList := range categorizedClients {
			if i < len(clientList) {
				client := clientList[i]
				match := useAllClients
				if !match {
					_, match = targetIDs[client.ID]
				}

				if match {
					if _, exists := filteredClients[key]; !exists {
						filteredClients[key] = []clientInfo{}
					}
					filteredClients[key] = append(filteredClients[key], client)

					if !seenClients[client.ID] {
						orderedMatchedClients = append(orderedMatchedClients, client)
						seenClients[client.ID] = true
					}
				}
			}
		}
	}

	// 3. Generate links
	var generatedLinks []string
	getRealityShortID := func(clientIndex int) string {
		for _, ib := range inbounds {
			if ib.Protocol == "vless" && ib.StreamSetting != nil && ib.StreamSetting.Security == "reality" && ib.StreamSetting.REALITYSettings != nil {
				shortIDs := ib.StreamSetting.REALITYSettings.ShortIds
				if len(shortIDs) > 0 {
					if clientIndex < len(shortIDs) {
						return shortIDs[clientIndex]
					}
					return shortIDs[len(shortIDs)-1] // Fallback to the last one
				}
			}
		}
		return ""
	}

	// Outer loop: Iterate through the matched clients in their original order (device order)
	for clientIndex, client := range orderedMatchedClients {
		// Inner loop: Iterate through subscription profiles to generate all links for this client
		for subIndex, sub := range profiles {
			subNetwork := sub.Network
			if subNetwork == "tcp" {
				subNetwork = "raw"
			}
			key := sub.Protocol + "_" + subNetwork

			// Check if this client exists for the profile's protocol/network type
			clientExistsForProfile := false
			for _, c := range categorizedClients[key] {
				if c.ID == client.ID {
					clientExistsForProfile = true
					break
				}
			}
			if !clientExistsForProfile {
				continue
			}

			if sub.Level != -1 && client.Level < sub.Level {
				continue
			}

			// --- Start of link generation logic ---
			port := sub.Port
			if port == 0 {
				port = 443
			}

			baseURL := fmt.Sprintf("%s://%s@%s:%d", sub.Protocol, client.ID, sub.Address, port)
			queryParams := url.Values{}

			if subNetwork != "raw" {
				queryParams.Add("type", subNetwork)
			}

			if sub.Encryption != "" {
				if sub.Protocol == "vless" && sub.Encryption != "none" {
					queryParams.Add("encryption", url.QueryEscape(sub.Encryption))
				} else if sub.Protocol == "vmess" && sub.Encryption != "auto" {
					queryParams.Add("encryption", sub.Encryption)
				}
			}

			security := sub.Security
			if security != "" && security != "none" {
				queryParams.Add("security", security)
			} else {
				security = "none"
			}

			var originalInbound *conf.InboundDetourConfig
			for i := range inbounds {
				ib := &inbounds[i]
				ibNet := "tcp"
				if ib.StreamSetting != nil && ib.StreamSetting.Network != nil {
					ibNet = string(*ib.StreamSetting.Network)
				}
				if ibNet == "tcp" {
					ibNet = "raw"
				}
				if ib.Protocol == sub.Protocol && ibNet == subNetwork {
					originalInbound = ib
					break
				}
			}

			switch subNetwork {
			case "xhttp":
				if sub.Host != "" {
					queryParams.Add("host", url.QueryEscape(sub.Host))
				} else if sub.Network == "http" || sub.Network == "xhttp" || sub.Network == "ws" || sub.Network == "httpupgrade" {
					queryParams.Add("host", url.QueryEscape(sub.Address))
				}
				if originalInbound != nil && originalInbound.StreamSetting != nil && originalInbound.StreamSetting.XHTTPSettings != nil {
					queryParams.Add("path", url.QueryEscape(originalInbound.StreamSetting.XHTTPSettings.Path))
				}
				mode := sub.Mode
				if mode == "" {
					mode = "auto"
				}
				queryParams.Add("mode", mode)

				if len(sub.Extra) > 2 { // not empty {}
					extraConfig := make(map[string]interface{})
					if err := json.Unmarshal(sub.Extra, &extraConfig); err == nil {
						if ds, ok := extraConfig["downloadSettings"].(map[string]interface{}); ok {
							if sec, ok := ds["security"].(string); ok && sec == "reality" {
								if rs, ok := ds["realitySettings"].(map[string]interface{}); ok {
									if _, ok := rs["shortId"]; !ok {
										sid := getRealityShortID(clientIndex)
										if sid != "" {
											rs["shortId"] = sid
										}
									}
									if _, ok := rs["spiderX"]; !ok {
										sid, _ := rs["shortId"].(string)
										spx := ""
										if len(sid) >= 8 {
											spx = "get-" + sid[len(sid)-8:]
										} else {
											spx = fmt.Sprintf("get-sub%05d", subIndex)
										}
										rs["spiderX"] = spx
									}
								}
							}
						}
						extraBytes, _ := json.Marshal(extraConfig)
						queryParams.Add("extra", url.QueryEscape(string(extraBytes)))
					}
				}

			case "http", "ws", "httpupgrade":
				if sub.Host != "" {
					host := sub.Host
					if subNetwork == "http" {
						host = strings.ReplaceAll(host, " ", "")
					}
					queryParams.Add("host", url.QueryEscape(host))
				} else {
					queryParams.Add("host", url.QueryEscape(sub.Address))
				}
				if sub.Path != "" && sub.Path != "/" {
					queryParams.Add("path", url.QueryEscape(sub.Path))
				}
			case "grpc":
				if originalInbound != nil && originalInbound.StreamSetting != nil && originalInbound.StreamSetting.GRPCSettings != nil {
					grpcSettings := originalInbound.StreamSetting.GRPCSettings
					serviceName := grpcSettings.ServiceName
					if strings.Contains(serviceName, "|") {
						serviceName = strings.Split(serviceName, "|")[0]
					}
					queryParams.Add("serviceName", url.QueryEscape(serviceName))

					mode := sub.Mode
					if mode == "" {
						mode = "gun"
					}
					queryParams.Add("mode", mode)
				}
			case "kcp":
				if originalInbound != nil && originalInbound.StreamSetting != nil && originalInbound.StreamSetting.KCPSettings != nil {
					kcpSettings := originalInbound.StreamSetting.KCPSettings
					if kcpSettings.HeaderConfig != nil {
						var header struct{ Type string `json:"type"` }
						if json.Unmarshal(kcpSettings.HeaderConfig, &header) == nil && header.Type != "none" {
							queryParams.Add("headerType", header.Type)
						}
					}
					if kcpSettings.Seed != nil {
						queryParams.Add("seed", url.QueryEscape(*kcpSettings.Seed))
					}
				}
			}

			switch security {
			case "tls":
				fp := sub.Fingerprint
				if fp != "" {
					queryParams.Add("fp", fp)
				}

				if sub.ServerName != "" {
					queryParams.Add("sni", sub.ServerName)
				}

				if len(sub.Alpn) > 0 {
					queryParams.Add("alpn", url.QueryEscape(strings.Join(sub.Alpn, ",")))
				}
				if sub.EchConfigList != "" {
					queryParams.Add("ech", url.QueryEscape(sub.EchConfigList))
				}
			case "reality":
				fp := sub.Fingerprint
				if fp == "" {
					fp = "chrome"
				}
				queryParams.Add("fp", fp)

				if sub.ServerName != "" {
					queryParams.Add("sni", sub.ServerName)
				}

				flow := client.Flow
				if sub.Flow != "" {
					flow = sub.Flow
				} else if flow == "" {
					// If sub.Flow is empty and client.Flow is empty, do not add the flow parameter.
				} else {
					queryParams.Add("flow", flow)
				}

				if sub.Password == "" {
					fmt.Printf("Warning: 'password' (pbk) is missing for a REALITY subscription profile (Address: %s). Skipping.\n", sub.Address)
					continue
				}
				queryParams.Add("pbk", sub.Password)

				if sub.Mldsa65Verify != "" {
					queryParams.Add("pqv", url.QueryEscape(sub.Mldsa65Verify))
				}

				sid := getRealityShortID(clientIndex)
				if sid != "" {
					queryParams.Add("sid", sid)
				}

				spx := ""
				if len(sid) >= 8 {
					spx = "get-" + sid[len(sid)-8:]
				} else {
					spx = fmt.Sprintf("get-sub%05d", subIndex)
				}
				queryParams.Add("spx", url.QueryEscape(spx))
			}

			finalURL := baseURL
			if encodedQuery := queryParams.Encode(); encodedQuery != "" {
				finalURL += "?" + encodedQuery
			}

			desp := sub.Description
			if desp == "" {
				desp = fmt.Sprintf("%s_%s_%s", sub.Protocol, subNetwork, security)
				if subNetwork == "xhttp" && len(sub.Extra) > 2 { // not empty {}
					var extraConfig struct {
						DownloadSettings struct {
							Security string `json:"security"`
						} `json:"downloadSettings"`
					}
					if json.Unmarshal(sub.Extra, &extraConfig) == nil {
						if dsSec := extraConfig.DownloadSettings.Security; dsSec != "" && security != dsSec {
							desp = fmt.Sprintf("%s2%s", desp, dsSec)
						}
					}
				}
			}
			finalURL += "#" + url.QueryEscape(desp)

			generatedLinks = append(generatedLinks, finalURL)
		}
	}

	return generatedLinks, nil
}

// loadSubscriptionProfiles loads the subscription profiles from a JSONC file.
func loadSubscriptionProfiles(path string) ([]SubscriptionProfile, error) {
	if path == "" {
		return nil, fmt.Errorf("subscription config path is not provided (XRAY_API_BRIDGE_SUBS_CONFIG)")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open subscription config file %s: %w", path, err)
	}
	defer file.Close()

	// Use a JSONC-compatible reader
	jsoncReader := &jsonconf.Reader{Reader: file}

	var profiles []SubscriptionProfile
	decoder := json.NewDecoder(jsoncReader)
	if err := decoder.Decode(&profiles); err != nil {
		return nil, fmt.Errorf("could not decode subscription config file %s: %w", path, err)
	}

	return profiles, nil
}