---
title: UserInfo API
---

OpenID Connect userinfo endpoint for retrieving authenticated user profile information.

## Endpoint

| Method | Path | Description |
|--------|------|-------------|
| GET | `/userinfo` | Get authenticated user's profile |

## UserInfo Request

```bash
GET /userinfo
Authorization: Bearer ACCESS_TOKEN
```

Returns the authenticated user's profile information using the access token from the token endpoint.

**Headers:**

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Bearer token from `/token` endpoint |

**Response:** `200 OK`

```json
{
  "sub": "user-123",
  "email": "user@example.com",
  "email_verified": true,
  "name": "Test User",
  "given_name": "Test",
  "family_name": "User",
  "picture": "https://example.com/avatar.jpg",
  "preferred_username": "user"
}
```

| Claim | Type | Description |
|-------|------|-------------|
| `sub` | string | Subject (user ID) |
| `email` | string | Email address |
| `email_verified` | boolean | Whether email is verified |
| `name` | string | Full name |
| `given_name` | string | Given name |
| `family_name` | string | Family name |
| `picture` | string | Avatar URL |
| `preferred_username` | string | Username |

Additional custom claims can be returned based on configuration.

## Error Responses

**Response:** `401 Unauthorized`

```json
{
  "error": "invalid_token",
  "error_description": "The access token is invalid or expired"
}
```

| Error | HTTP Status | Description |
|-------|------------|-------------|
| `invalid_token` | 401 | Access token is missing, invalid, or expired |
| `insufficient_scope` | 403 | Token doesn't have required scope |
| `server_error` | 500 | Internal server error |

## Examples

### Get User Profile

**Request:**

```bash
curl -H "Authorization: Bearer eyJhbGc..." http://localhost:8080/userinfo
```

**Success Response:**

```json
{
  "sub": "user-123",
  "email": "user@example.com",
  "email_verified": true,
  "name": "Test User",
  "given_name": "Test",
  "family_name": "User"
}
```

### Invalid Token

**Request:**

```bash
curl -H "Authorization: Bearer invalid_token" http://localhost:8080/userinfo
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
const response = await fetch('http://localhost:8080/userinfo', {
  headers: {
    'Authorization': `Bearer ${accessToken}`
  }
});

if (response.ok) {
  const user = await response.json();
  console.log('User:', user);
} else {
  console.error('Failed to get user info');
}
```

## See Also

- [Token](/api/token/) - Get access token
- [Authorization](/api/authorization/) - Authorization code flow
