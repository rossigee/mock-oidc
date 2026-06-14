package handler

import (
	"encoding/base64"
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
	return &AuthHandler{store: s, keyMgr: km, issuer: issuer}
}

type AuthorizationRequest struct {
	ClientID            string `form:"client_id" binding:"required"`
	RedirectURI         string `form:"redirect_uri" binding:"required"`
	Scope               string `form:"scope"`
	State               string `form:"state"`
	ResponseType        string `form:"response_type" binding:"required"`
	Nonce               string `form:"nonce"`
	CodeChallenge       string `form:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method"`
	Username            string `form:"username"`
	Password            string `form:"password"`
}

type TokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	RedirectURI  string `form:"redirect_uri"`
	Username     string `form:"username"`
	Password     string `form:"password"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
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

	if err := c.ShouldBind(&req); err != nil {
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
	}

	client, ok := h.store.GetClient(req.ClientID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if !isRedirectURIAllowed(req.RedirectURI, client.RedirectURIs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		return
	}

	if req.Username != "" && req.Password != "" {
		user, ok := h.store.GetUserByUsername(req.Username)
		if !ok || user.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
			return
		}

		code := h.store.StoreAuthCode(user.Sub, req.ClientID, req.RedirectURI, req.Nonce, req.CodeChallenge, req.CodeChallengeMethod)
		params := url.Values{"code": {code}, "state": {req.State}}
		redirectURL, _ := url.Parse(req.RedirectURI)
		redirectURL.RawQuery = params.Encode()
		c.Redirect(http.StatusFound, redirectURL.String())
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"client_id":             req.ClientID,
		"redirect_uri":          req.RedirectURI,
		"state":                 req.State,
		"scope":                 req.Scope,
		"nonce":                 req.Nonce,
		"code_challenge":        req.CodeChallenge,
		"code_challenge_method": req.CodeChallengeMethod,
		"response_type":         req.ResponseType,
	})
}

func (h *AuthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Support client_secret_basic as fallback
	if req.ClientID == "" {
		if id, secret, ok := parseBasicAuth(c); ok {
			req.ClientID = id
			req.ClientSecret = secret
		}
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	switch req.GrantType {
	case "authorization_code", "password", "refresh_token":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	client, ok := h.store.GetClient(req.ClientID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if client.Secret != req.ClientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	var userSub, nonce string

	switch req.GrantType {
	case "authorization_code":
		var ok bool
		userSub, _, _, nonce, ok = h.store.ExchangeAuthCode(req.Code, req.CodeVerifier)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
			return
		}

	case "password":
		user, ok := h.store.GetUserByUsername(req.Username)
		if !ok || user.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant"})
			return
		}
		userSub = user.Sub

	case "refresh_token":
		var ok bool
		userSub, _, ok = h.store.ExchangeRefreshToken(req.RefreshToken)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
			return
		}
	}

	user, _ := h.store.GetUserBySub(userSub)
	if user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	idToken, err := h.keyMgr.SignIDToken(h.issuer, req.ClientID, user.Sub, nonce, user.Email, user.Name, user.EmailVerified, user.IsAdmin, user.Groups)
	if err != nil {
		slog.Error("failed to sign ID token", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	accessToken, err := h.store.IssueAccessToken(user.Sub, req.ClientID, user)
	if err != nil {
		slog.Error("failed to issue access token", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	refreshToken := h.store.StoreRefreshToken(user.Sub, req.ClientID)

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		IDToken:      idToken,
		RefreshToken: refreshToken,
	})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	hint := c.PostForm("token_type_hint")

	if hint == "refresh_token" {
		h.store.RevokeRefreshToken(token)
		c.Status(http.StatusOK)
		return
	}

	// Try access token (JWT), fall back to refresh token
	if err := h.store.RevokeAccessToken(token); err != nil {
		h.store.RevokeRefreshToken(token)
	}
	c.Status(http.StatusOK)
}

type UserInfoHandler struct {
	store *store.Store
}

func NewUserInfoHandler(s *store.Store) *UserInfoHandler {
	return &UserInfoHandler{store: s}
}

func (h *UserInfoHandler) GetUserInfo(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" || token == auth {
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
	if user.IsAdmin {
		response["is_admin"] = true
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

func isRedirectURIAllowed(uri string, allowed []string) bool {
	for _, a := range allowed {
		if a == uri {
			return true
		}
	}
	return false
}

func parseBasicAuth(c *gin.Context) (string, string, bool) {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
