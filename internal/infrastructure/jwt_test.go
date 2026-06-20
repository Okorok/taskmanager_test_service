package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_jwt_generate_and_parse_roundtrip(t *testing.T) {
	manager := NewJWTManager("secret", time.Hour)

	token, err := manager.Generate(123)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	userID, err := manager.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, int64(123), userID)
}

func Test_jwt_parse_rejects_garbage(t *testing.T) {
	manager := NewJWTManager("secret", time.Hour)

	_, err := manager.Parse("not-a-jwt")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func Test_jwt_parse_rejects_wrong_secret(t *testing.T) {
	issuer := NewJWTManager("secret-a", time.Hour)
	verifier := NewJWTManager("secret-b", time.Hour)

	token, err := issuer.Generate(1)
	require.NoError(t, err)

	_, err = verifier.Parse(token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func Test_jwt_parse_rejects_expired_token(t *testing.T) {
	manager := NewJWTManager("secret", -time.Hour)

	token, err := manager.Generate(1)
	require.NoError(t, err)

	_, err = manager.Parse(token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}
