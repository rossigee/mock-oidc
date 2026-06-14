// Mock OIDC service for E2E testing.
package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rossigee/mock-oidc/internal/config"
	"github.com/rossigee/mock-oidc/internal/crypto"
	"github.com/rossigee/mock-oidc/internal/handler"
	"github.com/rossigee/mock-oidc/internal/middleware"
	"github.com/rossigee/mock-oidc/internal/store"
)

func main() {
	// Configure logging
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}),
	))

	// Load configuration
	configFile := os.Getenv("OIDC_CONFIG_FILE")
	if configFile == "" {
		configFile = "./config.yaml"
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("loaded configuration", slog.String("file", configFile))

	// Determine issuer URL
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}
	slog.Info("issuer URL", slog.String("issuer", issuer))

	// Generate RSA key pair
	keyMgr, err := crypto.NewKeyManager()
	if err != nil {
		slog.Error("failed to generate RSA keys", slog.Any("error", err))
		os.Exit(1)
	}

	// Create store
	dataStore := store.NewStore(cfg)

	// Configure Gin
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "release"
	}
	gin.SetMode(ginMode)

	// Create router
	router := gin.New()

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")

	// Apply middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogging())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())

	// Health check endpoints
	health := handler.NewHealthHandler()
	router.GET("/health", health.Health)
	router.GET("/ready", health.Ready)

	// OIDC discovery endpoint
	discoveryHandler := handler.NewOIDCDiscoveryHandler(issuer)
	router.GET("/.well-known/openid-configuration", discoveryHandler.Discovery)

	// OIDC endpoints
	authHandler := handler.NewAuthHandler(dataStore, keyMgr, issuer)
	router.GET("/authorize", authHandler.Authorize)
	router.POST("/authorize", authHandler.Authorize)
	router.POST("/token", authHandler.Token)

	// UserInfo endpoint
	userInfoHandler := handler.NewUserInfoHandler(dataStore)
	router.GET("/userinfo", userInfoHandler.GetUserInfo)

	// JWKS endpoint
	jwksHandler := handler.NewJWKSHandler(keyMgr)
	router.GET("/.well-known/jwks.json", jwksHandler.JWKS)

	// Start server
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("starting mock-oidc", slog.String("port", port))
	if err := router.Run(":" + port); err != nil {
		slog.Error("failed to start server", slog.Any("error", err))
		os.Exit(1)
	}
}
