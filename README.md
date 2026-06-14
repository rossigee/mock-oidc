# Mock OIDC Server

A lightweight, self-contained OIDC (OpenID Connect) identity provider for testing applications that integrate with OIDC authentication systems. Useful for E2E testing, local development, and validation of OIDC integrations.

## Features

- **Full OAuth2/OIDC Code & Password Flows**: Authorization code flow, password grant, and refresh token rotation
- **PKCE Support**: Proof Key for Public Clients (S256 and plain methods) for secure public client flows
- **JWT Tokens**: RS256-signed ID tokens and access tokens with proper claims (sub, iss, aud, exp, iat, email, name, groups, is_admin, nonce)
- **Real Token Signing**: RS256-signed JWTs with proper cryptographic signing (stdlib crypto, no external JWT lib)
- **Token Revocation**: Revoke access and refresh tokens via `/revoke` endpoint
- **OIDC Discovery**: Standard `/.well-known/openid-configuration` with full metadata
- **JWKS**: JSON Web Key Set endpoint for public key distribution
- **Admin API**: Runtime fixture management — add/delete users and clients without restart
- **Minimal Dependencies**: Only Gin framework + stdlib crypto
- **Docker Ready**: Multi-stage build with minimal scratch image
- **YAML Configuration**: User and client configuration via simple YAML

## Quick Start

### Run Locally

```bash
# Build
go build -o mock-oidc ./cmd/main

# Run with default config
./mock-oidc

# Or with custom config
OIDC_CONFIG_FILE=./my-config.yaml ./mock-oidc

# Or with custom issuer and admin key
OIDC_ISSUER=http://my-oidc:8080 ADMIN_API_KEY=my-secret-key ./mock-oidc
```

Server runs on port 8080 by default (configurable via `HTTP_PORT` env var). The `ADMIN_API_KEY` is auto-generated and logged if not set.

### Run in Docker

```bash
# Build image
make docker-build

# Run container
docker run -d \
  -p 8080:8080 \
  -e OIDC_ISSUER=http://localhost:8080 \
  -v $(pwd)/config.yaml:/config.yaml \
  -e OIDC_CONFIG_FILE=/config.yaml \
  mock-oidc:latest
```

## Configuration

Users and clients are defined in `config.yaml`:

```yaml
clients:
  - id: harbor
    secret: harbor-secret
    redirect_uris:
      - https://harbor.example.com/c/oidc/callback
      - http://localhost:8888/c/oidc/callback

users:
  - sub: user-001
    username: test1
    password: password
    name: Test User
    email: test1@example.com
    email_verified: true
    groups: [devops]
    is_admin: false

  - sub: admin-001
    username: admin
    password: admin
    name: Admin User
    email: admin@example.com
    email_verified: true
    groups: [admins]
    is_admin: true
```

Environment variables:
- `OIDC_CONFIG_FILE`: Path to config.yaml (default: `./config.yaml`)
- `OIDC_ISSUER`: Issuer URL (default: `http://localhost:8080`)
- `HTTP_PORT`: Server port (default: `8080`)
- `LOG_LEVEL`: Logging level: debug, info, warn, error (default: `info`)
- `GIN_MODE`: Gin mode: debug, release (default: `release`)
- `ADMIN_API_KEY`: Admin API authorization token (default: auto-generated UUID)

## Endpoints

### Discovery & Keys
- `GET /.well-known/openid-configuration` — OIDC discovery document
- `GET /.well-known/jwks.json` — JSON Web Key Set with public RSA key

### Authorization & Token
- `GET /authorize` — Show login form (supports PKCE: `code_challenge`, `code_challenge_method`, `nonce`)
- `POST /authorize` — Authenticate with username/password, return auth code
- `POST /token` — Exchange auth code, password grant, or refresh token for tokens
- `POST /revoke` — Revoke access or refresh tokens

### User Info
- `GET /userinfo` — Return user claims (requires bearer token)

### Admin API (Protected by `Authorization: Bearer <ADMIN_API_KEY>`)
- `GET /admin/users` — List all users
- `POST /admin/users` — Add a new user
- `DELETE /admin/users/:sub` — Delete user by sub
- `GET /admin/clients` — List all clients
- `POST /admin/clients` — Add a new client
- `DELETE /admin/clients/:id` — Delete client by ID
- `POST /admin/reset` — Flush all tokens/codes (users/clients remain)
- `GET /admin/tokens` — List currently active token JTIs

### Health
- `GET /health` — Liveness probe
- `GET /ready` — Readiness probe

## Authorization Code Flow Example (with PKCE)

