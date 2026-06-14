# Mock OIDC Server

A lightweight, self-contained OIDC (OpenID Connect) identity provider for testing applications that integrate with OIDC authentication systems. Useful for E2E testing, local development, and validation of OIDC integrations.

## Features

- **Full OAuth2/OIDC Code Flow**: Authorization code flow with authorization endpoint, token endpoint, userinfo endpoint
- **Real JWT Signing**: RS256-signed ID tokens with proper claims (sub, iss, aud, exp, iat, email, name, groups)
- **OIDC Discovery**: Standard `/.well-known/openid-configuration` endpoint with auto-discovery support
- **JWKS**: JSON Web Key Set endpoint for public key distribution
- **Minimal Dependencies**: Only uses Gin framework + stdlib crypto
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

# Or with custom issuer
OIDC_ISSUER=http://my-oidc:8080 ./mock-oidc
```

Server runs on port 8080 by default (configurable via `HTTP_PORT` env var).

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

## Endpoints

### Discovery
- `GET /.well-known/openid-configuration` — OIDC discovery document

### JWKS
- `GET /.well-known/jwks.json` — JSON Web Key Set with public RSA key

### Authorization
- `GET /authorize?client_id=...&redirect_uri=...&response_type=code&state=...` — Show login form
- `POST /authorize` with `username`, `password`, `client_id`, `redirect_uri`, `response_type`, `state` — Authenticate and return auth code

### Token
- `POST /token` with `grant_type=authorization_code`, `code`, `client_id`, `client_secret`, `redirect_uri` — Exchange code for tokens
- Also supports `grant_type=password` with `username` and `password`

### UserInfo
- `GET /userinfo` with `Authorization: Bearer <access_token>` — Return user claims

### Health
- `GET /health` — Liveness probe
- `GET /ready` — Readiness probe

## OAuth2 Code Flow Example

```bash
# 1. User logs in via authorize endpoint
curl -X POST http://localhost:8080/authorize \
  -d 'client_id=harbor' \
  -d 'redirect_uri=http://localhost:9999/callback' \
  -d 'response_type=code' \
  -d 'state=xyz123' \
  -d 'username=test1' \
  -d 'password=password'
# Returns: 302 redirect to http://localhost:9999/callback?code=<code>&state=xyz123

# 2. Exchange code for token
CODE=<auth-code-from-step-1>
curl -X POST http://localhost:8080/token \
  -d "grant_type=authorization_code" \
  -d "code=$CODE" \
  -d "client_id=harbor" \
  -d "client_secret=harbor-secret" \
  -d "redirect_uri=http://localhost:9999/callback"
# Returns: { access_token, token_type, expires_in, id_token }

# 3. Use access token to get user info
ACCESS_TOKEN=<token-from-step-2>
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/userinfo
# Returns: { sub, email, email_verified, name, groups }
```

## Testing with Harbor

To validate the blob association bugfix with Harbor using OIDC:

1. Start mock OIDC server (either locally or in Docker)
2. Configure Harbor to use the mock OIDC issuer:
   ```
   OIDC_NAME=MockOIDC
   OIDC_ENDPOINT=http://mock-oidc:8080
   OIDC_CLIENT_ID=harbor
   OIDC_CLIENT_SECRET=harbor-secret
   OIDC_AUTO_ONBOARD=false
   ```
3. Login to Harbor via browser using OIDC
4. Run `tests/validate-push-fix.sh` with OIDC user's CLI secret from Harbor API

## JWT Claims

ID tokens include:
- `sub` — Subject (user ID)
- `iss` — Issuer (server URL)
- `aud` — Audience (client ID)
- `exp` — Expiration time (1 hour from now)
- `iat` — Issued at time
- `email` — User email
- `email_verified` — Email verification status (always true)
- `name` — User display name
- `groups` — Array of group names

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
- **RSA key generation**: New RSA-2048 keypair generated at startup; no persistent key storage.
- **Minimal external deps**: Only Gin for HTTP routing. JWT signing uses stdlib crypto.
- **Docker-friendly**: Single binary, minimal image size, supports volume mounts for config.

## Limitations

- **In-memory storage only**: Not suitable for production. No database or persistent state.
- **Simple credential validation**: Passwords stored plaintext in config (for testing).
- **No refresh tokens**: Refresh token endpoint not implemented (placeholder only).
- **No end session endpoint**: RP-initiated logout not implemented.

## License

Part of the mock-servers project.
