package apiserver

import (
	"fmt"
	"net/http"
	"strconv"

	stats_command "github.com/xtls/xray-core/app/stats/command"
)

// handleGetSysStats handles the GET /stats/sys API request.
func (s *APIServer) handleGetSysStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := s.xrayClient.StatsClient.GetSysStats(r.Context(), &stats_command.SysStatsRequest{})
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get system stats: %v", err))
			return
		}
		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp})
	}
}

// handleGetNamedStats handles the GET /stats?name=<name>&reset=<bool> API request.
func (s *APIServer) handleGetNamedStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			RespondWithError(w, http.StatusBadRequest, "Stat name is required")
			return
		}

		resetStr := r.URL.Query().Get("reset")
		reset, _ := strconv.ParseBool(resetStr)

		req := &stats_command.GetStatsRequest{
			Name:  name,
			Reset_: reset,
		}

		resp, err := s.xrayClient.StatsClient.GetStats(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get named stat: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp.Stat})
	}
}

// handleQueryStats handles the GET /stats/query?pattern=<pattern>&reset=<bool> API request.
func (s *APIServer) handleQueryStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pattern := r.URL.Query().Get("pattern")
		if pattern == "" {
			RespondWithError(w, http.StatusBadRequest, "Stat pattern is required")
			return
		}

		resetStr := r.URL.Query().Get("reset")
		reset, _ := strconv.ParseBool(resetStr)

		req := &stats_command.QueryStatsRequest{
			Pattern: pattern,
			Reset_:  reset, // Use Reset_ as identified from example
		}

		resp, err := s.xrayClient.StatsClient.QueryStats(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query stats: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp.Stat})
	}
}

// handleGetStatsOnline handles the GET /stats/online?name=<name> API request.
func (s *APIServer) handleGetStatsOnline() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			RespondWithError(w, http.StatusBadRequest, "Stat name is required")
			return
		}

		req := &stats_command.GetStatsRequest{
			Name:  name,
			Reset_: false,
		}

		resp, err := s.xrayClient.StatsClient.GetStats(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get online stat: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp.Stat})
	}
}

// handleGetStatsOnlineIpList handles the GET /stats/online/iplist?name=<name> API request.
func (s *APIServer) handleGetStatsOnlineIpList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			RespondWithError(w, http.StatusBadRequest, "Stat name is required")
			return
		}

		req := &stats_command.GetStatsRequest{
			Name:  name,
			Reset_: false,
		}

		resp, err := s.xrayClient.StatsClient.GetStats(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get online IP list: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: resp.Stat})
	}
}
