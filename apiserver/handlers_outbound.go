package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	proxyman_command "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/infra/conf"
)

// handleListOutbounds handles the GET /outbound API request.
func (s *APIServer) handleListOutbounds() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &proxyman_command.ListOutboundsRequest{}

		resp, err := s.xrayClient.HandlerClient.ListOutbounds(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list outbounds: %v", err))
			return
		}

		// Create a new slice for our simplified response
		simplifiedOutbounds := make([]*conf.OutboundDetourConfig, 0, len(resp.GetOutbounds()))

		for _, outbound := range resp.GetOutbounds() {
			confOutbound, err := ReverseOutbound(outbound)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reverse map outbound %s: %v", outbound.Tag, err))
				return
			}
			simplifiedOutbounds = append(simplifiedOutbounds, confOutbound)
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: simplifiedOutbounds})
	}
}

// handleAddOutbound handles the POST /outbound API request.
func (s *APIServer) handleAddOutbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var outboundConfig conf.OutboundDetourConfig
		if err := json.NewDecoder(r.Body).Decode(&outboundConfig); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		outboundHandlerConfig, err := outboundConfig.Build()
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to build outbound config: %v", err))
			return
		}

		req := &proxyman_command.AddOutboundRequest{
			Outbound: outboundHandlerConfig,
		}

		_, err = s.xrayClient.HandlerClient.AddOutbound(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add outbound: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusCreated, JSONSuccessResponse{Success: true, Message: fmt.Sprintf("Outbound '%s' added successfully", outboundConfig.Tag)})
	}
}

// handleRemoveOutbound handles the DELETE /outbound/{tag} API request.
func (s *APIServer) handleRemoveOutbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Outbound tag is required")
			return
		}

		req := &proxyman_command.RemoveOutboundRequest{
			Tag: tag,
		}

		_, err := s.xrayClient.HandlerClient.RemoveOutbound(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove outbound: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: "Outbound removed successfully"})
	}
}

