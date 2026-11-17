package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"tv-sync-agent-go/internal/models"
)

func TestSyncLoungeAdsCachesRemoteMediaAndReusesIt(t *testing.T) {
	var downloadCount atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/lounge-ads/lounge/lounge-1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.LoungeAd{{
				ID:                "lounge-ad-1",
				AdvertisementName: "Staging lounge ad",
				MediaURL:          server.URL + "/media/lounge-ad.mp4",
				MediaType:         "video",
				IsActive:          true,
			}})
		case "/media/lounge-ad.mp4":
			downloadCount.Add(1)
			_, _ = w.Write([]byte("video bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	storeDir := t.TempDir()
	agent := New(models.Config{
		ServerURL:         server.URL,
		LoungeID:          "lounge-1",
		StoreDir:          storeDir,
		RequestTimeoutMs:  1000,
		DownloadTimeoutMs: 1000,
	})

	if err := agent.SyncLoungeAds(context.Background()); err != nil {
		t.Fatalf("first SyncLoungeAds() error = %v", err)
	}

	snapshotPath := filepath.Join(storeDir, "lounge-ads.json")
	readSnapshot := func() models.LoungeAdsSnapshot {
		t.Helper()
		data, err := os.ReadFile(snapshotPath)
		if err != nil {
			t.Fatalf("read lounge snapshot: %v", err)
		}
		var snapshot models.LoungeAdsSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			t.Fatalf("decode lounge snapshot: %v", err)
		}
		return snapshot
	}

	snapshot := readSnapshot()
	if len(snapshot.Items) != 1 {
		t.Fatalf("snapshot item count = %d, want 1", len(snapshot.Items))
	}
	item := snapshot.Items[0]
	if item.LocalFile != "lounge-ad-1.mp4" {
		t.Errorf("LocalFile = %q, want %q", item.LocalFile, "lounge-ad-1.mp4")
	}
	if item.MediaURL != server.URL+"/media/lounge-ad.mp4" {
		t.Errorf("MediaURL = %q, want original remote URL", item.MediaURL)
	}

	mediaPath := filepath.Join(storeDir, "media", item.LocalFile)
	media, err := os.ReadFile(mediaPath)
	if err != nil {
		t.Fatalf("read cached media: %v", err)
	}
	if string(media) != "video bytes" {
		t.Errorf("cached media = %q, want %q", media, "video bytes")
	}

	if err := agent.SyncLoungeAds(context.Background()); err != nil {
		t.Fatalf("second SyncLoungeAds() error = %v", err)
	}
	if got := downloadCount.Load(); got != 1 {
		t.Errorf("media download count = %d, want 1", got)
	}
	if _, err := os.Stat(mediaPath + ".part"); !os.IsNotExist(err) {
		t.Errorf("temporary download file still exists (stat error = %v)", err)
	}
}

func TestSyncLoungeAdsRemovesPartialMediaOnDownloadFailure(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/lounge-ads/lounge/lounge-1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.LoungeAd{{
				ID:                "lounge-ad-1",
				AdvertisementName: "Staging lounge ad",
				MediaURL:          server.URL + "/media/lounge-ad.mp4",
				MediaType:         "video",
				IsActive:          true,
			}})
		case "/media/lounge-ad.mp4":
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("partial"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	storeDir := t.TempDir()
	agent := New(models.Config{
		ServerURL:         server.URL,
		LoungeID:          "lounge-1",
		StoreDir:          storeDir,
		RequestTimeoutMs:  1000,
		DownloadTimeoutMs: 1000,
	})

	if err := agent.SyncLoungeAds(context.Background()); err == nil {
		t.Fatal("SyncLoungeAds() error = nil, want interrupted download error")
	}

	mediaDir := filepath.Join(storeDir, "media")
	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		t.Fatalf("read media directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("media directory contains %d files after failed download, want none", len(entries))
	}
	if _, err := os.Stat(filepath.Join(storeDir, "lounge-ads.json")); !os.IsNotExist(err) {
		t.Errorf("lounge snapshot was finalized after failed download (stat error = %v)", err)
	}
}

