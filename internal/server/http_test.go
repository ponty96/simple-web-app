package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_NewHttp(t *testing.T) {
	cfg := &Config{
		Host: "localhost",
		Port: 4050,
	}
	server := NewHTTP(cfg)
	if server == nil {
		t.Error("Expected server not to be nil")
	}

	if server.Config != cfg {
		t.Error("Expected server config to match config passed")
	}
}

func Test_HealthCheck(t *testing.T) {
	cfg := &Config{
		Host: "localhost",
		Port: 4050,
	}

	s := NewHTTP(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s.healthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected Ok status code, got %d", w.Code)
	}
}
