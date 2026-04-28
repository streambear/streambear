## **Streambear**

### **1. Introduction & Vision**

#### 1.1. Overview

Streambear is a high-performance, developer-focused video server for delivering scalable and secure VOD (Video on Demand) and live streaming content. It is designed from the ground up as a cloud-native application, written in Go, to run on container orchestration platforms like Kubernetes.

Its core architectural principle is the decoupling of the high-traffic streaming workload from stateful backend services. This is achieved by using a stateless streaming component where access is granted via short-lived, verifiable cryptographic tokens (JWTs), eliminating database bottlenecks at the edge for maximum performance and scalability.

#### 1.2. Target Audience

This document is intended for software engineers, DevOps professionals, and system architects involved in the development, deployment, and maintenance of the Streambear platform.

### **2. Core Architectural Principles**

*   **Stateless Streaming Nodes:** The components handling video segment delivery are completely stateless. All information required to authorize a request is contained within the request itself (via a JWT), enabling effortless horizontal scaling.
*   **Security by Design:** Security is addressed at two layers: application-level (who can watch what) and infrastructure-level (which services can talk to each other).
*   **Contract-First APIs:** The API is the product. All public-facing and internal service APIs are defined first using the OpenAPI 3 specification. This contract is the source of truth from which server code and documentation are generated.
*   **Single Binary, Multi-Role Deployment:** The entire application is compiled into a single Go binary. This binary can run in different "roles" (e.g., `authorizer`, `server`) by passing a command-line flag, simplifying the build and deployment pipeline and eliminating version skew between components.
*   **Cloud-Native & Agnostic:** The system is designed to run natively on Kubernetes and leverage its ecosystem (e.g., service meshes), but does not rely on any proprietary cloud provider services.

### **3. System Architecture**

#### 3.1. Component Overview

The Streambear ecosystem consists of three primary service roles, a backing database, and the client.

*   **Authorizer Service (Stateful):**
    *   The single source of truth for user data, credit balances, and content ownership.
    *   Connects to the primary PostgreSQL database.
    *   **Responsibilities:** Manages user accounts, debits credits, and, most importantly, **issues signed JWTs** upon successful authorization requests from a client's backend.
*   **Streaming Service (Stateless):**
    *   The high-performance, horizontally scalable workhorse.
    *   **Responsibilities:** Receives requests for video segments. It **validates the JWT** attached to each request (signature, expiration, claims) and, if valid, serves the requested video data from storage (e.g., object storage like S3). It **never** communicates with the database.
*   **Uploader Service (Stateful):**
    *   Handles the content ingestion pipeline.
    *   **Responsibilities:** Receives video uploads, performs transcoding into adaptive bitrate formats (HLS), populates the database with metadata, and places the final video segments into object storage.
*   **PostgreSQL Database:** The persistent storage for all stateful information.
*   **Client Application Backend (BFF):** The customer's backend server. It orchestrates the user-facing experience, calling the Streambear Authorizer to get tokens before directing the video player to the Streambear Streaming Service.

#### 3.2. Data Flow & Workflows

**Workflow: Pay-per-second Live Stream**

This is the primary workflow demonstrating the core architecture.

1.  A user's video player, via its Backend-for-Frontend (BFF), requests access to a live stream.
2.  The BFF sends a request to the **Streambear Authorizer**'s `/authorize/live` endpoint. Ex: `{"streamID": "live-xyz", "segments": 10}`.
3.  The **Authorizer** performs a transactional operation against the database:
    a. Verifies the user has enough credits for 10 segments.
    b. Debits the required credits.
4.  If successful, the **Authorizer** generates and signs a **short-lived JWT** containing claims for that specific user and a specific range of segments (e.g., `first_sid: 120`, `last_sid: 129`).
5.  The Authorizer returns this JWT to the BFF.
6.  The BFF provides the video player with URLs for segments 120-129, each with the JWT appended as a query parameter (`?token=...`).
7.  The video player requests a segment URL from the **Streambear Streaming Service**.
8.  The **Streaming Service** receives the request:
    a. Extracts the JWT from the query parameter.
    b. Verifies the JWT's signature using a shared secret.
    c. Validates the claims (`exp`, `aud`, `vid`).
    d. Checks that the requested segment number falls within the `first_sid` and `last_sid` range.
9.  If all checks pass, the Streaming Service serves the video segment. **No database call is made.**
10. The process repeats as the client's BFF requests a token for the next batch of segments.

