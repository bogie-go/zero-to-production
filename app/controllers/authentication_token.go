package controllers

import (
	"github.com/gin-gonic/gin"

	"ecommerce/app/middlewares"
	"ecommerce/app/services/token"
)

// Tokens is the token maker for this app: the secret from config
// (token.secret in the credentials file, TOKEN_SECRET in the environment) and the
// app's name as issuer. A sessions controller that signs users in here mints
// with it; a Rails app that signs them in mints the same token with the jwt
// gem and the same secret:
//
//	JWT.encode({ iss: "ecommerce", sub: user.id.to_s, iat: Time.now.to_i, exp: 15.minutes.from_now.to_i, jti: SecureRandom.uuid },
//	           Rails.application.credentials.ecommerce_token_secret, "HS256")
func (s *Server) Tokens() token.Maker {
	return token.Maker{Secret: []byte(s.Config.Token.Secret), Issuer: "ecommerce"}
}

// WithUser is the route group a signed-in user is required for. Mount a
// controller here instead of on the router, in routes.go:
//
//	s.Posts.SetupRoutes(s.WithUser(&r.RouterGroup))
//
// and in a handler, middlewares.Subject(c) is the token's subject.
func (s *Server) WithUser(r *gin.RouterGroup) *gin.RouterGroup {
	return r.Group("", middlewares.RequireUser(s.Tokens()))
}
