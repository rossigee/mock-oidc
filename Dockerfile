FROM golang:1.26.4 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app ./cmd/main

FROM scratch
COPY --from=builder /build/app /app
COPY --from=builder /build/templates/ /templates/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT ["/app"]
