---
title: Revocation API
---

OAuth 2.0 token revocation endpoint for revoking tokens.

## Endpoint

| Method | Path | Description |
|--------|------|-------------|
| POST | `/revoke` | Revoke an access or refresh token |

## Revocation Request

```bash
POST /revoke
Content-Type: application/x-www-form-urlencoded

token=TOKEN&
  client_id=test-client&
  client_secret=test-secret
```

Revokes an issued token, preventing its further use.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | Token to revoke (access token or refresh token) |
| `client_id` | Yes | OAuth 2.0 client identifier |
| `client_secret` | Yes | OAuth 2.0 client secret |
| `token_type_hint` | No | Hint about token type: `access_token` or `refresh_token` |

**Response:** `200 OK`

Empty response body on success.

## Behavior

- **Success**: Token is revoked and no longer valid. Subsequent requests with this token are rejected with `invalid_token` error.
- **Invalid Token**: Returns `200 OK` regardless (as per RFC 7009)
- **Auth Failure**: Returns `401 Unauthorized` if client credentials are invalid

## Error Responses

**Response:** `401 Unauthorized`

```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}
```

| Error | HTTP Status | Description |
|-------|------------|-------------|
| `invalid_client` | 401 | Invalid client credentials |
| `unsupported_token_type` | 400 | Token type not supported |
| `server_error` | 500 | Internal server error |

## Examples

### Revoke Access Token

**Request:**

```bash
curl -X POST http://localhost:8080/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=eyJhbGc...&
      client_id=test-client&
      client_secret=test-secret&
      token_type_hint=access_token"
```

**Success Response:**

```
HTTP/200 OK

(empty body)
```

### Subsequent Request with Revoked Token

**Request:**

```bash
curl -H "Authorization: Bearer eyJhbGc..." http://localhost:8080/userinfo
```

**Error Response:**

```
HTTP/401 Unauthorized

{
  "error": "invalid_token"
}
```

## JavaScript Example

```javascript
async function revokeToken(token) {
  const response = await fetch('http://localhost:8080/revoke', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    },
    body: new URLSearchParams({
      token,
      client_id: 'test-client',
      client_secret: 'test-secret'
    })
  });
  
  if (response.ok) {
    console.log('Token revoked');
  } else {
    console.error('Failed to revoke token');
  }
}
```

## See Also

- [Token](/api/token/) - Get tokens
- [UserInfo](/api/userinfo/) - Access protected resources
