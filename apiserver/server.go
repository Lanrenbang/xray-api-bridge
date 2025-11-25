package apiserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"xray-api-bridge/xrayapi"
)

// APIServer holds the HTTP server and its dependencies.
type APIServer struct {
	httpServer    *http.Server
	xrayClient    *xrayapi.Client
	subsConfigPath string

	// Store current listen address for reloading, though reload logic might need rework
	currentListenAddr string
}

// NewAPIServer creates a new APIServer instance.
func NewAPIServer(xrayClient *xrayapi.Client, listenAddr, subsConfigPath string) *APIServer {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through the chain of handlers, returning `http.StatusGatewayTimeout`
	// if the timeout is exceeded on the current request.
	r.Use(middleware.Timeout(60 * time.Second))

	apiServer := &APIServer{
		xrayClient: xrayClient,
		httpServer: &http.Server{
			Addr:         listenAddr,
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		subsConfigPath:    subsConfigPath,
		currentListenAddr: listenAddr,
	}

	apiServer.RegisterHandlers(r) // Register handlers on the created instance

	return apiServer
}

// Start starts the HTTP server.
func (s *APIServer) Start() error {
	log.Printf("HTTP server listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *APIServer) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

// GetHandler returns the underlying HTTP handler.
func (s *APIServer) GetHandler() http.Handler {
	return s.httpServer.Handler
}