### **4. Security Model**

#### 4.1. Application-Level Security: JWTs

Authorization is handled via JSON Web Tokens (JWTs) signed with the **HS256** algorithm (HMAC with SHA-256) for maximum verification performance.

*   **Key Claims:**
    *   `iss` (Issuer): `streambear-authorizer`
    *   `aud` (Audience): `streambear-server`
    *   `sub` (Subject): The unique ID of the user.
    *   `exp` (Expiration): A short Unix timestamp (e.g., 60 seconds) for live stream tokens; longer for purchased VOD.
    *   `vid` (Video ID): The unique ID of the stream or VOD content.
    *   `access_type`: "full" (full VOD access) or "segment" (live stream access).
    *   `first_sid` (First Segment ID): The starting segment number this token is valid for.
    *   `last_sid` (Last Segment ID): The ending segment number this token is valid for.
*   **Secret Management:** The HS256 shared secret will be managed via Kubernetes Secrets and mounted into the `Authorizer` and `Streaming Service` pods as environment variables. Key rotation procedures must be established.

#### 4.2. Infrastructure-Level Security: mTLS

All service-to-service communication within the Kubernetes cluster is secured using Mutual TLS (mTLS).

*   **Implementation:** This will be enforced by a service mesh (e.g., Istio, Linkerd) deployed to the cluster.
*   **Mechanism:** The service mesh's control plane acts as an internal Certificate Authority (CA), automatically issuing short-lived certificates to each service pod. The sidecar proxy injected into each pod handles the mTLS handshake transparently, encrypting all traffic.
*   **Benefit:** This enforces a Zero Trust network policy. The `Authorizer` service will only accept connections from pods with a valid service identity (e.g., from the BFF or Uploader), preventing unauthorized internal access.

### **5. API Specification**

The system will expose two primary APIs, each defined by its own OpenAPI 3 specification.

*   **Tooling:** We will use `oapi-codegen` to generate Go server interfaces and models from the OpenAPI specs. This enforces the contract at compile time.
*   **Framework:** The HTTP services will be built using the `chi` router due to its lightweight nature and excellent `net/http` compatibility.

#### 5.1. `authorizer.yaml`

*   **`POST /authorize/vod`**: For purchased content. Returns a long-lived JWT for full access.
*   **`POST /authorize/live`**: For live streams. Debits credits and returns a short-lived JWT for a batch of segments.

#### 5.2. `server.yaml`

*   **`GET /stream/{videoId}/{segmentId}.ts`**: The primary endpoint for delivering video segments. Requires a valid JWT in the `token` query parameter.
*   **`GET /healthz`**: Standard health check endpoint.

### **6. Deployment & Operations (DevOps)**

#### 6.1. Containerization

A single, multi-stage `Dockerfile` will be used to build the lean `streambear` Go binary and place it in a minimal "distroless" base image for production.

#### 6.2. Kubernetes Deployment

*   **Structure:** Manifests will be structured using Kustomize with a `base` configuration and environment-specific `overlays` (`staging`, `production`).
*   **Roles:** Each role (`authorizer`, `server`, `uploader`) will be defined as a separate Kubernetes `Deployment`, but they will all use the **same container image**. The specific role is activated by passing the `--role` argument to the container's command.
*   **Scaling:** The stateless `server` deployment will be configured with a Horizontal Pod Autoscaler (HPA) to automatically scale based on CPU utilization.

#### 6.3. Multi-Cluster Architecture

For high availability and geo-locality, Streambear is designed to run in a multi-cluster configuration.
*   **Model:** A **Multi-Primary** service mesh configuration (e.g., Istio) will be used.
*   **Trust:** Each cluster's control plane will be configured with an intermediate CA certificate signed by a shared Root of Trust, enabling cross-cluster mTLS.
*   **Routing:** The service mesh will handle locality-aware routing and failover between clusters.

### **7. Future Considerations & Roadmap**

*   **DRM Integration:** Add support for Digital Rights Management (e.g., Widevine, FairPlay) by integrating with a DRM key server. The JWT can be extended to carry DRM-specific claims.
*   **DASH Protocol:** Add support for MPEG-DASH in addition to HLS.
*   **Observability:** Integrate structured logging, Prometheus metrics for all services, and distributed tracing (e.g., OpenTelemetry) to provide deep insights into system performance.
*   **Webhooks:** Implement a webhook system in the Authorizer to notify customer backends of events (e.g., low credit balance).

