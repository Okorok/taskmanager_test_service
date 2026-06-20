package auth_handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/auth_handler/mocks"
	"taskmanager/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func postJSON(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
}

func Test_register_returns_created(t *testing.T) {
	svc := mocks.NewMockRegistrar(t)
	svc.EXPECT().Register(mock.Anything, mock.Anything).Return(&entity.User{ID: 1}, nil)

	h := NewRegisterHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`{"email":"a@b.com","password":"secret123","name":"A"}`))

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func Test_register_invalid_json(t *testing.T) {
	h := NewRegisterHandler(mocks.NewMockRegistrar(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`not-json`))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_register_validation_errors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid_email", `{"email":"bad","password":"secret123","name":"A"}`},
		{"short_password", `{"email":"a@b.com","password":"123","name":"A"}`},
		{"empty_name", `{"email":"a@b.com","password":"secret123","name":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewRegisterHandler(mocks.NewMockRegistrar(t), slog.Default())

			rr := httptest.NewRecorder()
			h.Handle(rr, postJSON(tc.body))

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func Test_register_email_already_taken(t *testing.T) {
	svc := mocks.NewMockRegistrar(t)
	svc.EXPECT().Register(mock.Anything, mock.Anything).Return(nil, service.ErrEmailAlreadyTaken)

	h := NewRegisterHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`{"email":"a@b.com","password":"secret123","name":"A"}`))

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func Test_login_returns_token(t *testing.T) {
	svc := mocks.NewMockAuthenticator(t)
	svc.EXPECT().Login(mock.Anything, mock.Anything).Return("jwt", nil)

	h := NewLoginHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`{"email":"a@b.com","password":"secret123"}`))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "jwt")
}

func Test_login_invalid_json(t *testing.T) {
	h := NewLoginHandler(mocks.NewMockAuthenticator(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`not-json`))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_login_invalid_credentials(t *testing.T) {
	svc := mocks.NewMockAuthenticator(t)
	svc.EXPECT().Login(mock.Anything, mock.Anything).Return("", service.ErrInvalidCredentials)

	h := NewLoginHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, postJSON(`{"email":"a@b.com","password":"secret123"}`))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
