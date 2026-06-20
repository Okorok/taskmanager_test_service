package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_password_hash_and_check(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("super-secret")
	require.NoError(t, err)
	assert.NotEqual(t, "super-secret", hash)

	assert.True(t, hasher.Check(hash, "super-secret"))
	assert.False(t, hasher.Check(hash, "wrong-password"))
}
