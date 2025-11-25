package apiserver

import (
	"fmt"
	"net/http"

	log_command "github.com/xtls/xray-core/app/log/command"
)

// handleRestartLogger handles the POST /logger/restart API request.
func (s *APIServer) handleRestartLogger() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := s.xrayClient.LogClient.RestartLogger(r.Context(), &log_command.RestartLoggerRequest{})
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to restart logger: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{
			Success: true,
			Message: "Logger restarted successfully",
		})
	}
}
