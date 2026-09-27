package media

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9router/proxy/internal/db"
)

func TestHandleSystemone_MissingModel(t *testing.T) {
	sqlDB, cleanup := setupEmbeddingsTestDB(t)
	defer cleanup()
	repo := db.NewRepo(sqlDB)
	handler := newTestMediaHandler(repo)

	for _, body := range []string{`{}`, `{"state":"x"}`, `not-json`} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader(body))
		handler.HandleSystemone(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: got status %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleSystemone_NoConnection(t *testing.T) {
	sqlDB, cleanup := setupEmbeddingsTestDB(t)
	defer cleanup()
	repo := db.NewRepo(sqlDB)
	handler := newTestMediaHandler(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/systemone",
		strings.NewReader(`{"model":"nosuchprovider/model-1","state":"x","questions":{}}`),
	)
	handler.HandleSystemone(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("got status %d, want 404", rec.Code)
	}
}
