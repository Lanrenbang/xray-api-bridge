package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	proxyman_command "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	"strings"
	vless "github.com/xtls/xray-core/proxy/vless"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handleAddInbound handles the POST /inbound API request.
func (s *APIServer) handleAddInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inboundConfig conf.InboundDetourConfig
		if err := json.NewDecoder(r.Body).Decode(&inboundConfig); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		inboundHandlerConfig, err := inboundConfig.Build()
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to build inbound config: %v", err))
			return
		}

		req := &proxyman_command.AddInboundRequest{
			Inbound: inboundHandlerConfig,
		}

		_, err = s.xrayClient.HandlerClient.AddInbound(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add inbound: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusCreated, JSONSuccessResponse{Success: true, Message: fmt.Sprintf("Inbound '%s' added successfully", inboundConfig.Tag)})
	}
}

// handleAlterInbound handles the PUT /inbound/{tag} API request to add or remove users.
func (s *APIServer) handleAlterInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}

		// For now, we only support adding a VLESS user.
		// A more robust implementation would inspect the request to decide whether to add/remove
		// and for which protocol.
		var simpleUser struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Flow  string `json:"flow"`
			Level uint32 `json:"level"`
		}

		if err := json.NewDecoder(r.Body).Decode(&simpleUser); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body for VLESS user: %v", err))
			return
		}

		// 1. Create the vless.Account from the simple user data
		vlessAccount := &vless.Account{
			Id:   simpleUser.ID,
			Flow: simpleUser.Flow,
		}

		// 2. Serialize the vless.Account into a serial.TypedMessage
		typedAccount := serial.ToTypedMessage(vlessAccount)

		// 3. Create the main protocol.User
		protoUser := &protocol.User{
			Level:   simpleUser.Level,
			Email:   simpleUser.Email,
			Account: typedAccount,
		}

		// 4. Create the AddUserOperation
		operation := &proxyman_command.AddUserOperation{
			User: protoUser,
		}

		// 5. Serialize the operation into the final TypedMessage for AlterInbound
		typedOperation := serial.ToTypedMessage(operation)

		// 6. Create and send the AlterInboundRequest
		req := &proxyman_command.AlterInboundRequest{
			Tag:       tag,
			Operation: typedOperation,
		}

		_, err := s.xrayClient.HandlerClient.AlterInbound(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to alter inbound: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: "Inbound altered successfully"})
	}
}

// handleListInbounds handles the GET /inbound API request.
func (s *APIServer) handleListInbounds() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &proxyman_command.ListInboundsRequest{}

		resp, err := s.xrayClient.HandlerClient.ListInbounds(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list inbounds: %v", err))
			return
		}

		// Create a new slice for our simplified response
		simplifiedInbounds := make([]*conf.InboundDetourConfig, 0, len(resp.GetInbounds()))

		for _, inbound := range resp.GetInbounds() {
			confInbound, err := ReverseInbound(inbound)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reverse map inbound %s: %v", inbound.Tag, err))
				return
			}
			simplifiedInbounds = append(simplifiedInbounds, confInbound)
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: simplifiedInbounds})
	}
}

// handleAddInboundUsers handles the POST /inbound/{tag}/users API request.
func (s *APIServer) handleAddInboundUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}

		// Parse request body for users
		var users []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Flow  string `json:"flow"`
			Level uint32 `json:"level"`
		}

		if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		if len(users) == 0 {
			RespondWithError(w, http.StatusBadRequest, "At least one user is required")
			return
		}

		// Process each user
		for _, user := range users {
			// Create the vless.Account from the user data
			vlessAccount := &vless.Account{
				Id:   user.ID,
				Flow: user.Flow,
			}

			// Serialize the vless.Account into a serial.TypedMessage
			typedAccount := serial.ToTypedMessage(vlessAccount)

			// Create the main protocol.User
			protoUser := &protocol.User{
				Level:   user.Level,
				Email:   user.Email,
				Account: typedAccount,
			}

			// Create the AddUserOperation
			operation := &proxyman_command.AddUserOperation{
				User: protoUser,
			}

			// Serialize the operation into the final TypedMessage for AlterInbound
			typedOperation := serial.ToTypedMessage(operation)

			// Create and send the AlterInboundRequest
			req := &proxyman_command.AlterInboundRequest{
				Tag:       tag,
				Operation: typedOperation,
			}

			_, err := s.xrayClient.HandlerClient.AlterInbound(r.Context(), req)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add user %s to inbound %s: %v", user.Email, tag, err))
				return
			}
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: fmt.Sprintf("%d users added to inbound '%s'", len(users), tag)})
	}
}

