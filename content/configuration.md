---
title: Configuration
---

Configure mock-oidc via environment variables or config file.

## Environment Variables

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ISSUER` | `http://localhost:8080` | OIDC provider issuer URL |
| `OIDC_PORT` | `8080` | HTTP server listen port |
| `OIDC_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

### Token Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ACCESS_TOKEN_TTL` | `3600` | Access token lifetime in seconds |
| `OIDC_ID_TOKEN_TTL` | `3600` | ID token lifetime in seconds |
| `OIDC_REFRESH_TOKEN_TTL` | `86400` | Refresh token lifetime in seconds |

### Security Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_SIGNING_KEY` | Generated | ECDSA private key for signing tokens (PEM format) |
| `OIDC_CORS_ORIGINS` | `*` | CORS allowed origins (comma-separated) |
| `OIDC_RATE_LIMIT` | `100` | Requests per minute limit |

### Default User

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_DEFAULT_USERNAME` | `user` | Default test user username |
| `OIDC_DEFAULT_PASSWORD` | `password` | Default test user password |

## Configuration File

Use a YAML configuration file for more complex setups:

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/mock-oidc/config.yaml \
  -e OIDC_CONFIG=/etc/mock-oidc/config.yaml \
  ghcr.io/rossigee/mock-oidc
```

### Example config.yaml

```yaml
issuer: http://localhost:8080
port: 8080
logLevel: info

token:
  accessTokenTTL: 3600
  idTokenTTL: 3600
  refreshTokenTTL: 86400

users:
  - username: user
    password: password
    claims:
      email: user@example.com
      name: Test User
      given_name: Test
      family_name: User
  - username: admin
    password: admin123
    claims:
      email: admin@example.com
      name: Admin User
      given_name: Admin
      family_name: User

scopes:
  - openid
  - profile
  - email
  - custom:scope

cors:
  allowedOrigins:
    - http://localhost:3000
    - http://localhost:8080
  allowedMethods:
    - GET
    - POST
    - OPTIONS
  allowedHeaders:
    - Content-Type
    - Authorization
  exposedHeaders:
    - Content-Type
  maxAge: 3600
```

## Docker Compose Example

```yaml
version: '3.8'

services:
  mock-oidc:
    image: ghcr.io/rossigee/mock-oidc
    ports:
      - "8080:8080"
    environment:
      OIDC_ISSUER: http://localhost:8080
      OIDC_LOG_LEVEL: debug
      OIDC_DEFAULT_USERNAME: testuser
      OIDC_DEFAULT_PASSWORD: testpass
    volumes:
      - ./config.yaml:/etc/mock-oidc/config.yaml

  app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - mock-oidc
    environment:
      OIDC_ISSUER: http://mock-oidc:8080
      OIDC_CLIENT_ID: test-client
      OIDC_CLIENT_SECRET: test-secret
```

## Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mock-oidc-config
  namespace: default
data:
  config.yaml: |
    issuer: http://mock-oidc:8080
    port: 8080
    token:
      accessTokenTTL: 3600
      idTokenTTL: 3600
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mock-oidc
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mock-oidc
  template:
    metadata:
      labels:
        app: mock-oidc
    spec:
      containers:
      - name: mock-oidc
        image: ghcr.io/rossigee/mock-oidc
        ports:
        - containerPort: 8080
        env:
        - name: OIDC_CONFIG
          value: /etc/config/config.yaml
        volumeMounts:
        - name: config
          mountPath: /etc/config
      volumes:
      - name: config
        configMap:
          name: mock-oidc-config
```

## Testing Configuration

For CI/CD pipelines, use minimal configuration:

```bash
docker run -d \
  -p 8080:8080 \
  -e OIDC_ISSUER=http://localhost:8080 \
  -e OIDC_LOG_LEVEL=warn \
  ghcr.io/rossigee/mock-oidc
```

The provider automatically creates test clients and users for testing purposes.
