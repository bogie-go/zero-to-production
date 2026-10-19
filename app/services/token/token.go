// Package token mints and verifies the bearer tokens a signed-in user
// presents. JWT, HS256, one shared secret: the Rails app next door can mint
// a token for its user with the jwt gem and this service verifies it, so
// users keep living in Rails, or this service mints its own from a sessions
// controller. Nothing here knows about HTTP.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalid is a token that does not verify: wrong signature, wrong
// issuer, malformed. ErrExpired is one that did, but too late.
var (
	ErrInvalid = errors.New("token: invalid")
	ErrExpired = errors.New("token: expired")
)

// Maker holds the secret and the issuer every token must carry. The secret
// is ecommerce.token_secret in the credentials file (ECOMMERCE_TOKEN_SECRET in the environment), at
// least 32 bytes; the issuer is the app's name, so a token minted for
// another service with the same secret is still refused.
type Maker struct {
	Secret []byte
	Issuer string
}

// Claims is what a verified token says.
type Claims struct {
	// Subject is who the token was minted for: the Rails user's id, usually.
	Subject   string
	ExpiresAt time.Time
	// ID is the token's own id, for a revocation list if you keep one.
	ID string
}

// Mint signs a token for subject that expires after ttl.
func (m Maker) Mint(subject string, ttl time.Duration) (string, error) {
	if subject == "" {
		return "", fmt.Errorf("%w: empty subject", ErrInvalid)
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    m.Issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		ID:        uuid.NewString(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.Secret)
}

// Verify checks the signature, the issuer and the expiry, and returns the
// claims. The method is pinned to HS256 so a token claiming "none" or an
// asymmetric algorithm is refused before the key is consulted.
func (m Maker) Verify(raw string) (Claims, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) { return m.Secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.Issuer),
		jwt.WithExpirationRequired(),
	)
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return Claims{}, ErrExpired
	case err != nil:
		return Claims{}, ErrInvalid
	case claims.Subject == "":
		return Claims{}, ErrInvalid
	}
	return Claims{Subject: claims.Subject, ExpiresAt: claims.ExpiresAt.Time, ID: claims.ID}, nil
}
