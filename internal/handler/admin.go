package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rossigee/mock-oidc/internal/config"
	"github.com/rossigee/mock-oidc/internal/store"
)

type AdminHandler struct {
	store *store.Store
}

func NewAdminHandler(s *store.Store) *AdminHandler {
	return &AdminHandler{store: s}
}

func AdminAuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth || token != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.ListUsers())
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	var user config.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if user.Sub == "" || user.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sub and username are required"})
		return
	}
	h.store.AddUser(user)
	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	sub := c.Param("sub")
	if !h.store.DeleteUser(sub) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListClients(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.ListClients())
}

func (h *AdminHandler) AddClient(c *gin.Context) {
	var client config.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if client.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	h.store.AddClient(client)
	c.JSON(http.StatusCreated, client)
}

func (h *AdminHandler) DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if !h.store.DeleteClient(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "client_not_found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) Reset(c *gin.Context) {
	h.store.Reset()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminHandler) ListTokens(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.ListActiveTokens())
}
