package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteSuccessAndError(t *testing.T) {
	success := httptest.NewRecorder()
	WriteSuccess(success, map[string]string{"ok": "yes"})
	if success.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", success.Code)
	}
	var successBody Response
	if err := json.Unmarshal(success.Body.Bytes(), &successBody); err != nil {
		t.Fatalf("decode success: %v", err)
	}
	if successBody.Code != 200 || successBody.Message != "success" || successBody.Data == nil {
		t.Fatalf("unexpected success body: %#v", successBody)
	}

	failure := httptest.NewRecorder()
	WriteError(failure, http.StatusInternalServerError, "boom")
	if failure.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", failure.Code)
	}
	var failureBody Response
	if err := json.Unmarshal(failure.Body.Bytes(), &failureBody); err != nil {
		t.Fatalf("decode failure: %v", err)
	}
	if failureBody.Code != 500 || failureBody.Message != "boom" || failureBody.Data != nil {
		t.Fatalf("unexpected failure body: %#v", failureBody)
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := NewAuthHandler(AuthConfig{Enabled: true, Token: "secret"})
	protected := handler.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, tt := range []struct {
		name   string
		header string
		want   int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer bad", want: http.StatusUnauthorized},
		{name: "correct", header: "Bearer secret", want: http.StatusNoContent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d", tt.want, rec.Code)
			}
		})
	}
}
