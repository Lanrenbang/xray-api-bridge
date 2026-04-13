package apiserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/infra/conf"
	jsonconf "github.com/xtls/xray-core/infra/conf/json"

	"xray-api-bridge/tokenauth"
)

// ---- Shared types ----

// clientInfo holds deserialized client data from an inbound's settings.
type clientInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow"`
	Level int64  `json:"level"`
}

// resolvedClients bundles all the data produced by resolveMatchedClients.
type resolvedClients struct {
	orderedClients     []clientInfo
	categorizedClients map[string][]clientInfo
	inbounds           []conf.InboundDetourConfig
}

// ---- Public Handlers ----

// HandleSubscription serves the subscription endpoint.
// Query parameter: token (encrypted, produced by /generateSubLinks).
// Response: Base64 encoded subscription URIs (text/plain).
func (s *APIServer) HandleSubscription(w http.ResponseWriter, r *http.Request) {
	if s.xrayClient == nil || s.xrayClient.HandlerClient == nil {
		RespondWithError(w, http.StatusInternalServerError, "Xray gRPC client is not available. This feature requires a running Xray-core instance.")
		return
	}

	// Validate auth secret availability
	if s.subsSecret == "" {
		RespondWithError(w, http.StatusInternalServerError, "Subscription auth secret (XRAY_API_BRIDGE_SUBS_AUTHSECRET) is not configured.")
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing required 'token' query parameter.")
		return
	}

	// Decrypt token to extract uuid query string
	uuidQuery, err := tokenauth.VerifyEncryptedToken(tokenStr, s.subsSecret)
	if err != nil {
		RespondWithError(w, http.StatusForbidden, fmt.Sprintf("Invalid or expired token: %v", err))
		return
	}

	profiles, err := loadSubscriptionProfiles(s.subsConfigPath)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load subscription config: %v", err))
		return
	}

	resolved, err := s.resolveMatchedClients(r.Context(), uuidQuery)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resolve clients: %v", err))
		return
	}

	links, err := generateSubscriptionLinks(resolved, profiles)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate subscription links: %v", err))
		return
	}

	if len(links) == 0 {
		RespondWithError(w, http.StatusNotFound, "No matching subscription links could be generated.")
		return
	}

	// Return Base64 encoded multi-line URI text
	raw := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, encoded)
}

// HandleGenerateSubLinks validates the uuid and returns a full subscription URL
// containing an encrypted token.
// Query parameter: uuid (same semantics as the old /subscription endpoint).
// Response: JSON with the full subscription URL.
func (s *APIServer) HandleGenerateSubLinks(w http.ResponseWriter, r *http.Request) {
	if s.xrayClient == nil || s.xrayClient.HandlerClient == nil {
		RespondWithError(w, http.StatusInternalServerError, "Xray gRPC client is not available.")
		return
	}

	if s.subsSecret == "" {
		RespondWithError(w, http.StatusInternalServerError, "Subscription auth secret (XRAY_API_BRIDGE_SUBS_AUTHSECRET) is not configured.")
		return
	}

	if s.subsURL == "" {
		RespondWithError(w, http.StatusInternalServerError, "Subscription URL (XRAY_API_BRIDGE_SUBS_URL) is not configured.")
		return
	}

	uuidQuery := r.URL.Query().Get("uuid")
	if uuidQuery == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing required 'uuid' query parameter.")
		return
	}

	// Validate uuid: either superkey or must match existing inbound clients
	superKey := os.Getenv("XRAY_API_BRIDGE_SUBS_SUPERKEY")
	isSuperKey := superKey != "" && uuidQuery == superKey

	if !isSuperKey {
		resolved, err := s.resolveMatchedClients(r.Context(), uuidQuery)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resolve clients: %v", err))
			return
		}
		if len(resolved.orderedClients) == 0 {
			RespondWithError(w, http.StatusForbidden, "The provided uuid does not match any active inbound client.")
			return
		}
	}

	// Generate encrypted token (default TTL: 30 days)
	token, err := tokenauth.GenerateEncryptedToken(s.subsSecret, uuidQuery, 30*24*time.Hour)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate token: %v", err))
		return
	}

	// Build full subscription URL
	fullURL := fmt.Sprintf("%s/subscription?token=%s", strings.TrimRight(s.subsURL, "/"), token)

	RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: fullURL})
}

// ---- Core Logic ----

