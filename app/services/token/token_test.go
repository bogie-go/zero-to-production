package token

import (
	"errors"
	"testing"
	"time"
)

var maker = Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}

func TestMintThenVerify(t *testing.T) {
	raw, err := maker.Mint("user-42", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := maker.Verify(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-42" || claims.ID == "" || time.Until(claims.ExpiresAt) <= 0 {
		t.Errorf("claims = %+v", claims)
	}
}

func TestVerifyRefuses(t *testing.T) {
	expired, _ := maker.Mint("user-42", -time.Minute)
	otherSecret, _ := Maker{Secret: []byte("another-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}.Mint("user-42", time.Minute)
	otherIssuer, _ := Maker{Secret: maker.Secret, Issuer: "someone-else"}.Mint("user-42", time.Minute)

	for name, tc := range map[string]struct {
		raw  string
		want error
	}{
		"expired":        {expired, ErrExpired},
		"another secret": {otherSecret, ErrInvalid},
		"another issuer": {otherIssuer, ErrInvalid},
		"garbage":        {"not.a.token", ErrInvalid},
		"empty":          {"", ErrInvalid},
	} {
		if _, err := maker.Verify(tc.raw); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
	if _, err := maker.Mint("", time.Minute); err == nil {
		t.Error("minted a token with no subject")
	}
}
