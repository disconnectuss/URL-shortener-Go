package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowBurst(t *testing.T) {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     10,
		burst:    3,
	}

	for i := range 3 {
		if !rl.allow("192.168.1.1") {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}

	if rl.allow("192.168.1.1") {
		t.Error("request beyond burst should be rejected")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     10,
		burst:    1,
	}

	if !rl.allow("10.0.0.1") {
		t.Error("first IP should be allowed")
	}

	if !rl.allow("10.0.0.2") {
		t.Error("second IP should be allowed independently")
	}
}

func TestRateLimitMiddleware_Returns429(t *testing.T) {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     10,
		burst:    1,
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimitMiddleware(rl)(dummyHandler)

	makeRequest := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if status := makeRequest(); status != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", status, http.StatusOK)
	}

	if status := makeRequest(); status != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", status, http.StatusTooManyRequests)
	}
}

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := loggingMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.Write([]byte("hello"))

	if rw.statusCode != http.StatusOK {
		t.Errorf("default status = %d, want %d", rw.statusCode, http.StatusOK)
	}
}

func TestResponseWriter_CustomStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", rw.statusCode, http.StatusCreated)
	}
}
