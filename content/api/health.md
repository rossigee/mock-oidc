---
title: Health API
---

Service health check endpoint for monitoring and orchestration.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Service health status |
| GET | `/metrics` | Prometheus-compatible metrics |

## Health Check

```bash
GET /health
```

Returns the service health status.

**Response:** `200 OK`

```json
{
  "status": "healthy"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Always `healthy` if responding |

## Metrics

```bash
GET /metrics
```

Returns Prometheus-compatible metrics endpoint for monitoring.

**Response:** `200 OK` (text/plain)

```
# HELP oidc_authorization_requests_total Total authorization requests processed
# TYPE oidc_authorization_requests_total counter
oidc_authorization_requests_total 42

# HELP oidc_token_requests_total Total token requests processed
# TYPE oidc_token_requests_total counter
oidc_token_requests_total 38

# HELP oidc_userinfo_requests_total Total userinfo requests processed
# TYPE oidc_userinfo_requests_total counter
oidc_userinfo_requests_total 35

# HELP oidc_request_duration_seconds Request latency histogram
# TYPE oidc_request_duration_seconds histogram
oidc_request_duration_seconds_bucket{le="0.005"} 120
oidc_request_duration_seconds_bucket{le="0.01"} 125
oidc_request_duration_seconds_bucket{le="0.025"} 128

# HELP oidc_revoked_tokens_total Total revoked tokens
# TYPE oidc_revoked_tokens_total counter
oidc_revoked_tokens_total 3
```

### Metrics

- `oidc_authorization_requests_total` - Total authorization requests
- `oidc_token_requests_total` - Total token requests
- `oidc_userinfo_requests_total` - Total userinfo requests
- `oidc_request_duration_seconds` - Request latency histogram
- `oidc_revoked_tokens_total` - Total revoked tokens

## Kubernetes Usage

For liveness and readiness probes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mock-oidc
spec:
  template:
    spec:
      containers:
      - name: mock-oidc
        image: ghcr.io/rossigee/mock-oidc
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 2
          periodSeconds: 5
```

## Docker Compose Usage

```yaml
services:
  mock-oidc:
    image: ghcr.io/rossigee/mock-oidc
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
```

## See Also

- [Configuration](/configuration/) - Service configuration options
