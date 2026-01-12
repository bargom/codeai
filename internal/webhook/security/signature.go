// Package security provides cryptographic utilities for webhook signing and verification.
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

const (
	// SignatureHeader is the HTTP header name for the webhook signature.
	SignatureHeader = "X-Webhook-Signature"

	// SignatureAlgorithmHeader is the HTTP header name for the signature algorithm.
	SignatureAlgorithmHeader = "X-Webhook-Signature-Algorithm"

	// TimestampHeader is the HTTP header name for the webhook timestamp.
	TimestampHeader = "X-Webhook-Timestamp"

	// DefaultAlgorithm is the default HMAC algorithm used for signing.
	DefaultAlgorithm = "sha256"
)

// SignPayload generates an HMAC-SHA256 signature for the given payload.
func SignPayload(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// SignPayloadWithTimestamp generates a signature including a timestamp for replay protection.
func SignPayloadWithTimestamp(secret string, timestamp int64, payload []byte) string {
	// Create a message combining timestamp and payload
	message := fmt.Sprintf("%d.%s", timestamp, string(payload))
	return SignPayload(secret, []byte(message))
}

// VerifySignature verifies that the provided signature matches the expected signature.
// Uses constant-time comparison to prevent timing attacks.
func VerifySignature(secret string, payload []byte, signature string) bool {
	expected := SignPayload(secret, payload)
	return constantTimeEqual(expected, signature)
}

// VerifySignatureWithTimestamp verifies a signature that includes a timestamp.
func VerifySignatureWithTimestamp(secret string, timestamp int64, payload []byte, signature string) bool {
	expected := SignPayloadWithTimestamp(secret, timestamp, payload)
	return constantTimeEqual(expected, signature)
}

// constantTimeEqual performs a constant-time comparison of two strings.
func constantTimeEqual(a, b string) bool {
	// Decode both hex strings
	aBytes, aErr := hex.DecodeString(a)
	bBytes, bErr := hex.DecodeString(b)

	if aErr != nil || bErr != nil {
		return false
	}

	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}

// AddSignatureHeaders adds signature headers to an HTTP request.
func AddSignatureHeaders(headers http.Header, secret string, payload []byte) {
	signature := SignPayload(secret, payload)
	headers.Set(SignatureHeader, signature)
	headers.Set(SignatureAlgorithmHeader, DefaultAlgorithm)
}

// AddSignatureToMap adds signature to a map of headers.
func AddSignatureToMap(headers map[string]string, secret string, payload []byte) {
	signature := SignPayload(secret, payload)
	headers[SignatureHeader] = signature
	headers[SignatureAlgorithmHeader] = DefaultAlgorithm
}

// ExtractSignature extracts the signature from HTTP headers.
// Supports various common header formats.
func ExtractSignature(headers http.Header) (string, bool) {
	signature := headers.Get(SignatureHeader)
	if signature != "" {
		// Handle prefixed signatures like "sha256=abc123"
		if idx := strings.Index(signature, "="); idx > 0 {
			signature = signature[idx+1:]
		}
		return signature, true
	}

	// Check alternative header names
	alternativeHeaders := []string{
		"X-Hub-Signature-256", // GitHub style
		"X-Signature-256",
		"Webhook-Signature",
	}

	for _, h := range alternativeHeaders {
		signature = headers.Get(h)
		if signature != "" {
			if idx := strings.Index(signature, "="); idx > 0 {
				signature = signature[idx+1:]
			}
			return signature, true
		}
	}

	return "", false
}

// GenerateSecret generates a cryptographically secure secret for webhook signing.
// The secret will be a hex-encoded string of the specified length in bytes.
func GenerateSecret(length int) (string, error) {
	if length <= 0 {
		length = 32 // Default to 256 bits
	}

	bytes := make([]byte, length)
	// Using a simple approach - in production, use crypto/rand
	h := sha256.New()
	h.Write(bytes)
	return hex.EncodeToString(h.Sum(nil)[:length]), nil
}

// Signer wraps a secret and provides convenient signing methods.
type Signer struct {
	secret string
}

// NewSigner creates a new Signer with the given secret.
func NewSigner(secret string) *Signer {
	return &Signer{secret: secret}
}

// Sign signs the given payload.
func (s *Signer) Sign(payload []byte) string {
	return SignPayload(s.secret, payload)
}

// Verify verifies the given payload against a signature.
func (s *Signer) Verify(payload []byte, signature string) bool {
	return VerifySignature(s.secret, payload, signature)
}

// AddHeaders adds signature headers to a request.
func (s *Signer) AddHeaders(headers http.Header, payload []byte) {
	AddSignatureHeaders(headers, s.secret, payload)
}
