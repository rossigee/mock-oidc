---
title: API Reference
---

## OpenID Connect Discovery

Provides OpenID Connect provider metadata.

<span class="method method-get">GET</span> `/.well-known/openid-configuration`

Returns standard OpenID Connect discovery document with endpoints, supported grant types, claims, and algorithms.

**Response:**

```json
{
  "issuer": "http://localhost:8080",
  "authorization_endpoint": "http://localhost:8080/authorize",
  "token_endpoint": "http://localhost:8080/token",
  "userinfo_endpoint": "http://localhost:8080/userinfo",
  "jwks_uri": "http://localhost:8080/.well-known/jwks.json",
  "scopes_supported": ["openid", "profile", "email"],
  "grant_types_supported": ["authorization_code"],
  "response_types_supported": ["code"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["ES256"],
  "response_modes_supported": ["query", "fragment"]
}
```

## JWKS Endpoint

Provides public keys for token validation.

<span class="method method-get">GET</span> `/.well-known/jwks.json`

Returns JSON Web Key Set containing public keys for verifying signed tokens.

**Response:**

```json
{
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "x": "base64-encoded-x",
      "y": "base64-encoded-y",
      "use": "sig",
      "kid": "key-id-1"
    }
  ]
}
```

## Authorization Endpoint

Initiates the OAuth 2.0 authorization code flow.

<span class="method method-get">GET</span> `/authorize`

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | OAuth 2.0 client identifier |
| `redirect_uri` | Yes | Redirect URI for authorization response |
| `response_type` | Yes | Must be `code` |
| `scope` | Yes | Requested scopes (space-separated) |
| `state` | Yes | Opaque state for CSRF protection |
| `nonce` | No | Random value to include in ID token |
| `code_challenge` | No | PKCE code challenge (required for public clients) |
| `code_challenge_method` | No | PKCE method: `S256` or `plain` |

**Response:**

Redirects to `redirect_uri` with authorization code:

```
http://example.com/callback?
  code=AUTH_CODE&
  state=STATE_VALUE
```

## Token Endpoint

Exchanges authorization code for tokens.

<span class="method method-post">POST</span> `/token`

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | Must be `authorization_code` |
| `code` | Yes | Authorization code from authorization endpoint |
| `redirect_uri` | Yes | Must match original redirect_uri |
| `client_id` | Yes | OAuth 2.0 client identifier |
| `client_secret` | Yes | OAuth 2.0 client secret |
| `code_verifier` | No | PKCE code verifier |

**Response:**

```json
{
  "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "openid profile email"
}
```

## UserInfo Endpoint

Returns authenticated user's profile information.

<span class="method method-get">GET</span> `/userinfo`

**Headers:**

```
Authorization: Bearer ACCESS_TOKEN
```

**Response:**

```json
{
  "sub": "user-id-123",
  "email": "user@example.com",
  "email_verified": true,
  "name": "Test User",
  "given_name": "Test",
  "family_name": "User",
  "preferred_username": "user"
}
```

## Token Revocation

Revokes an issued token.

<span class="method method-post">POST</span> `/revoke`

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | Token to revoke |
| `client_id` | Yes | OAuth 2.0 client identifier |
| `client_secret` | Yes | OAuth 2.0 client secret |

**Response:**

Returns `200 OK` on success.

## Health Check

<span class="method method-get">GET</span> `/health`

Health check endpoint for container orchestration.

**Response:**

```json
{
  "status": "healthy"
}
```

## Metrics

<span class="method method-get">GET</span> `/metrics`

Prometheus-compatible metrics endpoint.

**Metrics:**

- `oidc_authorization_requests_total` - Total authorization requests
- `oidc_token_requests_total` - Total token requests
- `oidc_userinfo_requests_total` - Total userinfo requests
- `oidc_request_duration_seconds` - Request latency histogram
- `oidc_revoked_tokens_total` - Total revoked tokens

## Error Responses

All endpoints return standard OAuth 2.0 error responses:

```json
{
  "error": "invalid_request",
  "error_description": "The request is missing a required parameter"
}
```

**Common Errors:**

| Error | HTTP Status | Description |
|-------|------------|-------------|
| `invalid_request` | 400 | Missing or invalid parameter |
| `invalid_client` | 401 | Invalid client credentials |
| `access_denied` | 403 | User denied authorization |
| `unsupported_response_type` | 400 | Unsupported response type |
| `invalid_grant` | 400 | Invalid or expired grant |
| `server_error` | 500 | Internal server error |

## ID Token Claims

ID tokens include standard OpenID Connect claims:

| Claim | Type | Description |
|-------|------|-------------|
| `iss` | string | Issuer URL |
| `sub` | string | Subject (user ID) |
| `aud` | array | Audience (client ID) |
| `exp` | number | Expiration time |
| `iat` | number | Issued at time |
| `nonce` | string | Nonce from authorization request |
| `auth_time` | number | Authentication time |
| `email` | string | User email address |
| `name` | string | User's full name |
| `given_name` | string | Given name |
| `family_name` | string | Family name |
| `picture` | string | User picture URL |

Custom claims can be configured via the configuration file.
