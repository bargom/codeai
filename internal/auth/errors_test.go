package auth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorsAreSentinelValues(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidToken", ErrInvalidToken},
		{"ErrExpiredToken", ErrExpiredToken},
		{"ErrInvalidIssuer", ErrInvalidIssuer},
		{"ErrInvalidAudience", ErrInvalidAudience},
		{"ErrMissingToken", ErrMissingToken},
		{"ErrKeyNotFound", ErrKeyNotFound},
		{"ErrUnsupportedAlgorithm", ErrUnsupportedAlgorithm},
		{"ErrNoSecretConfigured", ErrNoSecretConfigured},
		{"ErrNoPublicKeyConfigured", ErrNoPublicKeyConfigured},
		{"ErrJWKSFetchFailed", ErrJWKSFetchFailed},
		{"ErrJWKSDecodeFailed", ErrJWKSDecodeFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestErrorsCanBeUsedWithErrorsIs(t *testing.T) {
	wrappedErr := errors.New("wrapped: " + ErrInvalidToken.Error())
	assert.NotErrorIs(t, wrappedErr, ErrInvalidToken)

	// Direct comparison works
	assert.ErrorIs(t, ErrInvalidToken, ErrInvalidToken)
}
