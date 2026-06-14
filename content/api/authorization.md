---
title: Authorization API
---

OAuth 2.0 authorization endpoint for initiating the authorization code flow.

## Endpoint

| Method | Path | Description |
|--------|------|-------------|
| GET | `/authorize` | Initiate authorization code flow and user login |

## Authorization Request

```bash
GET /authorize?client_id=...&redirect_uri=...&response_type=code&scope=...&state=...
```

Initiates the OAuth 2.0 authorization code flow. The user is redirected to a login page.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | OAuth 2.0 client identifier |
| `redirect_uri` | Yes | Redirect URI for authorization response (must be registered) |
| `response_type` | Yes | Must be `code` |
| `scope` | Yes | Requested scopes (space-separated, e.g. `openid profile email`) |
| `state` | Yes | Opaque state for CSRF protection (returned in redirect) |
| `nonce` | No | Random value included in ID token for replay protection |
| `code_challenge` | No | PKCE code challenge (base64url(sha256(code_verifier))) |
| `code_challenge_method` | No | PKCE method: `S256` (recommended) or `plain` |

**Response:** Redirects to login page

User authenticates with username/password (default: `user` / `password`).

### Success Response

After authentication and consent, redirects to:

```
http://example.com/callback?code=AUTH_CODE&state=STATE_VALUE
```

| Parameter | Description |
|-----------|-------------|
| `code` | Authorization code (valid for 10 minutes) |
| `state` | Opaque state from request (for CSRF verification) |

### Error Response

On error, redirects to:

```
http://example.com/callback?error=error_code&error_description=...&state=STATE_VALUE
```

| Error | Description |
|-------|-------------|
| `invalid_request` | Missing or invalid parameter |
| `invalid_client` | Unknown or invalid client_id |
| `unsupported_response_type` | response_type not supported |
| `access_denied` | User denied authorization |

## Example

**Request:**

```
GET /authorize?
  client_id=test-client&
  redirect_uri=http://localhost:3000/callback&
  response_type=code&
  scope=openid+profile+email&
  state=abc123&
  nonce=xyz789
```

**Success Response:**

```
HTTP/302 Found
Location: http://localhost:3000/callback?code=AUTH_CODE&state=abc123
```

**Error Response:**

```
HTTP/302 Found
Location: http://localhost:3000/callback?error=access_denied&state=abc123
```

## PKCE Example

For public clients or mobile apps, use PKCE:

```bash
# Generate code verifier
code_verifier=$(openssl rand -hex 32)

# Generate code challenge
code_challenge=$(echo -n "$code_verifier" | sha256sum | base64 | tr '+/' '-_' | tr -d '=')

# Request authorization
GET /authorize?
  client_id=test-client&
  redirect_uri=http://localhost:3000/callback&
  response_type=code&
  scope=openid+profile&
  state=abc123&
  code_challenge=$code_challenge&
  code_challenge_method=S256
```

Then include `code_verifier` in the token request.

## See Also

- [Token](/api/token/) - Exchange authorization code for tokens
- [Discovery](/api/discovery/) - Provider metadata
