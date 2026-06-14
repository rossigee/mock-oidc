package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rossigee/mock-oidc/internal/config"
	"github.com/rossigee/mock-oidc/internal/crypto"
	"github.com/rossigee/mock-oidc/internal/store"
)

func setupTestRouter() (*gin.Engine, *store.Store, *crypto.KeyManager) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Clients: []config.Client{
			{
				ID:           "test-client",
				Secret:       "test-secret",
				RedirectURIs: []string{"http://localhost:9999/callback"},
			},
		},
		Users: []config.User{
			{
				Sub:           "user-123",
				Username:      "testuser",
				Password:      "testpass",
				Name:          "Test User",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"devops"},
				IsAdmin:       false,
			},
			{
				Sub:           "admin-456",
				Username:      "admin",
				Password:      "adminpass",
				Name:          "Admin User",
				Email:         "admin@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
				IsAdmin:       true,
			},
		},
	}

	km, _ := crypto.NewKeyManager()
	s := store.NewStore(cfg, km, "http://localhost:8080")

	router := gin.New()
	router.LoadHTMLGlob("../../templates/*")

	authHandler := NewAuthHandler(s, km, "http://localhost:8080")
	userInfoHandler := NewUserInfoHandler(s)
	adminHandler := NewAdminHandler(s)

	router.GET("/authorize", authHandler.Authorize)
	router.POST("/authorize", authHandler.Authorize)
	router.POST("/token", authHandler.Token)
	router.POST("/revoke", authHandler.Revoke)
	router.GET("/userinfo", userInfoHandler.GetUserInfo)

	admin := router.Group("/admin", AdminAuthMiddleware("test-admin-key"))
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users", adminHandler.AddUser)
		admin.DELETE("/users/:sub", adminHandler.DeleteUser)
		admin.POST("/reset", adminHandler.Reset)
		admin.GET("/tokens", adminHandler.ListTokens)
	}

	return router, s, km
}

func TestAuthorizationCodeFlow(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Step 1: GET /authorize shows form
	req := httptest.NewRequest("GET", "/authorize?client_id=test-client&redirect_uri=http://localhost:9999/callback&response_type=code&state=xyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "username") {
		t.Fatal("form should contain username field")
	}

	// Step 2: POST /authorize with credentials
	formData := url.Values{
		"client_id":    {"test-client"},
		"redirect_uri": {"http://localhost:9999/callback"},
		"response_type": {"code"},
		"state":        {"xyz"},
		"username":     {"testuser"},
		"password":     {"testpass"},
	}
	req = httptest.NewRequest("POST", "/authorize", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("expected 302 redirect, got %d", w.Code)
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "code=") {
		t.Fatal("redirect should contain auth code")
	}
	authCode := extractParam(location, "code")

	// Step 3: POST /token to exchange code
	tokenData := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {"http://localhost:9999/callback"},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.AccessToken == "" || resp.IDToken == "" {
		t.Fatal("response should contain access_token and id_token")
	}
	if resp.RefreshToken == "" {
		t.Fatal("response should contain refresh_token")
	}

	// Step 4: GET /userinfo with access token
	req = httptest.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resp.AccessToken))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var userInfo map[string]interface{}
	json.NewDecoder(w.Body).Decode(&userInfo)
	if userInfo["sub"] != "user-123" {
		t.Fatalf("expected sub=user-123, got %v", userInfo["sub"])
	}
	// is_admin should not be present for non-admin users, or be false
	if isAdmin, ok := userInfo["is_admin"].(bool); ok && isAdmin {
		t.Fatal("is_admin should not be true for non-admin user")
	}
}

