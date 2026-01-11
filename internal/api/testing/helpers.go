// Package testing provides test utilities for the API package.
package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// TestServer wraps httptest.Server with additional testing utilities.
type TestServer struct {
	*httptest.Server
	t      *testing.T
	router chi.Router
}

// NewTestServer creates a new test server with the given router.
func NewTestServer(t *testing.T, router chi.Router) *TestServer {
	ts := httptest.NewServer(router)
	return &TestServer{
		Server: ts,
		t:      t,
		router: router,
	}
}

// Client returns an HTTP client configured for the test server.
func (ts *TestServer) Client() *http.Client {
	return ts.Server.Client()
}

// URL returns the base URL of the test server.
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// MakeRequest makes an HTTP request to the test server.
func (ts *TestServer) MakeRequest(method, path string, body interface{}) *http.Response {
	return MakeRequest(ts.t, ts.Client(), ts.URL(), method, path, body)
}

// MakeRequestWithContext makes an HTTP request with context to the test server.
func (ts *TestServer) MakeRequestWithContext(ctx context.Context, method, path string, body interface{}) *http.Response {
	return MakeRequestWithContext(ctx, ts.t, ts.Client(), ts.URL(), method, path, body)
}

// MakeRequest makes an HTTP request and returns the response.
func MakeRequest(t *testing.T, client *http.Client, baseURL, method, path string, body interface{}) *http.Response {
	return MakeRequestWithContext(context.Background(), t, client, baseURL, method, path, body)
}

// MakeRequestWithContext makes an HTTP request with context and returns the response.
func MakeRequestWithContext(ctx context.Context, t *testing.T, client *http.Client, baseURL, method, path string, body interface{}) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "failed to marshal request body")
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
	require.NoError(t, err, "failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err, "failed to execute request")

	return resp
}

// AssertStatus asserts that the response has the expected status code.
func AssertStatus(t *testing.T, resp *http.Response, expectedCode int) {
	t.Helper()
	require.Equal(t, expectedCode, resp.StatusCode, "unexpected status code")
}

// AssertJSON unmarshals the response body into the given value and asserts no error.
func AssertJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	err = json.Unmarshal(body, v)
	require.NoError(t, err, "failed to unmarshal response: %s", string(body))
}

// AssertJSONError asserts that the response is a JSON error with the expected message.
func AssertJSONError(t *testing.T, resp *http.Response, expectedMessage string) {
	t.Helper()
	var errResp ErrorResponse
	AssertJSON(t, resp, &errResp)
	require.Equal(t, expectedMessage, errResp.Error, "unexpected error message")
}

// AssertContentType asserts that the response has the expected content type.
func AssertContentType(t *testing.T, resp *http.Response, expectedType string) {
	t.Helper()
	contentType := resp.Header.Get("Content-Type")
	require.Contains(t, contentType, expectedType, "unexpected content type")
}

// ErrorResponse represents a standard error response for assertions.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// ReadBody reads and returns the response body as a string.
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	return string(body)
}

// DecodeJSON reads the response body and unmarshals it into v.
func DecodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	AssertJSON(t, resp, v)
}