func TestSyncAdsCleanupPreservesActiveLoungeMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tv/ads/lounge-group" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	storeDir := t.TempDir()
	mediaDir := filepath.Join(storeDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("create media directory: %v", err)
	}

	for _, name := range []string{
		"active-lounge.mp4",
		"legacy-lounge.webm",
		"inactive-lounge.mp4",
		"orphan.mp4",
	} {
		if err := os.WriteFile(filepath.Join(mediaDir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("write media file %q: %v", name, err)
		}
	}

	loungeID := "lounge-1"
	loungeSnapshot := models.LoungeAdsSnapshot{
		LoungeID: "lounge-1",
		Items: []models.LoungeAd{
			{ID: "active-1", LoungeID: &loungeID, MediaURL: server.URL + "/active.mp4", LocalFile: "active-lounge.mp4", IsActive: true},
			{ID: "legacy", LoungeID: &loungeID, MediaURL: server.URL + "/legacy-lounge.webm", IsActive: true},
			{ID: "inactive", LoungeID: &loungeID, MediaURL: server.URL + "/inactive.mp4", LocalFile: "inactive-lounge.mp4", IsActive: false},
		},
	}
	snapshotBytes, err := json.Marshal(loungeSnapshot)
	if err != nil {
		t.Fatalf("marshal lounge snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "lounge-ads.json"), snapshotBytes, 0o644); err != nil {
		t.Fatalf("write lounge snapshot: %v", err)
	}

	agent := New(models.Config{
		ServerURL:         server.URL,
		LoungeID:          "lounge-1",
		LoungeGroup:       "lounge-group",
		TVPurpose:         models.TVPurposeBoth,
		StoreDir:          storeDir,
		RequestTimeoutMs:  1000,
		DownloadTimeoutMs: 1000,
	})
	if err := agent.SyncAds(context.Background()); err != nil {
		t.Fatalf("SyncAds() error = %v", err)
	}

	for _, name := range []string{"active-lounge.mp4", "legacy-lounge.webm"} {
		if _, err := os.Stat(filepath.Join(mediaDir, name)); err != nil {
			t.Errorf("active lounge media %q was removed: %v", name, err)
		}
	}

	for _, name := range []string{"inactive-lounge.mp4", "orphan.mp4"} {
		if _, err := os.Stat(filepath.Join(mediaDir, name)); !os.IsNotExist(err) {
			t.Errorf("inactive/orphan media %q was not removed (stat error = %v)", name, err)
		}
	}
}

func TestMergeLocalAndRemoteLoungeAdsFiltersOtherLounge(t *testing.T) {
	currentLoungeID := "lounge-1"
	otherLoungeID := "lounge-2"

	merged := mergeLocalAndRemoteLoungeAds(
		[]models.LoungeAd{
			{
				ID:       "local-current",
				LoungeID: &currentLoungeID,
				MediaURL: "/local/media/current.mp4",
			},
			{
				ID:       "local-other",
				LoungeID: &otherLoungeID,
				MediaURL: "/local/media/other.mp4",
			},
			{
				ID:              "local-default",
				IsDefaultForAll: true,
				MediaURL:        "/local/media/default.mp4",
			},
			{
				ID:       "local-unassigned",
				MediaURL: "/local/media/unassigned.mp4",
			},
		},
		[]models.LoungeAd{{
			ID:       "remote-current",
			LoungeID: &currentLoungeID,
			MediaURL: "https://example.test/remote.mp4",
		}},
		currentLoungeID,
	)

	wantIDs := []string{"local-current", "local-default", "remote-current"}
	if len(merged) != len(wantIDs) {
		t.Fatalf("merged item count = %d, want %d", len(merged), len(wantIDs))
	}
	for i, wantID := range wantIDs {
		if merged[i].ID != wantID {
			t.Errorf("merged[%d].ID = %q, want %q", i, merged[i].ID, wantID)
		}
	}
}