// resolveMatchedClients fetches inbounds via gRPC, categorizes clients by
// protocol+network, and filters them by the given uuidQuery.
// This shared function is used by both /subscription and /generateSubLinks.
func (s *APIServer) resolveMatchedClients(ctx context.Context, uuidQuery string) (*resolvedClients, error) {
	listResp, err := s.xrayClient.HandlerClient.ListInbounds(ctx, &command.ListInboundsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list inbounds via gRPC: %w", err)
	}

	var inbounds []conf.InboundDetourConfig
	for _, core := range listResp.Inbounds {
		confIb, reverseErr := ReverseInbound(core)
		if reverseErr != nil {
			fmt.Printf("Warning: failed to reverse map inbound %s from gRPC: %v\n", core.Tag, reverseErr)
			continue
		}
		inbounds = append(inbounds, *confIb)
	}

	if len(inbounds) == 0 {
		return nil, fmt.Errorf("no inbounds found in Xray-core")
	}

	// Categorize clients by protocol_network key
	categorized := make(map[string][]clientInfo)
	for _, ib := range inbounds {
		if ib.Protocol != "vless" && ib.Protocol != "vmess" {
			continue
		}

		net := normalizeNetwork(ib.StreamSetting)
		key := ib.Protocol + "_" + net

		var clients []clientInfo
		if ib.Settings != nil {
			var settings struct {
				Clients json.RawMessage `json:"clients"`
			}
			if err := json.Unmarshal(*ib.Settings, &settings); err == nil {
				json.Unmarshal(settings.Clients, &clients)
			}
		}
		if len(clients) > 0 {
			categorized[key] = append(categorized[key], clients...)
		}
	}

	// Filter by uuidQuery
	superKey := os.Getenv("XRAY_API_BRIDGE_SUBS_SUPERKEY")
	useAll := superKey != "" && uuidQuery == superKey

	var targetIDs map[string]struct{}
	if !useAll {
		targetIDs = make(map[string]struct{})
		for _, id := range strings.Split(uuidQuery, ",") {
			targetIDs[strings.TrimSpace(id)] = struct{}{}
		}
	}

	// Cross-iterate to build orderedMatchedClients (device-based ordering)
	maxClients := 0
	for _, list := range categorized {
		if len(list) > maxClients {
			maxClients = len(list)
		}
	}

	seen := make(map[string]bool)
	var ordered []clientInfo

	for i := 0; i < maxClients; i++ {
		for _, list := range categorized {
			if i >= len(list) {
				continue
			}
			c := list[i]
			match := useAll
			if !match {
				_, match = targetIDs[c.ID]
			}
			if match && !seen[c.ID] {
				ordered = append(ordered, c)
				seen[c.ID] = true
			}
		}
	}

	return &resolvedClients{
		orderedClients:     ordered,
		categorizedClients: categorized,
		inbounds:           inbounds,
	}, nil
}

// generateSubscriptionLinks creates share link URIs from resolved clients and profiles.
func generateSubscriptionLinks(resolved *resolvedClients, profiles []SubscriptionProfile) ([]string, error) {
	inbounds := resolved.inbounds
	categorized := resolved.categorizedClients

	if len(categorized) == 0 {
		return nil, fmt.Errorf("subscription feature is only available on server-side configurations with 'vless' or 'vmess' inbounds")
	}

	getRealityShortID := func(clientIndex int) string {
		for _, ib := range inbounds {
			if ib.Protocol == "vless" && ib.StreamSetting != nil && ib.StreamSetting.Security == "reality" && ib.StreamSetting.REALITYSettings != nil {
				sids := ib.StreamSetting.REALITYSettings.ShortIds
				if len(sids) > 0 {
					if clientIndex < len(sids) {
						return sids[clientIndex]
					}
					return sids[len(sids)-1]
				}
			}
		}
		return ""
	}

	var links []string

	for clientIndex, client := range resolved.orderedClients {
		for subIndex, sub := range profiles {
			subNet := normalizeNetwork2(sub.Network)
			key := sub.Protocol + "_" + subNet

			// Verify client exists for this profile's protocol/network
			if !clientExistsInCategory(categorized[key], client.ID) {
				continue
			}

			// Level gate
			if sub.Level != -1 && client.Level < sub.Level {
				continue
			}

			link := buildShareLink(client, sub, subNet, inbounds, clientIndex, subIndex, getRealityShortID)
			if link != "" {
				links = append(links, link)
			}
		}
	}

	return links, nil
}

// ---- Link Building ----

