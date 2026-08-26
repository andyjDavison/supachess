package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken covers every way a token can fail to be trusted: bad
// signature, expired, wrong signing method, missing subject. Handlers don't
// need to distinguish between these - they all mean "not logged in."
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims embeds the standard registered claims (exp, iat, sub) so
// expiration is checked automatically by the jwt library during parsing -
// we don't have to compare timestamps ourselves.
type Claims struct {
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT identifying userID, valid for ttl.
func GenerateToken(userID string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken verifies tokenString's signature and expiry, returning the
// user id stored in its subject claim if valid.
func ParseToken(tokenString string, secret []byte) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Reject tokens signed with anything other than HMAC. Without this
		// check, a token signed with "none" or an asymmetric algorithm the
		// server never intended to accept could otherwise be crafted to
		// pass verification - this is the classic JWT "alg confusion" attack.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	if claims.Subject == "" {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}