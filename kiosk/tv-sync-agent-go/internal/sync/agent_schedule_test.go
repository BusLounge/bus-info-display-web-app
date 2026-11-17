package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tv-sync-agent-go/internal/models"
)

func TestSyncScheduleReplacesPreviousRowsWithCurrentEmptySchedule(t *testing.T) {
	const serviceDate = "2026-11-11"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/departures/lounge/lounge-1":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"loungeId":        "lounge-1",
				"serviceDate":     serviceDate,
				"serviceTimeZone": "Asia/Colombo",
				"departures":      []interface{}{},
			})
		case "/api/arrivals/lounge/lounge-1":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"loungeId":        "lounge-1",
				"serviceDate":     serviceDate,
				"serviceTimeZone": "Asia/Colombo",
				"arrivals":        []interface{}{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	storeDir := t.TempDir()
	snapshotPath := filepath.Join(storeDir, "schedule.json")
	previousSnapshot := []byte(`{
		"loungeId": "lounge-1",
		"departuresRaw": {"departures": [{"busNo": "stale"}]},
		"arrivalsRaw": {"arrivals": [{"busNo": "stale"}]}
	}`)
	if err := os.WriteFile(snapshotPath, previousSnapshot, 0o600); err != nil {
		t.Fatalf("write previous schedule snapshot: %v", err)
	}

	agent := New(models.Config{
		ServerURL:        server.URL,
		LoungeID:         "lounge-1",
		TVPurpose:        models.TVPurposeSchedule,
		StoreDir:         storeDir,
		RequestTimeoutMs: 1000,
	})
	if err := agent.SyncSchedule(context.Background()); err != nil {
		t.Fatalf("SyncSchedule() error = %v", err)
	}

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read schedule snapshot: %v", err)
	}
	var snapshot models.ScheduleSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode schedule snapshot: %v", err)
	}
	if snapshot.ServiceDate != serviceDate {
		t.Errorf("ServiceDate = %q, want %q", snapshot.ServiceDate, serviceDate)
	}
	if snapshot.ServiceTimeZone != "Asia/Colombo" {
		t.Errorf("ServiceTimeZone = %q, want %q", snapshot.ServiceTimeZone, "Asia/Colombo")
	}

	departures, ok := snapshot.DeparturesRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("DeparturesRaw type = %T, want map", snapshot.DeparturesRaw)
	}
	departureRows, ok := departures["departures"].([]interface{})
	if !ok {
		t.Fatalf("departures payload type = %T, want array", departures["departures"])
	}
	if got := len(departureRows); got != 0 {
		t.Errorf("departure count = %d, want 0", got)
	}
	arrivals, ok := snapshot.ArrivalsRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("ArrivalsRaw type = %T, want map", snapshot.ArrivalsRaw)
	}
	arrivalRows, ok := arrivals["arrivals"].([]interface{})
	if !ok {
		t.Fatalf("arrivals payload type = %T, want array", arrivals["arrivals"])
	}
	if got := len(arrivalRows); got != 0 {
		t.Errorf("arrival count = %d, want 0", got)
	}
}

func TestScheduleServiceDayRejectsMismatchedResponses(t *testing.T) {
	departures := map[string]interface{}{
		"serviceDate":     "2026-11-11",
		"serviceTimeZone": "Asia/Colombo",
	}
	arrivals := map[string]interface{}{
		"serviceDate":     "2026-11-10",
		"serviceTimeZone": "Asia/Colombo",
	}
	if _, _, err := scheduleServiceDay(departures, arrivals); err == nil {
		t.Fatal("scheduleServiceDay() accepted inconsistent service dates")
	}
}
