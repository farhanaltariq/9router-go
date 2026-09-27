package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"9router/proxy/internal/db"
)

// ChartPoint represents a single data point for the usage chart.
type ChartPoint struct {
	Label  string  `json:"label"`
	Tokens int64   `json:"tokens"`
	Cost   float64 `json:"cost"`
}

// HandleUsageChart returns bucketed usage stats for the chart: "today", "24h", "7d", "30d", "60d".
func HandleUsageChart(repo *db.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "7d"
		}

		w.Header().Set("Content-Type", "application/json")
		now := time.Now()

		switch period {
		case "today":
			const bucketCount = 24
			const bucketDur = time.Hour
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			startTime := startOfDay.Unix()

			points := make([]ChartPoint, bucketCount)
			for i := 0; i < bucketCount; i++ {
				t := startOfDay.Add(time.Duration(i) * bucketDur)
				points[i] = ChartPoint{
					Label: t.Format("15:04"),
				}
			}

			cutoffStr := startOfDay.UTC().Format(time.RFC3339)
			rows, err := repo.GetUsageHistorySince(cutoffStr)
			if err == nil {
				for _, row := range rows {
					t, err := time.Parse(time.RFC3339, row.Timestamp)
					if err != nil {
						continue
					}
					tLocal := t.In(now.Location())
					sec := tLocal.Unix()
					if sec < startTime {
						continue
					}
					idx := int((sec - startTime) / 3600)
					if idx >= 0 && idx < bucketCount {
						points[idx].Tokens += int64(row.PromptTokens + row.CompletionTokens)
						points[idx].Cost += row.Cost
					}
				}
			}
			_ = json.NewEncoder(w).Encode(points)
			return

		case "24h":
			const bucketCount = 24
			startTime := now.Add(-24 * time.Hour)
			startSec := startTime.Unix()

			points := make([]ChartPoint, bucketCount)
			for i := 0; i < bucketCount; i++ {
				t := startTime.Add(time.Duration(i) * time.Hour)
				points[i] = ChartPoint{
					Label: t.Format("15:04"),
				}
			}

			cutoffStr := startTime.UTC().Format(time.RFC3339)
			rows, err := repo.GetUsageHistorySince(cutoffStr)
			if err == nil {
				for _, row := range rows {
					t, err := time.Parse(time.RFC3339, row.Timestamp)
					if err != nil {
						continue
					}
					sec := t.Unix()
					if sec < startSec || sec > now.Unix() {
						continue
					}
					idx := int((sec - startSec) / 3600)
					if idx >= bucketCount {
						idx = bucketCount - 1
					}
					if idx >= 0 {
						points[idx].Tokens += int64(row.PromptTokens + row.CompletionTokens)
						points[idx].Cost += row.Cost
					}
				}
			}
			_ = json.NewEncoder(w).Encode(points)
			return

		default:
			// "7d", "30d", "60d"
			bucketCount := 7
			if period == "30d" {
				bucketCount = 30
			} else if period == "60d" {
				bucketCount = 60
			}

			startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
				AddDate(0, 0, -(bucketCount - 1))
			cutoffKey := startDate.Format("2006-01-02")

			dayRows, _ := repo.GetUsageDailyAfter(cutoffKey)
			dayMap := make(map[string]struct {
				tokens int64
				cost   float64
			})

			for _, dr := range dayRows {
				var data map[string]any
				if err := json.Unmarshal([]byte(dr.Data), &data); err == nil {
					var pTokens, cTokens int64
					var cost float64
					if v, ok := data["promptTokens"].(float64); ok {
						pTokens = int64(v)
					}
					if v, ok := data["completionTokens"].(float64); ok {
						cTokens = int64(v)
					}
					if v, ok := data["cost"].(float64); ok {
						cost = v
					}
					dayMap[dr.DateKey] = struct {
						tokens int64
						cost   float64
					}{
						tokens: pTokens + cTokens,
						cost:   cost,
					}
				}
			}

			// Complement with usageHistory for dates where usageDaily might be missing or under-reported
			cutoffStr := startDate.UTC().Format(time.RFC3339)
			if histRows, err := repo.GetUsageHistorySince(cutoffStr); err == nil {
				histMap := make(map[string]struct {
					tokens int64
					cost   float64
				})
				for _, row := range histRows {
					t, err := time.Parse(time.RFC3339, row.Timestamp)
					if err != nil {
						continue
					}
					dateKey := t.In(now.Location()).Format("2006-01-02")
					cur := histMap[dateKey]
					cur.tokens += int64(row.PromptTokens + row.CompletionTokens)
					cur.cost += row.Cost
					histMap[dateKey] = cur
				}
				for k, v := range histMap {
					existing, exists := dayMap[k]
					if !exists || existing.tokens < v.tokens {
						dayMap[k] = v
					}
				}
			}

			points := make([]ChartPoint, bucketCount)
			for i := 0; i < bucketCount; i++ {
				d := startDate.AddDate(0, 0, i)
				dateKey := d.Format("2006-01-02")
				val := dayMap[dateKey]
				points[i] = ChartPoint{
					Label:  fmt.Sprintf("%s %d", d.Format("Jan"), d.Day()),
					Tokens: val.tokens,
					Cost:   val.cost,
				}
			}

			_ = json.NewEncoder(w).Encode(points)
			return
		}
	}
}