```bash
# 1. Generate PKCE challenge
CODE_VERIFIER=$(openssl rand -hex 32)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | openssl enc -base64 -A | tr '+/' '-_' | tr -d '=')

# 2. User logs in via authorize endpoint
curl -X POST http://localhost:8080/authorize \
  -d 'client_id=harbor' \
  -d 'redirect_uri=http://localhost:9999/callback' \
  -d 'response_type=code' \
  -d 'state=xyz123' \
  -d 'nonce=abc456' \
  -d "code_challenge=$CODE_CHALLENGE" \
  -d 'code_challenge_method=S256' \
  -d 'username=test1' \
  -d 'password=password'
# Returns: 302 redirect to http://localhost:9999/callback?code=<code>&state=xyz123

# 3. Exchange code for token (with PKCE verifier)
CODE=<auth-code-from-step-2>
curl -X POST http://localhost:8080/token \
  -d "grant_type=authorization_code" \
  -d "code=$CODE" \
  -d "client_id=harbor" \
  -d "client_secret=harbor-secret" \
  -d "redirect_uri=http://localhost:9999/callback" \
  -d "code_verifier=$CODE_VERIFIER"
# Returns: { access_token, token_type, expires_in, id_token, refresh_token }

# 4. Use access token to get user info
ACCESS_TOKEN=<token-from-step-3>
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/userinfo
# Returns: { sub, email, email_verified, name, groups, is_admin }

# 5. Refresh the access token
REFRESH_TOKEN=<refresh-token-from-step-3>
curl -X POST http://localhost:8080/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=$REFRESH_TOKEN" \
  -d "client_id=harbor" \
  -d "client_secret=harbor-secret"
# Returns new access_token + id_token

# 6. Revoke the token (optional cleanup)
curl -X POST http://localhost:8080/revoke \
  -d "token=$REFRESH_TOKEN" \
  -d "token_type_hint=refresh_token"
```

## Password Grant Flow Example

```bash
# Get token directly without authorization flow
curl -X POST http://localhost:8080/token \
  -d "grant_type=password" \
  -d "username=test1" \
  -d "password=password" \
  -d "client_id=harbor" \
  -d "client_secret=harbor-secret"
# Returns: { access_token, token_type, expires_in, id_token, refresh_token }
```

## Admin API Usage Example

```bash
export ADMIN_KEY="your-api-key-from-startup-log"

# List existing users
curl -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/admin/users

# Add a test user
curl -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:8080/admin/users \
  -d '{
    "sub": "test-scenario-1",
    "username": "scenario1",
    "password": "pass123",
    "name": "Test Scenario 1",
    "email": "scenario1@test.local",
    "email_verified": true,
    "groups": ["testers", "qa"],
    "is_admin": false
  }'

# Reset state between test runs
curl -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/admin/reset

# Delete a user
curl -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
  http://localhost:8080/admin/users/test-scenario-1
```

## JWT Claims

### ID Token Claims
- `sub` — Subject (user ID)
- `iss` — Issuer (server URL)
- `aud` — Audience (client ID)
- `exp` — Expiration time (1 hour from now)
- `iat` — Issued at time
- `nonce` — Nonce from auth request (if provided)
- `email` — User email
- `email_verified` — Email verification status (from config)
- `name` — User display name
- `groups` — Array of group names
- `is_admin` — Admin flag

### Access Token Claims
- `sub` — Subject (user ID)
- `iss` — Issuer
- `aud` — Audience (client ID)
- `exp` — Expiration (1 hour)
- `iat` — Issued at
- `jti` — Unique token ID (for revocation tracking)
- `groups` — Group membership
- `is_admin` — Admin flag

## Building & Testing

```bash
# Run tests
make test

# Run linter
make lint

# Build binary
make run

# Build Docker image
make docker-build

# Run from Docker
make docker-run

# Clean up
make clean
```

## Design Notes

- **No persistent state**: Auth codes and tokens are stored in-memory with TTL. Suitable for testing, not production.
- **JWT Access Tokens**: Access tokens are RS256-signed JWTs with claims, enabling local validation without /userinfo calls.
- **RSA key generation**: New RSA-2048 keypair generated at startup; no persistent key storage.
- **PKCE Support**: Both S256 (SHA256) and plain methods supported; verified on token exchange.
- **Token Revocation**: Revocation tracked via JTI; reset flushes all outstanding tokens.
- **Admin API**: Protected by bearer token key; enables test scenario setup without server restart.
- **Minimal external deps**: Only Gin for HTTP routing. JWT signing uses stdlib crypto.
- **Docker-friendly**: Single binary, minimal image size, supports volume mounts for config.

## Limitations

- **In-memory storage only**: Not suitable for production. No database or persistent state.
- **Simple credential validation**: Passwords stored plaintext in config (for testing).
- **No LDAP/AD integration**: Only YAML-based user storage.
- **No client assertions**: JWT bearer credentials not supported (only client_secret_basic/post).

## License

Part of the mock-servers project.
