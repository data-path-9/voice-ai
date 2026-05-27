// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := c.GetHeader(types.PROJECT_SCOPE_KEY)
		if !validator.NotBlank(authToken) {
			authToken = c.Query(types.PROJECT_SCOPE_KEY)
			if !validator.NotBlank(authToken) {
				authToken = c.Param(types.PROJECT_SCOPE_KEY)
			}
		}
		if !validator.NotBlank(authToken) {
			c.Next()
			return
		}
		auth, err := resolver.Claim(c, authToken)
		if err != nil {
			c.Next()
			return
		}
		c.Set(string(types.CTX_), auth)
		c.Next()
	}
}
