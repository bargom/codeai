// Package auth provides JWT authentication and authorization for CodeAI.
package auth

import "errors"

// Sentinel errors for JWT authentication.
var (
	// ErrInvalidToken indicates the token is malformed or has an invalid signature.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken indicates the token has expired.
	ErrExpiredToken = errors.New("token has expired")

	// ErrInvalidIssuer indicates the token issuer doesn't match the expected value.
	ErrInvalidIssuer = errors.New("invalid token issuer")

	// ErrInvalidAudience indicates the token audience doesn't match the expected value.
	ErrInvalidAudience = errors.New("invalid token audience")

	// ErrMissingToken indicates no authentication token was provided.
	ErrMissingToken = errors.New("missing authentication token")

	// ErrKeyNotFound indicates the signing key was not found (for JWKS lookups).
	ErrKeyNotFound = errors.New("signing key not found")

	// ErrUnsupportedAlgorithm indicates the token uses an unsupported signing algorithm.
	ErrUnsupportedAlgorithm = errors.New("unsupported signing algorithm")

	// ErrNoSecretConfigured indicates HS256 was requested but no secret is configured.
	ErrNoSecretConfigured = errors.New("no secret configured for symmetric algorithm")

	// ErrNoPublicKeyConfigured indicates RS256 was requested but no public key is available.
	ErrNoPublicKeyConfigured = errors.New("no public key configured for asymmetric algorithm")

	// ErrJWKSFetchFailed indicates failure to fetch keys from the JWKS endpoint.
	ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")

	// ErrJWKSDecodeFailed indicates failure to decode the JWKS response.
	ErrJWKSDecodeFailed = errors.New("failed to decode JWKS")
)
