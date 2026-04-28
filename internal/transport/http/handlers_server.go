package http

import (
	"net/http"
	"streambear/internal/transport/http/serverapi"
)

// ServerApiServer implements the serverapi.ServerInterface
type ServerApiServer struct {
	streamService *server.Service
}

var _ serverapi.ServerInterface = (*ServerApiServer)(nil)

// Implement all the handlers required by the serverapi.ServerInterface...
func (s *ServerApiServer) GetVideoStreamSegment(w http.ResponseWriter, r *http.Request, videoId string, segmentId string) {
	// ... logic
}