// handleRemoveInboundUsers handles the DELETE /inbound/{tag}/users API request.
func (s *APIServer) handleRemoveInboundUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}

		// Parse request body for emails to remove
		var request struct {
			Emails []string `json:"emails"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		if len(request.Emails) == 0 {
			RespondWithError(w, http.StatusBadRequest, "At least one email is required")
			return
		}

		// Process each email
		for _, email := range request.Emails {
			// Create the RemoveUserOperation
			operation := &proxyman_command.RemoveUserOperation{
				Email: email,
			}

			// Serialize the operation into TypedMessage for AlterInbound
			typedOperation := serial.ToTypedMessage(operation)

			// Create and send the AlterInboundRequest
			req := &proxyman_command.AlterInboundRequest{
				Tag:       tag,
				Operation: typedOperation,
			}

			_, err := s.xrayClient.HandlerClient.AlterInbound(r.Context(), req)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove user %s from inbound %s: %v", email, tag, err))
				return
			}
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: fmt.Sprintf("%d users removed from inbound '%s'", len(request.Emails), tag)})
	}
}

// handleGetInboundUsers handles the GET /inbound/{tag}/users API request.
func (s *APIServer) handleGetInboundUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}
		email := r.URL.Query().Get("email")

		req := &proxyman_command.GetInboundUserRequest{
			Tag:   tag,
			Email: email,
		}

		resp, err := s.xrayClient.HandlerClient.GetInboundUsers(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get inbound users: %v", err))
			return
		}

		// Decode user accounts to human-readable and flattened format
		users := resp.GetUsers()
		decodedUsers := make([]map[string]interface{}, 0, len(users))

		for _, user := range users {
			decodedUser := map[string]interface{}{
				"level": user.Level,
				"email": user.Email,
			}

			// Decode and flatten the account field
			if user.Account != nil {
				var accountDetails map[string]interface{}

				// Try to decode the account into a map using the generic decoder first
				decodedAccount, err := DecodeTypedMessage(user.Account)
				if err == nil {
					if accMap, ok := decodedAccount.(map[string]interface{}); ok {
						accountDetails = accMap
					}
				} else {
					// If generic decoding fails, fall back to the specific VLESS account type
					if instance, instanceErr := user.Account.GetInstance(); instanceErr == nil {
						if account, ok := instance.(*vless.Account); ok {
							// Manually create a map for the VLESS account
							accountDetails = map[string]interface{}{
								"id":   account.Id,
								"flow": account.Flow,
							}
						}
					}
				}

				// Merge the flattened account details into the main user object
				if accountDetails != nil {
					for key, value := range accountDetails {
						decodedUser[key] = value
					}
				} else {
					// Fallback for unknown types that couldn't be decoded into a map
					decodedUser["account_raw"] = map[string]interface{}{
						"type":  user.Account.Type,
						"value": fmt.Sprintf("base64:%s", string(user.Account.Value)),
					}
				}
			}

			decodedUsers = append(decodedUsers, decodedUser)
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: decodedUsers})
	}
}

// handleGetInboundUsersCount handles the GET /inbound/{tag}/users/count API request.
func (s *APIServer) handleGetInboundUsersCount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}
		email := r.URL.Query().Get("email")

		req := &proxyman_command.GetInboundUserRequest{
			Tag:   tag,
			Email: email,
		}

		resp, err := s.xrayClient.HandlerClient.GetInboundUsersCount(r.Context(), req)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get inbound users count: %v", err))
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Data: struct {
			Count int64 `json:"count"`
		}{Count: resp.GetCount()}})
	}
}

// handleRemoveInbound handles the DELETE /inbound/{tag} API request.
func (s *APIServer) handleRemoveInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		if tag == "" {
			RespondWithError(w, http.StatusBadRequest, "Inbound tag is required")
			return
		}

		req := &proxyman_command.RemoveInboundRequest{
			Tag: tag,
		}

		_, err := s.xrayClient.HandlerClient.RemoveInbound(r.Context(), req)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && (st.Code() == codes.NotFound || strings.Contains(st.Message(), common.ErrNoClue.Error())) {
				RespondWithError(w, http.StatusNotFound, fmt.Sprintf("Inbound '%s' not found", tag))
			} else {
				RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove inbound: %v", err))
			}
			return
		}

		RespondWithJSON(w, http.StatusOK, JSONSuccessResponse{Success: true, Message: fmt.Sprintf("Inbound '%s' removed successfully", tag)})
	}
}