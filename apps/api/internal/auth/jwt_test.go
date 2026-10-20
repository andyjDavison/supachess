package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("test-secret")

func TestTokenRoundTrip(t *testing.T) {
	token, err := GenerateToken("user-123", secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	got, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if got != "user-123" {
		t.Errorf("subject = %q, want user-123", got)
	}
}

func TestParseTokenRejectsExpired(t *testing.T) {
	// Negative TTL puts exp in the past; the jwt library checks it for us.
	token, err := GenerateToken("user-123", secret, -time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	if _, err := ParseToken(token, secret); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expired token gave %v, want ErrInvalidToken", err)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, _ := GenerateToken("user-123", secret, time.Hour)

	if _, err := ParseToken(token, []byte("different-secret")); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("got %v, want ErrInvalidToken", err)
	}
}

func TestParseTokenRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "not-a-token", "a.b.c"} {
		if _, err := ParseToken(bad, secret); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("ParseToken(%q) = %v, want ErrInvalidToken", bad, err)
		}
	}
}

func TestParseTokenRejectsEmptySubject(t *testing.T) {
	// Signed correctly, but carries no subject - there is no user to trust.
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	if _, err := ParseToken(token, secret); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token with no subject gave %v, want ErrInvalidToken", err)
	}
}

// The classic JWT "alg confusion" attack: a token declaring alg=none must
// not be accepted just because it parses. This is what the
// SigningMethodHMAC check in ParseToken exists to stop.
func TestParseTokenRejectsUnsignedToken(t *testing.T) {
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "attacker",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing with alg=none: %v", err)
	}

	if _, err := ParseToken(token, secret); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("alg=none token gave %v, want ErrInvalidToken", err)
	}
}
