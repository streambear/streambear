package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ... imports for chi, authorizerapi, serverapi ...

// NewAuthorizerRouter creates a router specifically for the authorizer role.
func NewAuthorizerRouter(authService *authorizer.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	apiServer := &AuthorizerApiServer{authService: authService}

	// Use the generated handler from the authorizer's package
	return authorizerapi.HandlerFromIntf(apiServer, r)
}

// NewServerRouter creates a router specifically for the server role.
func NewServerRouter(streamService *server.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	apiServer := &ServerApiServer{streamService: streamService}

	// Use the generated handler from the server's package
	return serverapi.HandlerFromIntf(apiServer, r)
}
