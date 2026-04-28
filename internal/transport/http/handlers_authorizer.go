package http

import (
	"net/http"
	"streambear/internal/transport/http/authorizerapi"
)

// AuthorizerApiServer implements the authorizerapi.ServerInterface
type AuthorizerApiServer struct {
	authService *authorizer.Service
}

var _ authorizerapi.ServerInterface = (*AuthorizerApiServer)(nil)

// Implement all the handlers required by the authorizerapi.ServerInterface...
func (s *AuthorizerApiServer) AuthorizeLiveStream(w http.ResponseWriter, r *http.Request) {
	// ... logic
}
