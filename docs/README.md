# Mock OIDC Service

Mock OpenID Connect (OIDC) provider for E2E testing of OAuth2/OIDC authentication flows.

## Endpoints

### Discovery
- `GET /.well-known/openid-configuration` - OpenID Connect discovery document

### Authorization
- `GET /authorize` - Authorization endpoint
  - Parameters: `client_id`, `redirect_uri`, `response_type`, `scope`, `state`, `nonce`
  
### Token
- `POST /token` - Token endpoint
  - Grant types: `authorization_code`, `password`

### User Info
- `GET /userinfo` - User information endpoint (requires Bearer token)

### JWKS
- `GET /.well-known/jwks.json` - JSON Web Key Set

### Health
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
