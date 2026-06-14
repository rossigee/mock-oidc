// Mock OIDC service for E2E testing.
package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rossigee/mock-oidc/internal/config"
	"github.com/rossigee/mock-oidc/internal/crypto"
	"github.com/rossigee/mock-oidc/internal/handler"
	"github.com/rossigee/mock-oidc/internal/middleware"
	"github.com/rossigee/mock-oidc/internal/store"
)

func main() {
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

	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}
	slog.Info("issuer URL", slog.String("issuer", issuer))

	keyMgr, err := crypto.NewKeyManager()
	if err != nil {
		slog.Error("failed to generate RSA keys", slog.Any("error", err))
		os.Exit(1)
	}

	dataStore := store.NewStore(cfg, keyMgr, issuer)

	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		adminKey = uuid.New().String()
		slog.Info("generated admin API key", slog.String("ADMIN_API_KEY", adminKey))
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "release"
	}
	gin.SetMode(ginMode)

	router := gin.New()
	router.LoadHTMLGlob("templates/*")
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogging())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())

	health := handler.NewHealthHandler()
	router.GET("/health", health.Health)
	router.GET("/ready", health.Ready)

	discoveryHandler := handler.NewOIDCDiscoveryHandler(issuer)
	router.GET("/.well-known/openid-configuration", discoveryHandler.Discovery)

	authHandler := handler.NewAuthHandler(dataStore, keyMgr, issuer)
	router.GET("/authorize", authHandler.Authorize)
	router.POST("/authorize", authHandler.Authorize)
	router.POST("/token", authHandler.Token)
	router.POST("/revoke", authHandler.Revoke)

	userInfoHandler := handler.NewUserInfoHandler(dataStore)
	router.GET("/userinfo", userInfoHandler.GetUserInfo)

	jwksHandler := handler.NewJWKSHandler(keyMgr)
	router.GET("/.well-known/jwks.json", jwksHandler.JWKS)

	adminHandler := handler.NewAdminHandler(dataStore)
	admin := router.Group("/admin", handler.AdminAuthMiddleware(adminKey))
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users", adminHandler.AddUser)
		admin.DELETE("/users/:sub", adminHandler.DeleteUser)
		admin.GET("/clients", adminHandler.ListClients)
		admin.POST("/clients", adminHandler.AddClient)
		admin.DELETE("/clients/:id", adminHandler.DeleteClient)
		admin.POST("/reset", adminHandler.Reset)
		admin.GET("/tokens", adminHandler.ListTokens)
	}

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