// buildShareLink constructs a single vless:// or vmess:// share URI.
func buildShareLink(
	client clientInfo,
	sub SubscriptionProfile,
	subNet string,
	inbounds []conf.InboundDetourConfig,
	clientIndex, subIndex int,
	getRealityShortID func(int) string,
) string {
	port := sub.Port
	if port == 0 {
		port = 443
	}

	baseURL := fmt.Sprintf("%s://%s@%s:%d", sub.Protocol, client.ID, sub.Address, port)
	q := url.Values{}

	// ---- Protocol-specific fields ----

	// type (network)
	if subNet != "raw" {
		q.Add("type", subNet)
	}

	// encryption
	addEncryption(q, sub)

	// security
	security := sub.Security
	if security != "" && security != "none" {
		q.Add("security", security)
	} else {
		security = "none"
	}

	// flow - only for protocol=vless (no longer limited to security=reality)
	if sub.Protocol == "vless" {
		flow := client.Flow
		if sub.Flow != "" {
			flow = sub.Flow
		}
		if flow != "" {
			q.Add("flow", flow)
		}
	}

	// ---- Transport-specific fields ----

	originalInbound := findMatchingInbound(inbounds, sub.Protocol, subNet)

	switch subNet {
	case "xhttp":
		addXHTTPParams(q, sub, originalInbound, clientIndex, subIndex, getRealityShortID)
	case "http", "ws", "httpupgrade":
		addHTTPLikeParams(q, sub, subNet)
	case "grpc":
		addGRPCParams(q, sub)
	case "kcp":
		// kcp: headerType and seed removed, now handled by fm (finalmask)
	}

	// ---- Finalmask (fm) - all transport types ----
	addFinalmask(q, sub)

	// ---- TLS-specific fields ----
	switch security {
	case "tls":
		addTLSParams(q, sub)
	case "reality":
		addRealityParams(q, sub, client, clientIndex, subIndex, getRealityShortID)
	}

	// ---- Assemble final URL ----
	finalURL := baseURL
	if encoded := q.Encode(); encoded != "" {
		finalURL += "?" + encoded
	}

	// Description fragment
	desp := sub.Description
	if desp == "" {
		desp = buildDefaultDescription(sub, subNet, security)
	}
	finalURL += "#" + url.QueryEscape(desp)

	return finalURL
}

// ---- Parameter Helpers ----

func addEncryption(q url.Values, sub SubscriptionProfile) {
	if sub.Encryption == "" {
		return
	}
	if sub.Protocol == "vless" && sub.Encryption != "none" {
		q.Add("encryption", url.QueryEscape(sub.Encryption))
	} else if sub.Protocol == "vmess" && sub.Encryption != "auto" {
		q.Add("encryption", sub.Encryption)
	}
}

func addXHTTPParams(q url.Values, sub SubscriptionProfile, ib *conf.InboundDetourConfig, clientIdx, subIdx int, getSID func(int) string) {
	// host
	if sub.Host != "" {
		q.Add("host", url.QueryEscape(sub.Host))
	} else {
		q.Add("host", url.QueryEscape(sub.Address))
	}

	// path (from server-side inbound)
	if ib != nil && ib.StreamSetting != nil && ib.StreamSetting.XHTTPSettings != nil {
		q.Add("path", url.QueryEscape(ib.StreamSetting.XHTTPSettings.Path))
	}

	// mode
	mode := sub.Mode
	if mode == "" {
		mode = "auto"
	}
	q.Add("mode", mode)

	// extra (with reality shortId/spiderX injection)
	if len(sub.Extra) > 2 {
		extraConfig := make(map[string]interface{})
		if err := json.Unmarshal(sub.Extra, &extraConfig); err == nil {
			injectRealityIntoExtra(extraConfig, clientIdx, subIdx, getSID)
			extraBytes, _ := json.Marshal(extraConfig)
			q.Add("extra", url.QueryEscape(string(extraBytes)))
		}
	}
}

func addHTTPLikeParams(q url.Values, sub SubscriptionProfile, subNet string) {
	if sub.Host != "" {
		host := sub.Host
		if subNet == "http" {
			host = strings.ReplaceAll(host, " ", "")
		}
		q.Add("host", url.QueryEscape(host))
	} else {
		q.Add("host", url.QueryEscape(sub.Address))
	}
	if sub.Path != "" && sub.Path != "/" {
		q.Add("path", url.QueryEscape(sub.Path))
	}
}

func addGRPCParams(q url.Values, sub SubscriptionProfile) {
	// serviceName from config.path (not from server-side)
	if sub.Path != "" {
		q.Add("serviceName", url.QueryEscape(sub.Path))
	}

	// mode
	mode := sub.Mode
	if mode == "" {
		mode = "gun"
	}
	q.Add("mode", mode)

	// authority from config.host (empty does NOT fallback to address)
	if sub.Host != "" {
		q.Add("authority", url.QueryEscape(sub.Host))
	}
}

func addFinalmask(q url.Values, sub SubscriptionProfile) {
	if len(sub.Finalmask) > 2 { // not empty {}
		q.Add("fm", url.QueryEscape(string(sub.Finalmask)))
	}
}

