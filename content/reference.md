---
title: API Reference
---

## Overview

All OIDC endpoints return JSON responses. Standard OpenID Connect protocol with support for authorization code flow, token exchange, and userinfo queries.

## Base URL

```
http://localhost:8080
```

## Response Format

### Success

Standard successful responses include HTTP 200/201 status with JSON body containing the requested data.

### Error

All endpoints return standard OAuth 2.0 error responses:

```json
{
  "error": "error_code",
  "error_description": "Detailed error message"
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

## Endpoints

| Endpoint | Description |
|----------|-------------|
| [Discovery](/api/discovery/) | OpenID Connect metadata and JWKS |
| [Authorization](/api/authorization/) | Authorization code request and user login |
| [Token](/api/token/) | Token endpoint for code exchange |
| [UserInfo](/api/userinfo/) | User profile information |
| [Revocation](/api/revocation/) | Token revocation |
| [Health](/api/health/) | Service health check |

## Token Claims

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

Custom claims can be configured via the configuration file.
