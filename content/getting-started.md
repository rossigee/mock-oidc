---
title: Getting Started
---

## Installation

### Docker (Recommended)

Pull and run the latest image:

```bash
docker run -d \
  -p 8080:8080 \
  -e OIDC_ISSUER=http://localhost:8080 \
  ghcr.io/rossigee/mock-oidc
```

### Kubernetes

Deploy using the provided manifests:

```bash
kubectl apply -f k8s/
```

### From Source

```bash
git clone https://github.com/rossigee/mock-oidc.git
cd mock-oidc
make run
```

## Quick Test

Once running, verify the OIDC provider is responding:

```bash
curl http://localhost:8080/.well-known/openid-configuration
```

You should see the OpenID Connect Discovery metadata.

## Authorization Code Flow

### 1. Authorization Request

Redirect the user to the authorization endpoint:

```
http://localhost:8080/authorize?
  client_id=test-client&
  redirect_uri=http://localhost:3000/callback&
  response_type=code&
  scope=openid+profile+email&
  state=random-state-value
```

### 2. Login

The user logs in (default credentials: `user` / `password`)

### 3. Consent

The user consents to the requested scopes.

### 4. Authorization Code

The provider redirects to your redirect_uri with an authorization code:

```
http://localhost:3000/callback?
  code=AUTH_CODE&
  state=random-state-value
```

### 5. Token Exchange

Exchange the authorization code for tokens:

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&
      code=AUTH_CODE&
      client_id=test-client&
      client_secret=test-secret&
      redirect_uri=http://localhost:3000/callback"
```

Response:

```json
{
  "access_token": "eyJhbGc...",
  "id_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### 6. Validate Token

Verify and decode the ID token using the JWKS endpoint:

```bash
curl http://localhost:8080/.well-known/jwks.json
```

## Default Configuration

The mock OIDC provider comes with sensible defaults:

- **Issuer**: `http://localhost:8080`
- **Default User**: `user` / `password`
- **Access Token TTL**: 3600 seconds
- **ID Token TTL**: 3600 seconds
- **Scope**: `openid`, `profile`, `email`

See [Configuration](/configuration/) for customization options.

## Common Integration Patterns

### Go Client

```go
import "github.com/coreos/go-oidc/v3/oidc"

ctx := context.Background()
provider, err := oidc.NewProvider(ctx, "http://localhost:8080")
// Configure your OAuth2 config with the provider's endpoints
```

### Node.js Client

```javascript
const { Issuer } = require('openid-client');

const issuer = await Issuer.discover('http://localhost:8080');
const client = new issuer.Client({
  client_id: 'test-client',
  client_secret: 'test-secret'
});
```

### Python Client

```python
from authlib.integrations.requests_client import OAuth2Session

client = OAuth2Session(
    client_id='test-client',
    client_secret='test-secret',
    redirect_uri='http://localhost:3000/callback'
)
```

## Next Steps

- [Configure custom users and claims](/configuration/)
- [Explore all endpoints](/reference/)
- [View raw OpenAPI spec](/openapi.yaml)
