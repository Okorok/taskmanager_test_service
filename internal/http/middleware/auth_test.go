package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/http/middleware/mocks"
	"taskmanager/internal/infrastructure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_auth_missing_header(t *testing.T) {
	mw := NewAuthMiddleware(mocks.NewMockTokenParser(t))

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw.Handle(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func Test_auth_invalid_header_prefix(t *testing.T) {
	mw := NewAuthMiddleware(mocks.NewMockTokenParser(t))

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")

	mw.Handle(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func Test_auth_invalid_token(t *testing.T) {
	parser := mocks.NewMockTokenParser(t)
	parser.EXPECT().Parse(mock.Anything).Return(0, assert.AnError)

	mw := NewAuthMiddleware(parser)

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")

	mw.Handle(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func Test_auth_success_sets_user_id(t *testing.T) {
	parser := mocks.NewMockTokenParser(t)
	parser.EXPECT().Parse("good-token").Return(77, nil)

	mw := NewAuthMiddleware(parser)

	var seenUserID int64
	var seenOK bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenUserID, seenOK = infrastructure.UserIDFromCtx(r.Context())
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")

	mw.Handle(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, seenOK)
	assert.Equal(t, int64(77), seenUserID)
}
