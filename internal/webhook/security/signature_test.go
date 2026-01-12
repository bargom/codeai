package security

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignPayload(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event": "test", "data": {"id": 123}}`)

	signature := SignPayload(secret, payload)

	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64) // SHA256 produces 32 bytes = 64 hex chars
}

func TestSignPayload_Deterministic(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event": "test"}`)

	sig1 := SignPayload(secret, payload)
	sig2 := SignPayload(secret, payload)

	assert.Equal(t, sig1, sig2)
}

func TestSignPayload_DifferentSecrets(t *testing.T) {
	payload := []byte(`{"event": "test"}`)

	sig1 := SignPayload("secret-1", payload)
	sig2 := SignPayload("secret-2", payload)

	assert.NotEqual(t, sig1, sig2)
}

func TestSignPayload_DifferentPayloads(t *testing.T) {
	secret := "my-secret"

	sig1 := SignPayload(secret, []byte(`{"a": 1}`))
	sig2 := SignPayload(secret, []byte(`{"a": 2}`))

	assert.NotEqual(t, sig1, sig2)
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event": "test"}`)

	signature := SignPayload(secret, payload)
	valid := VerifySignature(secret, payload, signature)

	assert.True(t, valid)
}

func TestVerifySignature_Invalid(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event": "test"}`)

	valid := VerifySignature(secret, payload, "invalid-signature")

	assert.False(t, valid)
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"event": "test"}`)
	signature := SignPayload("secret-1", payload)

	valid := VerifySignature("secret-2", payload, signature)

	assert.False(t, valid)
}

func TestVerifySignature_ModifiedPayload(t *testing.T) {
	secret := "my-secret-key"
	originalPayload := []byte(`{"event": "test"}`)
	modifiedPayload := []byte(`{"event": "modified"}`)

	signature := SignPayload(secret, originalPayload)
	valid := VerifySignature(secret, modifiedPayload, signature)

	assert.False(t, valid)
}

func TestSignPayloadWithTimestamp(t *testing.T) {
	secret := "my-secret"
	timestamp := int64(1704067200)
	payload := []byte(`{"event": "test"}`)

	signature := SignPayloadWithTimestamp(secret, timestamp, payload)

	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64)
}

func TestVerifySignatureWithTimestamp(t *testing.T) {
	secret := "my-secret"
	timestamp := int64(1704067200)
	payload := []byte(`{"event": "test"}`)

	signature := SignPayloadWithTimestamp(secret, timestamp, payload)
	valid := VerifySignatureWithTimestamp(secret, timestamp, payload, signature)

	assert.True(t, valid)
}

func TestVerifySignatureWithTimestamp_WrongTimestamp(t *testing.T) {
	secret := "my-secret"
	payload := []byte(`{"event": "test"}`)

	signature := SignPayloadWithTimestamp(secret, 1704067200, payload)
	valid := VerifySignatureWithTimestamp(secret, 1704067201, payload, signature)

	assert.False(t, valid)
}

func TestAddSignatureHeaders(t *testing.T) {
	headers := http.Header{}
	secret := "my-secret"
	payload := []byte(`{"event": "test"}`)

	AddSignatureHeaders(headers, secret, payload)

	assert.NotEmpty(t, headers.Get(SignatureHeader))
	assert.Equal(t, DefaultAlgorithm, headers.Get(SignatureAlgorithmHeader))
}

func TestAddSignatureToMap(t *testing.T) {
	headers := make(map[string]string)
	secret := "my-secret"
	payload := []byte(`{"event": "test"}`)

	AddSignatureToMap(headers, secret, payload)

	assert.NotEmpty(t, headers[SignatureHeader])
	assert.Equal(t, DefaultAlgorithm, headers[SignatureAlgorithmHeader])
}

func TestExtractSignature(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		expected string
		found    bool
	}{
		{
			name: "standard header",
			headers: http.Header{
				SignatureHeader: []string{"abc123"},
			},
			expected: "abc123",
			found:    true,
		},
		{
			name: "prefixed signature",
			headers: http.Header{
				SignatureHeader: []string{"sha256=abc123"},
			},
			expected: "abc123",
			found:    true,
		},
		{
			name: "github style",
			headers: http.Header{
				"X-Hub-Signature-256": []string{"sha256=github123"},
			},
			expected: "github123",
			found:    true,
		},
		{
			name:     "no signature",
			headers:  http.Header{},
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, found := ExtractSignature(tt.headers)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.expected, sig)
		})
	}
}

func TestSigner(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event": "test"}`)

	signer := NewSigner(secret)

	// Test signing
	signature := signer.Sign(payload)
	assert.NotEmpty(t, signature)

	// Test verification
	valid := signer.Verify(payload, signature)
	assert.True(t, valid)

	// Test invalid verification
	invalid := signer.Verify(payload, "wrong-signature")
	assert.False(t, invalid)
}

func TestSigner_AddHeaders(t *testing.T) {
	secret := "my-secret"
	payload := []byte(`{"event": "test"}`)
	headers := http.Header{}

	signer := NewSigner(secret)
	signer.AddHeaders(headers, payload)

	assert.NotEmpty(t, headers.Get(SignatureHeader))
	assert.Equal(t, DefaultAlgorithm, headers.Get(SignatureAlgorithmHeader))
}
