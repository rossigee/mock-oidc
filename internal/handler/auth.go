package handler

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rossigee/mock-oidc/internal/crypto"
	"github.com/rossigee/mock-oidc/internal/store"
)

type AuthHandler struct {
	store  *store.Store
	keyMgr *crypto.KeyManager
	issuer string
}

func NewAuthHandler(s *store.Store, km *crypto.KeyManager, issuer string) *AuthHandler {
	return &AuthHandler{
		store:  s,
		keyMgr: km,
		issuer: issuer,
	}
}

type AuthorizationRequest struct {
	ClientID     string `form:"client_id" binding:"required"`
	RedirectURI  string `form:"redirect_uri" binding:"required"`
	Scope        string `form:"scope"`
	State        string `form:"state"`
	ResponseType string `form:"response_type" binding:"required"`
	Nonce        string `form:"nonce"`
	Username     string `form:"username"`
	Password     string `form:"password"`
}

type TokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code"`
	ClientID     string `form:"client_id" binding:"required"`
	ClientSecret string `form:"client_secret"`
	RedirectURI  string `form:"redirect_uri"`
	Username     string `form:"username"`
	Password     string `form:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (h *AuthHandler) Authorize(c *gin.Context) {
	var req AuthorizationRequest

	// Try to bind from form data first (POST), then fall back to query params (GET)
	if err := c.ShouldBind(&req); err != nil {
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
	}

	// If username/password provided, validate and generate code
	if req.Username != "" && req.Password != "" {
		user, ok := h.store.GetUserByUsername(req.Username)
		if !ok || user.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
			return
		}

		code := h.store.StoreAuthCode(user.Sub, req.ClientID, req.RedirectURI)
		redirectParams := url.Values{
			"code":  []string{code},
			"state": []string{req.State},
		}

		redirectURL, _ := url.Parse(req.RedirectURI)
		redirectURL.RawQuery = redirectParams.Encode()
		c.Redirect(http.StatusFound, redirectURL.String())
		return
	}

	// Otherwise, render login form
	c.HTML(http.StatusOK, "login.html", gin.H{
		"client_id":    req.ClientID,
		"redirect_uri": req.RedirectURI,
		"state":        req.State,
		"scope":        req.Scope,
	})
}

func (h *AuthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	if req.GrantType != "authorization_code" && req.GrantType != "password" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	// Validate client
	client, ok := h.store.GetClient(req.ClientID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if client.Secret != req.ClientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	var userSub string

	if req.GrantType == "authorization_code" {
		var ok bool
		userSub, _, _, ok = h.store.ExchangeAuthCode(req.Code)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_code"})
			return
		}
	} else if req.GrantType == "password" {
		user, ok := h.store.GetUserByUsername(req.Username)
		if !ok || user.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
			return
		}
		userSub = user.Sub
	}

	user, _ := h.store.GetUserBySub(userSub)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_user"})
		return
	}

	idToken, err := h.keyMgr.SignIDToken(h.issuer, req.ClientID, user.Sub, user.Email, user.Name, user.Groups)
	if err != nil {
		slog.Error("failed to sign ID token", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	accessToken := h.store.StoreAccessToken(user.Sub)

	response := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IDToken:     idToken,
	}

	c.JSON(http.StatusOK, response)
}

type UserInfoHandler struct {
	store *store.Store
}

func NewUserInfoHandler(s *store.Store) *UserInfoHandler {
	return &UserInfoHandler{store: s}
}

func (h *UserInfoHandler) GetUserInfo(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	userSub, ok := h.store.ValidateAccessToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	user, ok := h.store.GetUserBySub(userSub)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}

	response := gin.H{
		"sub":            user.Sub,
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"name":           user.Name,
	}

	if len(user.Groups) > 0 {
		response["groups"] = user.Groups
	}

	c.JSON(http.StatusOK, response)
}

type JWKSHandler struct {
	keyMgr *crypto.KeyManager
}

func NewJWKSHandler(km *crypto.KeyManager) *JWKSHandler {
	return &JWKSHandler{keyMgr: km}
}

func (h *JWKSHandler) JWKS(c *gin.Context) {
	c.JSON(http.StatusOK, h.keyMgr.JWKS())
}
