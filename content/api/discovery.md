---
title: Discovery API
---

OpenID Connect Discovery and JWKS endpoints for provider metadata and key discovery.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/.well-known/openid-configuration` | OpenID Connect Discovery metadata |
| GET | `/.well-known/jwks.json` | JSON Web Key Set for token validation |

## OpenID Connect Discovery

```bash
GET /.well-known/openid-configuration
```

Returns standard OpenID Connect discovery document with endpoints, supported grant types, claims, and algorithms.

**Response:** `200 OK`

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
  "response_modes_supported": ["query", "fragment"],
  "claim_types_supported": ["normal"],
  "claims_supported": ["sub", "email", "name", "given_name", "family_name"]
}
```

## JWKS Endpoint

```bash
GET /.well-known/jwks.json
```

Returns JSON Web Key Set containing public keys for verifying signed tokens.

**Response:** `200 OK`

```json
{
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "x": "base64-encoded-x-coordinate",
      "y": "base64-encoded-y-coordinate",
      "use": "sig",
      "alg": "ES256",
      "kid": "2024-01"
    }
  ]
}
```

The public key can be used to verify the signature of JWTs returned by the token and userinfo endpoints.

## See Also

- [Authorization](/api/authorization/) - OAuth 2.0 authorization code flow
- [Token](/api/token/) - Token endpoint for code exchange
