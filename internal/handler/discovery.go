package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type OIDCDiscoveryHandler struct {
	issuer string
}

func NewOIDCDiscoveryHandler(issuer string) *OIDCDiscoveryHandler {
	return &OIDCDiscoveryHandler{issuer: issuer}
}

func (h *OIDCDiscoveryHandler) Discovery(c *gin.Context) {
	issuer := h.issuer

	c.JSON(http.StatusOK, gin.H{
		"issuer":                 issuer,
		"authorization_endpoint": issuer + "/authorize",
		"token_endpoint":         issuer + "/token",
		"userinfo_endpoint":      issuer + "/userinfo",
		"jwks_uri":               issuer + "/.well-known/jwks.json",
		"revocation_endpoint":    issuer + "/revoke",

		"response_types_supported": []string{
			"code",
		},
		"grant_types_supported": []string{
			"authorization_code",
			"password",
			"refresh_token",
		},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"subject_types_supported":               []string{"public"},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
		"code_challenge_methods_supported": []string{"S256", "plain"},
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
			"nonce",
			"name",
			"email",
			"email_verified",
			"groups",
			"is_admin",
		},
	})
}
