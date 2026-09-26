package oauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9router/proxy/internal/db"
)

func TestHandleDeviceStart_bad(t *testing.T) {
	handler := NewOAuthHandler(nil)
	for _, payload := range []string{`{"provider":"google"}`, `{"provider":"nope"}`} {
		req := httptest.NewRequest("POST", "/api/oauth/device/start", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.HandleDeviceStart(rec, req)
		if rec.Code == http.StatusOK {
			t.Errorf("expected error for %s, got %s", payload, rec.Body.String())
		}
	}
}

func TestHandleDevicePoll_validation(t *testing.T) {
	handler := NewOAuthHandler(nil)
	for _, payload := range []string{`{}`, `{"provider":"github"}`, `{"provider":"nope","device_code":"x"}`} {
		req := httptest.NewRequest("POST", "/api/oauth/device/poll", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.HandleDevicePoll(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", payload, rec.Code)
		}
	}
}

func TestHandleDevicePoll_PendingDoesNotInsert(t *testing.T) {
	database, cleanup := setupOAuthTestDB(t)
	defer cleanup()
	repo := db.NewRepo(database)
	handler := NewOAuthHandler(repo)

	// Poll with unauthorized device code
	req := httptest.NewRequest("POST", "/api/oauth/device/poll", strings.NewReader(`{"provider":"github","device_code":"pending-code"}`))
	rec := httptest.NewRecorder()
	handler.HandleDevicePoll(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"status":"pending"`) && !strings.Contains(rec.Body.String(), `"status":"error"`) {
		t.Errorf("expected pending or error status, got: %s", rec.Body.String())
	}

	var count int
	err := repo.RawDB().QueryRow("SELECT COUNT(*) FROM providerConnections").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query providerConnections: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 connections inserted while pending, got %d", count)
	}
}
