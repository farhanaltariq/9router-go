package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"9router/proxy/internal/db"
	"9router/proxy/internal/dbtest"
)

func setupUsageChartTestDB(t *testing.T) (*db.Repo, func()) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_usage_chart_*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	sqlDB, err := db.OpenDatabase(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("OpenDatabase failed: %v", err)
	}

	if err := dbtest.CreateTables(sqlDB); err != nil {
		sqlDB.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("CreateTables failed: %v", err)
	}

	cleanup := func() {
		sqlDB.Close()
		os.Remove(tmpFile.Name())
	}

	return db.NewRepo(sqlDB), cleanup
}

func TestHandleUsageChart(t *testing.T) {
	repo, cleanup := setupUsageChartTestDB(t)
	defer cleanup()

	// Seed some usageHistory data
	now := time.Now().UTC()
	err := repo.InsertUsageHistory(
		"openai", "gpt-4o", "conn-1", "sk-test", "/v1/chat/completions",
		100, 50, 0.005, "ok", 150, "", `{"prompt":100,"completion":50}`,
	)
	if err != nil {
		t.Fatalf("InsertUsageHistory failed: %v", err)
	}

	// Seed daily usage for today
	todayKey := now.Format("2006-01-02")
	dailyPayload := `{"promptTokens":200,"completionTokens":100,"cost":0.01}`
	if err := repo.UpsertUsageDaily(todayKey, dailyPayload); err != nil {
		t.Fatalf("UpsertUsageDaily failed: %v", err)
	}

	tests := []struct {
		name        string
		period      string
		wantBuckets int
	}{
		{
			name:        "today period returns 24 buckets",
			period:      "today",
			wantBuckets: 24,
		},
		{
			name:        "24h period returns 24 buckets",
			period:      "24h",
			wantBuckets: 24,
		},
		{
			name:        "7d period returns 7 buckets",
			period:      "7d",
			wantBuckets: 7,
		},
		{
			name:        "30d period returns 30 buckets",
			period:      "30d",
			wantBuckets: 30,
		},
		{
			name:        "60d period returns 60 buckets",
			period:      "60d",
			wantBuckets: 60,
		},
		{
			name:        "default fallback returns 7 buckets",
			period:      "",
			wantBuckets: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/usage/chart"
			if tt.period != "" {
				url += "?period=" + tt.period
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			handler := HandleUsageChart(repo)
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
			}

			var points []ChartPoint
			if err := json.Unmarshal(rec.Body.Bytes(), &points); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if len(points) != tt.wantBuckets {
				t.Errorf("len(points) = %d, want %d", len(points), tt.wantBuckets)
			}

			// Verify that today's bucket has non-zero tokens
			if tt.period == "7d" || tt.period == "" {
				lastPoint := points[len(points)-1]
				if lastPoint.Tokens == 0 {
					t.Errorf("last point in 7d should have tokens, got 0")
				}
			}
		})
	}
}

func TestHandleUsageChart_FallbackToHistory(t *testing.T) {
	repo, cleanup := setupUsageChartTestDB(t)
	defer cleanup()

	// Only insert into usageHistory (no usageDaily)
	err := repo.InsertUsageHistory(
		"anthropic", "claude-3-5-sonnet", "conn-2", "sk-test", "/v1/chat/completions",
		500, 250, 0.02, "ok", 750, "", `{"prompt":500,"completion":250}`,
	)
	if err != nil {
		t.Fatalf("InsertUsageHistory failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/usage/chart?period=7d", nil)
	rec := httptest.NewRecorder()

	handler := HandleUsageChart(repo)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var points []ChartPoint
	if err := json.Unmarshal(rec.Body.Bytes(), &points); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	lastPoint := points[len(points)-1]
	if lastPoint.Tokens < 750 {
		t.Errorf("expected lastPoint.Tokens >= 750 from history fallback, got %d", lastPoint.Tokens)
	}
}
