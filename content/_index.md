---
title: Home
---

[![Build](https://github.com/rossigee/mock-oidc/actions/workflows/build.yaml/badge.svg)](https://github.com/rossigee/mock-oidc/actions/workflows/build.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rossigee/mock-oidc)](https://github.com/rossigee/mock-oidc)
[![Docker Image Size](https://img.shields.io/docker/image-size/rossigee/mock-oidc)](https://github.com/rossigee/mock-oidc/pkgs/container/mock-oidc)
[![License](https://img.shields.io/github/license/rossigee/mock-oidc)](LICENSE)

A mock OIDC provider for E2E testing of OIDC-based applications without requiring a real identity provider. Provides full OpenID Connect protocol support with configurable users, scopes, and token claims.

Supports testing OIDC clients including web applications, native apps, and API services against a fully-functional OIDC provider.

## Overview

This service emulates a complete OIDC provider for testing purposes. It supports:

- **Authorization Code Flow**: Full OAuth 2.0 authorization code grant with PKCE
- **Token Endpoint**: OAuth 2.0 token exchange with JWT access and ID tokens
- **UserInfo Endpoint**: OpenID Connect userinfo with configurable claims
- **JWKS Endpoint**: JSON Web Key Set for token validation
- **Discovery**: OpenID Connect Discovery metadata endpoint
- **Revocation**: OAuth 2.0 token revocation support

All data is in-memory and reset on service restart.

## Quick Links

- [Getting Started](/getting-started/) - Run your first OAuth flow
- [Configuration](/configuration/) - Environment variables and settings
- [API Reference](/reference/) - All endpoints and claims
- [OpenAPI Spec](/openapi.yaml) - Raw OpenAPI/Swagger spec

## Use Cases

### CI/CD Testing

Run the mock service in your CI pipeline to test OIDC client code:

```yaml
# .github/workflows/test.yaml
- name: Run mock-oidc
  run: docker run -d -p 8080:8080 ghcr.io/rossigee/mock-oidc
- name: Run tests
  run: go test -v ./...
```

### Local Development

Develop OIDC client applications without requiring a real identity provider:

```bash
docker run -d -p 8080:8080 ghcr.io/rossigee/mock-oidc
# Your client code can now connect to localhost:8080
```

### Kubernetes

Deploy alongside your application in staging environments:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mock-oidc
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
```

## Features

- **Full OIDC Compliance**: Authorization code flow, token endpoint, userinfo
- **Configurable Users**: Define test users with custom claims
- **Configurable Scopes**: Support arbitrary OpenID and custom scopes
- **JWT Tokens**: Signed JWTs with configurable expiration
- **JWKS Endpoint**: Public key discovery for token validation
- **Metrics**: Prometheus-compatible metrics endpoint
- **Rate Limiting**: Configurable request rate limiting
- **CORS**: Configurable CORS for browser-based testing

## License

[MIT](LICENSE)