func TestPKCEFlow_S256(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Generate PKCE verifier and challenge
	verifier := "a" + strings.Repeat("b", 127) // 128 chars, valid length
	challenge := base64.RawURLEncoding.EncodeToString(hashSHA256([]byte(verifier)))

	// Step 1: POST /authorize with PKCE challenge
	formData := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:9999/callback"},
		"response_type":         {"code"},
		"state":                 {"xyz"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"username":              {"testuser"},
		"password":              {"testpass"},
	}
	req := httptest.NewRequest("POST", "/authorize", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	authCode := extractParam(w.Header().Get("Location"), "code")

	// Step 2: Exchange code with PKCE verifier
	tokenData := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {"http://localhost:9999/callback"},
		"code_verifier": {verifier},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPKCEFlow_Mismatch(t *testing.T) {
	router, _, _ := setupTestRouter()

	verifier := "a" + strings.Repeat("b", 127)
	challenge := base64.RawURLEncoding.EncodeToString(hashSHA256([]byte(verifier)))

	// Get auth code with valid PKCE
	formData := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:9999/callback"},
		"response_type":         {"code"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"username":              {"testuser"},
		"password":              {"testpass"},
	}
	req := httptest.NewRequest("POST", "/authorize", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	authCode := extractParam(w.Header().Get("Location"), "code")

	// Try to exchange with wrong verifier
	wrongVerifier := "c" + strings.Repeat("d", 127)
	tokenData := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {"http://localhost:9999/callback"},
		"code_verifier": {wrongVerifier},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == 200 {
		t.Fatal("expected error, but got 200")
	}
}

func TestPasswordGrant(t *testing.T) {
	router, _, _ := setupTestRouter()

	tokenData := url.Values{
		"grant_type":    {"password"},
		"username":      {"testuser"},
		"password":      {"testpass"},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.AccessToken == "" || resp.IDToken == "" || resp.RefreshToken == "" {
		t.Fatal("should return access_token, id_token, refresh_token")
	}
}

func TestRefreshTokenGrant(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Get initial tokens
	tokenData := url.Values{
		"grant_type":    {"password"},
		"username":      {"testuser"},
		"password":      {"testpass"},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	refreshToken := resp.RefreshToken

	// Use refresh token to get new tokens
	tokenData = url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var newResp TokenResponse
	json.NewDecoder(w.Body).Decode(&newResp)
	if newResp.AccessToken == "" {
		t.Fatal("should return new access token")
	}

	// Old refresh token should be single-use (consumed)
	tokenData = url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == 200 {
		t.Fatal("reusing same refresh token should fail (single-use)")
	}
}

func TestTokenRevocation(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Get token
	tokenData := url.Values{
		"grant_type":    {"password"},
		"username":      {"testuser"},
		"password":      {"testpass"},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	accessToken := resp.AccessToken

	// Use it at /userinfo (should work)
	req = httptest.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("token should be valid before revocation")
	}

	// Revoke it
	revokeData := url.Values{
		"token": {accessToken},
	}
	req = httptest.NewRequest("POST", "/revoke", strings.NewReader(revokeData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("revoke should return 200, got %d", w.Code)
	}

	// Try to use it again (should fail)
	req = httptest.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("revoked token should be rejected")
	}
}

func TestAdminAPI_AddUser(t *testing.T) {
	router, _, _ := setupTestRouter()

	newUser := config.User{
		Sub:           "test-new",
		Username:      "newuser",
		Password:      "newpass",
		Name:          "New User",
		Email:         "new@example.com",
		EmailVerified: true,
		Groups:        []string{"testers"},
		IsAdmin:       false,
	}
	body, _ := json.Marshal(newUser)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// New user should be able to authenticate
	tokenData := url.Values{
		"grant_type":    {"password"},
		"username":      {"newuser"},
		"password":      {"newpass"},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("new user should authenticate, got %d", w.Code)
	}
}

func TestAdminAPI_Reset(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Get a token
	tokenData := url.Values{
		"grant_type":    {"password"},
		"username":      {"testuser"},
		"password":      {"testpass"},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	token := resp.AccessToken

	// Token should work
	req = httptest.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("token should be valid before reset")
	}

	// Reset
	req = httptest.NewRequest("POST", "/admin/reset", nil)
	req.Header.Set("Authorization", "Bearer test-admin-key")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("reset should return 200, got %d", w.Code)
	}

	// Token should now be invalid
	req = httptest.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("reset should invalidate tokens")
	}
}

func TestBasicAuth_TokenEndpoint(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Get auth code first
	formData := url.Values{
		"client_id":    {"test-client"},
		"redirect_uri": {"http://localhost:9999/callback"},
		"response_type": {"code"},
		"username":     {"testuser"},
		"password":     {"testpass"},
	}
	req := httptest.NewRequest("POST", "/authorize", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	authCode := extractParam(w.Header().Get("Location"), "code")

	// Exchange with Basic auth instead of form credentials
	tokenData := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {authCode},
		"redirect_uri": {"http://localhost:9999/callback"},
	}
	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("test-client:test-secret")))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("basic auth should work, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminAPI_InvalidKey(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("invalid admin key should return 401, got %d", w.Code)
	}
}

// Helper functions

func extractParam(urlStr, param string) string {
	parsed, _ := url.Parse(urlStr)
	return parsed.Query().Get(param)
}

func hashSHA256(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
