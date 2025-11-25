package apiserver

import (
	"github.com/go-chi/chi/v5"
)

// RegisterHandlers registers all the API routes and their handlers.
func (s *APIServer) RegisterHandlers(r *chi.Mux) {
	r.Get("/status", s.HandleStatus)
	r.Get("/subscription", s.HandleSubscription)


	// StatsService
	r.Get("/stats/sys", s.handleGetSysStats())
	r.Get("/stats", s.handleGetNamedStats())
	r.Get("/stats/query", s.handleQueryStats())
	r.Get("/stats/online", s.handleGetStatsOnline())
	r.Get("/stats/online/iplist", s.handleGetStatsOnlineIpList())

	// HandlerService
	r.Get("/inbound", s.handleListInbounds())
	r.Post("/inbound", s.handleAddInbound())
	r.Delete("/inbound/{tag}", s.handleRemoveInbound())
	r.Put("/inbound/{tag}", s.handleAlterInbound())
	r.Post("/inbound/{tag}/users", s.handleAddInboundUsers())
	r.Delete("/inbound/{tag}/users", s.handleRemoveInboundUsers())
	r.Get("/inbound/{tag}/users", s.handleGetInboundUsers())
	r.Get("/inbound/{tag}/users/count", s.handleGetInboundUsersCount())

	r.Get("/outbound", s.handleListOutbounds())
	r.Post("/outbound", s.handleAddOutbound())
	r.Delete("/outbound/{tag}", s.handleRemoveOutbound())

	// RoutingService
	r.Post("/routing/rule", s.handleAddRoutingRule())
	r.Delete("/routing/rule/{tag}", s.handleRemoveRoutingRule())
	r.Get("/routing/balancer/{tag}", s.handleGetBalancerStats())
	r.Post("/routing/balancer/{tag}/choose", s.handleChooseOutbound())
	r.Post("/routing/blockip", s.handleBlockIP())

	// LoggerService
	r.Post("/logger/restart", s.handleRestartLogger())
}
