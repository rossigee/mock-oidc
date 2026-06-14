---
title: Token API
---

OAuth 2.0 token endpoint for exchanging authorization codes for tokens.

## Endpoint

| Method | Path | Description |
|--------|------|-------------|
| POST | `/token` | Exchange authorization code for tokens |

## Token Request

```bash
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
  code=AUTH_CODE&
  redirect_uri=http://example.com/callback&
  client_id=test-client&
  client_secret=test-secret
```

Exchanges an authorization code for access and ID tokens.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | Must be `authorization_code` |
| `code` | Yes | Authorization code from `/authorize` endpoint |
| `redirect_uri` | Yes | Must match original redirect_uri in authorization request |
| `client_id` | Yes | OAuth 2.0 client identifier |
| `client_secret` | Yes | OAuth 2.0 client secret |
| `code_verifier` | No | PKCE code verifier (required if PKCE was used) |

**Response:** `200 OK`

```json
{
  "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "openid profile email"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `access_token` | string | Bearer token for accessing protected resources |
| `token_type` | string | Token type (always `Bearer`) |
| `expires_in` | number | Token lifetime in seconds |
| `id_token` | string | JWT containing user identity claims |
| `scope` | string | Granted scopes (space-separated) |

### ID Token

The `id_token` is a JWT containing user identity information. Decode and verify:

```json
{
  "iss": "http://localhost:8080",
  "sub": "user-123",
  "aud": "test-client",
  "exp": 1718000000,
  "iat": 1717996400,
  "nonce": "xyz789",
  "auth_time": 1717996400,
  "email": "user@example.com",
  "email_verified": true,
  "name": "Test User",
  "given_name": "Test",
  "family_name": "User"
}
```

## Error Responses

**Response:** `400 Bad Request` or `401 Unauthorized`

```json
{
  "error": "invalid_grant",
  "error_description": "Authorization code has expired"
}
```

| Error | HTTP Status | Description |
|-------|------------|-------------|
| `invalid_request` | 400 | Missing or invalid parameter |
| `invalid_client` | 401 | Invalid client credentials |
| `invalid_grant` | 400 | Invalid or expired authorization code |
| `invalid_grant` | 400 | code_verifier doesn't match code_challenge |
| `unsupported_grant_type` | 400 | Unknown grant_type |
| `server_error` | 500 | Internal server error |

## Examples

### Authorization Code Flow

**Step 1: Get Authorization Code** (see [Authorization](/api/authorization/))

```
GET /authorize?client_id=test-client&redirect_uri=http://localhost:3000/callback&response_type=code&scope=openid+profile&state=abc123
```

**Step 2: Exchange for Tokens**

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&
      code=AUTH_CODE&
      client_id=test-client&
      client_secret=test-secret&
      redirect_uri=http://localhost:3000/callback"
```

**Response:**

```json
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "eyJhbGc...",
  "scope": "openid profile"
}
```

### PKCE Flow

For public clients:

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&
      code=AUTH_CODE&
      client_id=test-client&
      code_verifier=$code_verifier&
      redirect_uri=http://localhost:3000/callback"
```

Note: No `client_secret` needed when using PKCE.

## See Also

- [Authorization](/api/authorization/) - Get authorization code
- [UserInfo](/api/userinfo/) - Access user profile
- [Discovery](/api/discovery/) - Provider metadata
