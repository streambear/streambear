Yes, absolutely. That is not only possible but is the **highly recommended and superior architectural pattern** for your multi-role application.

Using a separate OpenAPI spec for each role is a perfect example of applying strong **Separation of Concerns** at the API contract level. It aligns perfectly with the goals of a large-scale, cloud-native project.

Here’s the breakdown of why this is a great idea and how to implement it within your project structure.

### Why Separate Specs Are a Best Practice Here

1.  **Clear API Boundaries:** The `server` role has a completely different API surface (e.g., serving video segments, maybe a health check) than the `authorizer` role (e.g., debiting credits, issuing tokens). A single, massive OpenAPI file would be confusing and hard to manage.
2.  **Independent Evolution:** You can evolve the authorization API (e.g., add new grant types) without touching the server's API spec, and vice-versa. This reduces the cognitive load and the chance of accidental breaking changes.
3.  **Targeted Code Generation:** You generate code that is specific to each role. The stateless `server` binary doesn't need to contain the Go models or server interfaces for the `authorizer`'s complex API endpoints. This keeps each binary leaner.
4.  **Clearer Security & Infrastructure Mapping:** It's much easier to configure API gateways, load balancers, and firewalls.
    *   The `server` endpoints (e.g., `/stream/...`) might be exposed publicly via a CDN.
    *   The `authorizer` endpoints (e.g., `/authorize/...`) are likely internal, only accessible from your other backend services.
    Separate specs make this routing and security policy mapping explicit and clean.
5.  **Team Ownership:** In a larger team, the "Auth Team" can own `authorizer.yaml` and the "Streaming Team" can own `server.yaml`.

---

### How to Structure and Implement This

Let's update our project structure and workflow to support this pattern.

#### 1. Update the Directory Structure

Create a YAML file for each role within the `api/` directory. It's also a best practice to have a `common.yaml` for shared components like error responses.

```
streambear/
├── api/
│   ├── authorizer.yaml  <-- Spec for the authorizer role
│   ├── server.yaml      <-- Spec for the streaming server role
│   └── common.yaml      <-- (Optional but recommended) For shared schemas
└── ...
```

**Example `common.yaml`:**

```yaml
# api/common.yaml
openapi: 3.0.3
info:
  title: Streambear Common Types
  version: 1.0.0
paths: {} # No paths, just components
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
```

**Example `authorizer.yaml` referencing common types:**

```yaml
# api/authorizer.yaml
openapi: 3.0.3
info:
  title: Streambear Authorizer API
  version: 1.0.0
paths:
  /authorize/live:
    post:
      # ...
      responses:
        '400':
          description: Bad Request
          content:
            application/json:
              schema:
                $ref: 'common.yaml#/components/schemas/ErrorResponse' # Reference common schema
        # ...
```

#### 2. Update Code Generation

Your `Makefile` or `go:generate` comments will now have multiple commands, one for each spec. We will generate the code into separate sub-packages for clarity.

**Create sub-packages for generated code:**

```
internal/
└── transport/
    └── http/
        ├── authorizerapi/   <-- Generated code for authorizer
        │   └── generated.go
        ├── serverapi/       <-- Generated code for server
        │   └── generated.go
        ├── handlers_authorizer.go
        ├── handlers_server.go
        └── router.go
```

**Update your `Makefile`:**

```makefile
# Makefile

.PHONY: generate
generate: generate-authorizer generate-server

generate-authorizer:
	@echo "Generating authorizer API..."
	@oapi-codegen -generate "types,chi-server" \
		-package authorizerapi \
		-o internal/transport/http/authorizerapi/generated.go \
		api/authorizer.yaml

generate-server:
	@echo "Generating server API..."
	@oapi-codegen -generate "types,chi-server" \
		-package serverapi \
		-o internal/transport/http/serverapi/generated.go \
		api/server.yaml
```

#### 3. Implement the Interfaces in Separate Handler Files

You'll have separate handler structs, each implementing its own generated `ServerInterface`.

**`internal/transport/http/handlers_authorizer.go`:**

```go
package http

import "streambear/internal/transport/http/authorizerapi"

// AuthorizerApiServer implements the authorizerapi.ServerInterface
type AuthorizerApiServer struct {
    authService *authorizer.Service
}
var _ authorizerapi.ServerInterface = (*AuthorizerApiServer)(nil)

// Implement all the handlers required by the authorizerapi.ServerInterface...
func (s *AuthorizerApiServer) AuthorizeLiveStream(w http.ResponseWriter, r *http.Request) {
    // ... logic
}
```

**`internal/transport/http/handlers_server.go`:**

```go
package http

import "streambear/internal/transport/http/serverapi"

// ServerApiServer implements the serverapi.ServerInterface
type ServerApiServer struct {
    streamService *server.Service
}
var _ serverapi.ServerInterface = (*ServerApiServer)(nil)

// Implement all the handlers required by the serverapi.ServerInterface...
func (s *ServerApiServer) GetVideoStreamSegment(w http.ResponseWriter, r *http.Request, videoId string, segmentId string) {
    // ... logic
}
```

#### 4. Update the Router and `main.go`

Your `main.go` now conditionally builds and runs the router for the specific role.

**`internal/transport/http/router.go`:**

```go
package http

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
```

**`cmd/streambear/main.go` (The relevant `switch` block):**

```go
// ... inside main() function ...
switch role {
case "authorizer":
    authService := authorizer.NewService(creditStore, tokenService)
    router := http.NewAuthorizerRouter(authService) // <-- Use authorizer router
    log.Printf("Starting authorizer service on :%s", cfg.HTTPPort)
    log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, router))

case "server":
    streamService := server.NewService(tokenService)
    router := http.NewServerRouter(streamService) // <-- Use server router
    log.Printf("Starting streaming server on :%s", cfg.HTTPPort)
    log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, router))

// ...
}
```

This approach gives you a clean, maintainable, and highly professional structure that is perfectly suited for a large-scale, multi-faceted application like Streambear.