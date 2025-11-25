package apiserver

import (
	"net/http"
)

// HandleStatus returns a simple success message.
func (s *APIServer) HandleStatus(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Xray API Bridge is running!",
	})
}