func addTLSParams(q url.Values, sub SubscriptionProfile) {
	// fp - omit if chrome (default)
	fp := sub.Fingerprint
	if fp != "" && fp != "chrome" {
		q.Add("fp", fp)
	}

	// sni
	if sub.ServerName != "" {
		q.Add("sni", sub.ServerName)
	}

	// alpn
	if len(sub.Alpn) > 0 {
		q.Add("alpn", url.QueryEscape(strings.Join(sub.Alpn, ",")))
	}

	// ech
	if sub.EchConfigList != "" {
		q.Add("ech", url.QueryEscape(sub.EchConfigList))
	}

	// pcs & vcn (new TLS fields)
	if sub.PinnedPeerCertSha256 != "" {
		q.Add("pcs", url.QueryEscape(sub.PinnedPeerCertSha256))
	}
	if sub.VerifyPeerCertByName != "" {
		q.Add("vcn", url.QueryEscape(sub.VerifyPeerCertByName))
	}
}

func addRealityParams(q url.Values, sub SubscriptionProfile, client clientInfo, clientIdx, subIdx int, getSID func(int) string) {
	// fp - required, default chrome
	fp := sub.Fingerprint
	if fp == "" {
		fp = "chrome"
	}
	q.Add("fp", fp)

	// sni
	if sub.ServerName != "" {
		q.Add("sni", sub.ServerName)
	}

	// pbk (password) - mandatory for reality
	if sub.Password == "" {
		fmt.Printf("Warning: 'password' (pbk) is missing for a REALITY subscription profile (Address: %s). Skipping.\n", sub.Address)
		return
	}
	q.Add("pbk", sub.Password)

	// pqv
	if sub.Mldsa65Verify != "" {
		q.Add("pqv", url.QueryEscape(sub.Mldsa65Verify))
	}

	// sid
	sid := getSID(clientIdx)
	if sid != "" {
		q.Add("sid", sid)
	}

	// spx
	spx := ""
	if len(sid) >= 8 {
		spx = "get-" + sid[len(sid)-8:]
	} else {
		spx = fmt.Sprintf("get-sub%05d", subIdx)
	}
	q.Add("spx", url.QueryEscape(spx))
}

// ---- Utility Functions ----

// normalizeNetwork extracts the network type from StreamSettingsConfig and normalizes tcp→raw.
func normalizeNetwork(ss *conf.StreamConfig) string {
	net := "tcp"
	if ss != nil && ss.Network != nil {
		net = string(*ss.Network)
	}
	if net == "tcp" {
		net = "raw"
	}
	return net
}

// normalizeNetwork2 normalizes a network string from subscription profile config.
func normalizeNetwork2(network string) string {
	if network == "tcp" || network == "" {
		return "raw"
	}
	return network
}

// findMatchingInbound locates the first inbound matching the given protocol and normalized network.
func findMatchingInbound(inbounds []conf.InboundDetourConfig, protocol, subNet string) *conf.InboundDetourConfig {
	for i := range inbounds {
		ib := &inbounds[i]
		if ib.Protocol == protocol && normalizeNetwork(ib.StreamSetting) == subNet {
			return ib
		}
	}
	return nil
}

// clientExistsInCategory checks if a client ID appears in the given category list.
func clientExistsInCategory(list []clientInfo, id string) bool {
	for _, c := range list {
		if c.ID == id {
			return true
		}
	}
	return false
}

// injectRealityIntoExtra injects shortId and spiderX into the extra config's
// downloadSettings.realitySettings if applicable.
func injectRealityIntoExtra(extra map[string]interface{}, clientIdx, subIdx int, getSID func(int) string) {
	ds, ok := extra["downloadSettings"].(map[string]interface{})
	if !ok {
		return
	}
	sec, ok := ds["security"].(string)
	if !ok || sec != "reality" {
		return
	}
	rs, ok := ds["realitySettings"].(map[string]interface{})
	if !ok {
		return
	}

	if _, ok := rs["shortId"]; !ok {
		sid := getSID(clientIdx)
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
			spx = fmt.Sprintf("get-sub%05d", subIdx)
		}
		rs["spiderX"] = spx
	}
}

// buildDefaultDescription generates a fallback description when none is configured.
func buildDefaultDescription(sub SubscriptionProfile, subNet, security string) string {
	desp := fmt.Sprintf("%s_%s_%s", sub.Protocol, subNet, security)
	if subNet == "xhttp" && len(sub.Extra) > 2 {
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
	return desp
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

	jsoncReader := &jsonconf.Reader{Reader: file}

	var profiles []SubscriptionProfile
	decoder := json.NewDecoder(jsoncReader)
	if err := decoder.Decode(&profiles); err != nil {
		return nil, fmt.Errorf("could not decode subscription config file %s: %w", path, err)
	}

	return profiles, nil
}