# Mock OIDC Server - Quick Start

## 🚀 Get Running in 30 Seconds

### Local Development
```bash
cd /home/rossg/src/mock-servers/services/mock-oidc
go build -o mock-oidc ./cmd/main
./mock-oidc
# → Listens on http://localhost:8080
```

### Docker
```bash
docker pull ghcr.io/rossigee/mock-oidc:latest
docker run -d -p 8080:8080 \
  -e OIDC_ISSUER=http://localhost:8080 \
  ghcr.io/rossigee/mock-oidc:latest
```

### Test It
```bash
curl http://localhost:8080/.well-known/openid-configuration | jq .
curl http://localhost:8080/.well-known/jwks.json | jq .
```

## 📝 Test Credentials

| User | Password |
|------|----------|
| test1 | password |
| admin | admin |

## 🔗 Key Links

| What | Where |
|------|-------|
| Source Code | https://github.com/rossigee/mock-oidc |
| Builds | https://github.com/rossigee/mock-oidc/actions |
| Docker Image | ghcr.io/rossigee/mock-oidc:latest |
| Full Guide | /home/rossg/src/mock-servers/services/mock-oidc/README.md |
| Harbor Integration | /home/rossg/src/harbor/OIDC_TESTING_SETUP.md |

## 📚 Documentation

- **README.md** - Complete usage guide with all endpoints
- **IMPLEMENTATION_SUMMARY.md** - Architecture and design decisions  
- **QUICK_START.md** - This file

## 🧪 OAuth2 Code Flow Example

```bash
# 1. Get auth code
curl -X POST http://localhost:8080/authorize \
  -d 'client_id=harbor&redirect_uri=http://localhost:9999' \
  -d 'response_type=code&state=xyz' \
  -d 'username=test1&password=password' \
  -i | grep Location

# 2. Extract code and exchange for token
CODE=<code-from-above>
curl -X POST http://localhost:8080/token \
  -d "grant_type=authorization_code&code=$CODE" \
  -d 'client_id=harbor&client_secret=harbor-secret' \
  -d 'redirect_uri=http://localhost:9999' | jq .

# 3. Use access token to get user info
TOKEN=<access_token-from-above>
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/userinfo | jq .
```

## 🏗️ Configuration

Edit `config.yaml` to add users/clients:

```yaml
users:
  - sub: user-001
    username: myuser
    password: mypass
    email: user@example.com
    name: My User
    groups: [mygroup]

clients:
  - id: myapp
    secret: mysecret
    redirect_uris:
      - http://localhost:3000/callback
```

Run with:
```bash
OIDC_CONFIG_FILE=./config.yaml ./mock-oidc
```

## 🆘 Common Issues

### "Connection refused"
- Check if server is running: `curl http://localhost:8080/health`
- Check port isn't in use: `lsof -i :8080`

### "Invalid credentials"
- Check username/password in config.yaml
- Default: `test1` / `password`

### "Token invalid"
- Ensure you extracted correct access_token from response
- Bearer token should be exact UUID from token endpoint

## 📦 Deployment

### As Container in docker-compose
```yaml
services:
  mock-oidc:
    image: ghcr.io/rossigee/mock-oidc:latest
    environment:
      OIDC_ISSUER: http://mock-oidc:8080
    ports:
      - "5556:8080"
    networks:
      - harbor
```

### With Harbor
See: `/home/rossg/src/harbor/OIDC_TESTING_SETUP.md`

## 🔄 CI/CD

Code automatically builds and publishes to GHCR on:
- `git push origin master` → `ghcr.io/rossigee/mock-oidc:latest`
- `git tag v1.0.0 && git push origin v1.0.0` → version tags

Monitor: https://github.com/rossigee/mock-oidc/actions

## 🛠️ Development

```bash
# Run tests
make test

# Run linter
make lint

# Build binary
make run

# Build Docker image
make docker-build
```

## 📋 Environment Variables

```bash
OIDC_CONFIG_FILE=./config.yaml   # Config file path
OIDC_ISSUER=http://localhost:8080 # Issuer URL (must match browser)
HTTP_PORT=8080                    # Server port
LOG_LEVEL=info                    # debug, info, warn, error
GIN_MODE=release                  # debug, release
```

---

**Ready to test the Harbor blob association bugfix?** See `/home/rossg/src/harbor/OIDC_TESTING_SETUP.md`
