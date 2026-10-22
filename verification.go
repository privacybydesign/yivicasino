package main

import (
	"crypto/rsa"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	proofStatusValid = "VALID"
	jwtLeeway        = 60 * time.Second
)

// affirmativeAnswers are the lower-cased raw values of an over-18 attribute
// that mean the user is over 18.
var affirmativeAnswers = []string{"yes", "ja"}

type disclosedAttribute struct {
	ID       string `json:"id"`
	RawValue string `json:"rawvalue"`
}

// sessionResult holds the claims of an IRMA server result JWT that the age
// check relies on.
type sessionResult struct {
	jwt.RegisteredClaims
	ProofStatus string                 `json:"proofStatus"`
	Disclosed   [][]disclosedAttribute `json:"disclosed"`
}

type tokenVerifier struct {
	key    *rsa.PublicKey
	parser *jwt.Parser
}

func newVerifierFromFile(path string) (*tokenVerifier, error) {
	pemKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read IRMA server public key: %w", err)
	}

	return newTokenVerifier(pemKey)
}

func newTokenVerifier(pemKey []byte) (*tokenVerifier, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(pemKey)
	if err != nil {
		return nil, fmt.Errorf("parse IRMA server public key: %w", err)
	}

	// The algorithm and key come from our configuration, never from the token.
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithLeeway(jwtLeeway),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	return &tokenVerifier{key: key, parser: parser}, nil
}

// verify checks the signature and the time claims of a result JWT.
func (v *tokenVerifier) verify(token string) (*sessionResult, error) {
	result := &sessionResult{}

	_, err := v.parser.ParseWithClaims(token, result, func(*jwt.Token) (any, error) {
		return v.key, nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// isProofValid reports whether the IRMA server marked the disclosure as valid.
// A valid signature only proves the token came from our IRMA server, not that
// the disclosure succeeded.
func isProofValid(result *sessionResult) bool {
	return result.ProofStatus == proofStatusValid
}

// isAgeAllowed reports whether the first disclosed age attribute says the user
// is over 18.
func isAgeAllowed(disclosed [][]disclosedAttribute) bool {
	for _, conjunction := range disclosed {
		for _, attr := range conjunction {
			if !slices.Contains(ageAttributes, attr.ID) {
				continue
			}

			return slices.Contains(affirmativeAnswers, strings.ToLower(attr.RawValue))
		}
	}

	return false
}
