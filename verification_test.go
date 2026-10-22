package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	return key
}

func publicKeyPEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

// attr builds a single disclosed attribute as it appears in a result JWT.
func attr(id, rawvalue string) disclosedAttribute {
	return disclosedAttribute{ID: id, RawValue: rawvalue}
}

func resultClaims(proofStatus string, disclosed [][]disclosedAttribute) jwt.MapClaims {
	return jwt.MapClaims{
		"iat":         time.Now().Unix(),
		"exp":         time.Now().Add(time.Hour).Unix(),
		"proofStatus": proofStatus,
		"disclosed":   disclosed,
	}
}

func validClaims() jwt.MapClaims {
	return resultClaims("VALID", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over18", "yes")}})
}

func signToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()

	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}

	return token
}

func TestIsProofValid(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"a valid proof status is accepted", "VALID", true},
		{"a non-valid proof status is rejected", "INVALID", false},
		{"an expired proof status is rejected", "EXPIRED", false},
		{"a missing proof status is rejected", "", false},
		{"proof status matching is case-sensitive", "valid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProofValid(&sessionResult{ProofStatus: tt.status}); got != tt.want {
				t.Errorf("isProofValid(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsAgeAllowed(t *testing.T) {
	tests := []struct {
		name      string
		disclosed [][]disclosedAttribute
		want      bool
	}{
		{"over-18 yes from a supported credential", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over18", "yes")}}, true},
		{"over-18 ja from a supported credential", [][]disclosedAttribute{{attr("pbdf.pbdf.passport.over18", "ja")}}, true},
		{"the raw value comparison is case-insensitive", [][]disclosedAttribute{{attr("irma-demo.gemeente.personalData.over18", "YES")}}, true},
		{"over-18 no is rejected", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over18", "no")}}, false},
		{"an unrelated attribute does not grant access", [][]disclosedAttribute{{attr("pbdf.pbdf.email.email", "x@example.com")}}, false},
		{"an over-21 attribute does not grant access", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over21", "yes")}}, false},
		{"an empty disclosure is rejected", nil, false},
		{"an empty conjunction is rejected", [][]disclosedAttribute{{}}, false},
		{"a missing raw value is rejected", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over18", "")}}, false},
		{"the first age attribute decides", [][]disclosedAttribute{{attr("pbdf.pbdf.idcard.over18", "no"), attr("pbdf.pbdf.passport.over18", "yes")}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAgeAllowed(tt.disclosed); got != tt.want {
				t.Errorf("isAgeAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgeAllowedEachCredential(t *testing.T) {
	for _, id := range ageAttributes {
		t.Run(id, func(t *testing.T) {
			if !isAgeAllowed([][]disclosedAttribute{{attr(id, "yes")}}) {
				t.Errorf("%s with value yes was not allowed", id)
			}
		})
	}
}

func TestVerifyAcceptsValidToken(t *testing.T) {
	key := newTestKey(t)
	verifier, err := newTokenVerifier(publicKeyPEM(t, key))
	if err != nil {
		t.Fatal(err)
	}

	result, err := verifier.verify(signToken(t, jwt.SigningMethodRS256, key, validClaims()))
	if err != nil {
		t.Fatalf("verify() = %v, want no error", err)
	}

	if !isProofValid(result) || !isAgeAllowed(result.Disclosed) {
		t.Errorf("decoded result = %+v, want a valid over-18 disclosure", result)
	}
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	key := newTestKey(t)
	otherKey := newTestKey(t)
	pemKey := publicKeyPEM(t, key)

	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()

	issuedInFuture := validClaims()
	issuedInFuture["iat"] = time.Now().Add(time.Hour).Unix()

	notYetValid := validClaims()
	notYetValid["nbf"] = time.Now().Add(time.Hour).Unix()

	withoutExpiry := validClaims()
	delete(withoutExpiry, "exp")

	tests := []struct {
		name  string
		token string
	}{
		{"signed with another key", signToken(t, jwt.SigningMethodRS256, otherKey, validClaims())},
		{"HS256 with the public key as secret", signToken(t, jwt.SigningMethodHS256, pemKey, validClaims())},
		{"RS384 with the right key", signToken(t, jwt.SigningMethodRS384, key, validClaims())},
		{"alg none", signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims())},
		{"expired", signToken(t, jwt.SigningMethodRS256, key, expired)},
		{"issued in the future", signToken(t, jwt.SigningMethodRS256, key, issuedInFuture)},
		{"not valid yet", signToken(t, jwt.SigningMethodRS256, key, notYetValid)},
		{"no expiry", signToken(t, jwt.SigningMethodRS256, key, withoutExpiry)},
		{"not a JWT", "not-a-jwt"},
		{"empty", ""},
	}

	verifier, err := newTokenVerifier(pemKey)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := verifier.verify(tt.token); err == nil {
				t.Error("verify() = nil error, want a rejection")
			}
		})
	}
}

func TestVerifyAllowsClockSkew(t *testing.T) {
	key := newTestKey(t)
	verifier, err := newTokenVerifier(publicKeyPEM(t, key))
	if err != nil {
		t.Fatal(err)
	}

	claims := validClaims()
	claims["exp"] = time.Now().Add(-30 * time.Second).Unix()

	if _, err := verifier.verify(signToken(t, jwt.SigningMethodRS256, key, claims)); err != nil {
		t.Errorf("verify() = %v, want a token expired within the leeway accepted", err)
	}
}

func TestVerifierRejectsBadKey(t *testing.T) {
	if _, err := newTokenVerifier([]byte("not a pem key")); err == nil {
		t.Error("newTokenVerifier() = nil error, want a rejection")
	}
}
