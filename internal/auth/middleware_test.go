package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserFromContext_NoUser(t *testing.T) {
	ctx := context.Background()
	user := UserFromContext(ctx)
	assert.Nil(t, user)
}

func TestUserFromContext_WithUser(t *testing.T) {
	expectedUser := &User{ID: "123", Email: "test@example.com"}
	ctx := ContextWithUser(context.Background(), expectedUser)

	user := UserFromContext(ctx)

	assert.Equal(t, expectedUser, user)
}

func TestContextWithUser(t *testing.T) {
	user := &User{ID: "456"}
	ctx := ContextWithUser(context.Background(), user)

	// Verify user is stored
	retrieved := ctx.Value(userContextKey)
	assert.NotNil(t, retrieved)
	assert.Equal(t, user, retrieved)
}

func TestExtractToken_Bearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-token")

	token := ExtractToken(req)

	assert.Equal(t, "my-token", token)
}

func TestExtractToken_QueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)

	token := ExtractToken(req)

	assert.Equal(t, "query-token", token)
}

func TestExtractToken_Cookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})

	token := ExtractToken(req)

	assert.Equal(t, "cookie-token", token)
}

func TestExtractToken_Priority(t *testing.T) {
	// Bearer header takes priority over query and cookie
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})

	token := ExtractToken(req)

	assert.Equal(t, "header-token", token)
}

func TestExtractToken_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token := ExtractToken(req)

	assert.Empty(t, token)
}

func TestExtractToken_NonBearerAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	token := ExtractToken(req)

	assert.Empty(t, token)
}

func TestMiddleware_Authenticate_Required_NoToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "authentication required", resp.Error)
}

func TestMiddleware_Authenticate_Required_ValidToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := generateHS256Token(t, claims, testSecret)

	handlerCalled := false
	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		user := UserFromContext(r.Context())
		assert.NotNil(t, user)
		assert.Equal(t, "user123", user.ID)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_Authenticate_Required_InvalidToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_Authenticate_Required_ExpiredToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "token has expired", resp.Error)
}

func TestMiddleware_Authenticate_Required_InvalidIssuer(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret, Issuer: "expected"})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "wrong",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "invalid token issuer", resp.Error)
}

func TestMiddleware_Authenticate_Required_InvalidAudience(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret, Audience: "expected"})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub": "user123",
		"aud": "wrong",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.Authenticate(AuthRequired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "invalid token audience", resp.Error)
}

func TestMiddleware_Authenticate_Optional_NoToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handlerCalled := false
	handler := m.Authenticate(AuthOptional)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		user := UserFromContext(r.Context())
		assert.Nil(t, user) // No user when no token
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_Authenticate_Optional_ValidToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := generateHS256Token(t, claims, testSecret)

	handlerCalled := false
	handler := m.Authenticate(AuthOptional)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		user := UserFromContext(r.Context())
		assert.NotNil(t, user)
		assert.Equal(t, "user123", user.ID)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_Authenticate_Optional_InvalidToken(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handlerCalled := false
	handler := m.Authenticate(AuthOptional)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		user := UserFromContext(r.Context())
		assert.Nil(t, user) // Invalid token results in no user
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_Authenticate_Public(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handlerCalled := false
	handler := m.Authenticate(AuthPublic)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_RequireAuth(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.RequireAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_OptionalAuth(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handlerCalled := false
	handler := m.OptionalAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_RequireRole_NoUser(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_RequireRole_HasRole(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []interface{}{"admin", "user"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handlerCalled := false
	handler := m.RequireAuth()(
		m.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_RequireRole_MissingRole(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []interface{}{"user"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.RequireAuth()(
		m.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMiddleware_RequirePermission_NoUser(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.RequirePermission("write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_RequirePermission_HasPermission(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret, PermsClaim: "permissions"})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":         "user123",
		"exp":         time.Now().Add(time.Hour).Unix(),
		"permissions": []interface{}{"read", "write"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handlerCalled := false
	handler := m.RequireAuth()(
		m.RequirePermission("write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_RequirePermission_MissingPermission(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret, PermsClaim: "permissions"})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":         "user123",
		"exp":         time.Now().Add(time.Hour).Unix(),
		"permissions": []interface{}{"read"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.RequireAuth()(
		m.RequirePermission("write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMiddleware_RequireAnyRole_NoUser(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	handler := m.RequireAnyRole("admin", "moderator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_RequireAnyRole_HasOneRole(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []interface{}{"moderator"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handlerCalled := false
	handler := m.RequireAuth()(
		m.RequireAnyRole("admin", "moderator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestMiddleware_RequireAnyRole_NoMatchingRole(t *testing.T) {
	v, _ := NewValidator(Config{Secret: testSecret})
	m := NewMiddleware(v)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []interface{}{"user"},
	}
	token := generateHS256Token(t, claims, testSecret)

	handler := m.RequireAuth()(
		m.RequireAnyRole("admin", "moderator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWriteJSONError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSONError(rec, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp errorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "bad request", resp.Error)
}
