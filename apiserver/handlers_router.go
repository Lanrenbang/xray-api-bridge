package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	router "github.com/xtls/xray-core/app/router"
	router_command "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	proto "google.golang.org/protobuf/proto"
)

// handleAddRoutingRule handles the POST /routing/rule API request.
func (s *APIServer) handleAddRoutingRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rawRule json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawRule); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		rule, err := conf.ParseRule(rawRule)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse routing rule: %v", err))
			return
		}

		config := &router.Config{
			Rule: []*router.RoutingRule{rule},
		}

		configBytes, err := proto.Marshal(config)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to marshal routing config: %v", err))
			return
		}

		typedConfig := &serial.TypedMessage{
			Type:  "xray.app.router.Config",
			Value: configBytes,
		}

		addReq := &router_command.AddRuleRequest{
			Config: typedConfig,
		}

		_, err = s.xrayClient.RouterClient.AddRule(r.Context(), addReq)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add routing rule: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusCreated, JSONSuccessResponse{Success: true, Message: "Routing rule added successfully"})
	}
}

// handleRemoveRoutingRule handles the DELETE /routing/rule/{tag} API request.
func (s *APIServer) handleRemoveRoutingRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Rule tag is required")
			return
		}

		req := &router_command.RemoveRuleRequest{
			RuleTag: tag,
		}

		_, err := s.xrayClient.RouterClient.RemoveRule(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove routing rule: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: "Routing rule removed successfully"})
	}
}

// handleGetBalancerStats handles the GET /routing/balancer/{tag} API request.
func (s *APIServer) handleGetBalancerStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Balancer tag is required")
			return
		}

		req := &router_command.GetBalancerInfoRequest{
			Tag: tag,
		}

		resp, err := s.xrayClient.RouterClient.GetBalancerInfo(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get balancer stats: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp})
	}
}

// handleChooseOutbound handles the POST /routing/balancer/{tag}/choose API request.
func (s *APIServer) handleChooseOutbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Balancer tag is required")
			return
		}

		var chooseReq struct {
			OutboundTag string `json:"outboundTag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&chooseReq); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		req := &router_command.OverrideBalancerTargetRequest{
			BalancerTag: tag,
			Target:      chooseReq.OutboundTag,
		}

		_, err := s.xrayClient.RouterClient.OverrideBalancerTarget(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to choose outbound: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: "Outbound chosen successfully"})
	}
}

// handleBlockIP handles the POST /routing/blockip API request (sib command).

func (s *APIServer) handleBlockIP() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var req struct {

			IPs         []string `json:"ips"`

			InboundTag  string   `json:"inboundTag"`

			OutboundTag string   `json:"outboundTag"`

			RuleTag     string   `json:"ruleTag,omitempty"`

			Reset       bool     `json:"reset,omitempty"`

		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))

			return

		}



		if req.RuleTag == "" {

			req.RuleTag = "sourceIpBlock"

		}



		if req.Reset {

			removeReq := &router_command.RemoveRuleRequest{

				RuleTag: req.RuleTag,

			}

			// We don't care about the error here, as the rule might not exist.

			_, _ = s.xrayClient.RouterClient.RemoveRule(r.Context(), removeReq)

		}



		// Construct a map that represents the RuleObject JSON

		ruleMap := map[string]interface{}{

			"ruleTag":     req.RuleTag,

			"inboundTag":  []string{req.InboundTag},

			"outboundTag": req.OutboundTag,

			"ip":          req.IPs,

		}



		rawRule, err := json.Marshal(ruleMap)

		if err != nil {

			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to marshal rule map: %v", err))

			return

		}



		rule, err := conf.ParseRule(rawRule)

		if err != nil {

			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse routing rule: %v", err))

			return

		}



		config := &router.Config{

			Rule: []*router.RoutingRule{rule},

		}



		configBytes, err := proto.Marshal(config)

		if err != nil {

			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to marshal routing config: %v", err))

			return

		}



		typedConfig := &serial.TypedMessage{

			Type:  "xray.app.router.Config",

			Value: configBytes,

		}



		addReq := &router_command.AddRuleRequest{

			Config: typedConfig,

		}



		_, err = s.xrayClient.RouterClient.AddRule(r.Context(), addReq)

		if err != nil {

			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add routing rule: %v", err))

			return

		}



		RespondWithJSON(w, http.StatusCreated, JSONSuccessResponse{Success: true, Message: "Routing rule for blocking IPs added successfully"})

	}

}




