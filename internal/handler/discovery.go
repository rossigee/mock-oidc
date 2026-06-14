package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OIDCDiscoveryHandler handles OIDC discovery endpoint
type OIDCDiscoveryHandler struct {
	issuer string
}

// NewOIDCDiscoveryHandler creates a new OIDC discovery handler
func NewOIDCDiscoveryHandler(issuer string) *OIDCDiscoveryHandler {
	return &OIDCDiscoveryHandler{issuer: issuer}
}

// Discovery returns the OpenID Connect discovery document
func (h *OIDCDiscoveryHandler) Discovery(c *gin.Context) {
	issuer := h.issuer

	config := gin.H{
		"issuer":                 issuer,
		"authorization_endpoint": issuer + "/authorize",
		"token_endpoint":         issuer + "/token",
		"userinfo_endpoint":      issuer + "/userinfo",
		"jwks_uri":               issuer + "/.well-known/jwks.json",
		"response_types_supported": []string{
			"code",
			"token",
			"id_token",
			"code token",
			"code id_token",
			"token id_token",
			"code token id_token",
		},
		"grant_types_supported": []string{
			"authorization_code",
			"implicit",
			"refresh_token",
		},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"subject_types_supported": []string{
			"public",
		},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
		"scopes_supported": []string{
			"openid",
			"profile",
			"email",
		},
		"claims_supported": []string{
			"sub",
			"iss",
			"aud",
			"exp",
			"iat",
			"name",
			"email",
			"email_verified",
		},
	}

	c.JSON(http.StatusOK, config)
}
