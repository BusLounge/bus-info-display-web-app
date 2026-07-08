package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tv-sync-agent-go/internal/config"
	"tv-sync-agent-go/internal/models"
)

var errMediaNotFound = errors.New("media not found")

type Agent struct {
	cfg           models.Config
	httpClient    *http.Client
	downloadHTTP  *http.Client
	storeDir      string
	mediaDir      string
	schedulePath  string
	adsPath       string
	broadcastPath string
	loungeAdsPath string

	mu     sync.RWMutex
	status models.AgentStatus
}

func New(cfg models.Config) *Agent {
	storeDir := cfg.StoreDir
	mediaDir := filepath.Join(storeDir, "media")

	_ = os.MkdirAll(mediaDir, 0o755)

	return &Agent{
		cfg:           cfg,
		httpClient:    &http.Client{Timeout: time.Duration(cfg.RequestTimeoutMs) * time.Millisecond},
		downloadHTTP:  &http.Client{Timeout: time.Duration(cfg.DownloadTimeoutMs) * time.Millisecond},
		storeDir:      storeDir,
		mediaDir:      mediaDir,
		schedulePath:  filepath.Join(storeDir, "schedule.json"),
		adsPath:       filepath.Join(storeDir, "ads-manifest.json"),
		broadcastPath: filepath.Join(storeDir, "broadcasts.json"),
		loungeAdsPath: filepath.Join(storeDir, "lounge-ads.json"),
		status: models.AgentStatus{
			StartedAt:            time.Now().UTC(),
			TVPurpose:            cfg.TVPurpose,
			Language:             cfg.Language,
			DisplayMode:          displayModeFromPurpose(cfg.TVPurpose),
			LayoutMode:           "split",
			SyncFrequencySeconds: cfg.SyncFrequencySeconds,
			DisplayResolution:    cfg.DisplayResolution,
			BroadcastsEnabled:    true,
			ScheduleEnabled:      cfg.ScheduleEnabled(),
			AdsEnabled:           cfg.AdsEnabled(),
			SchedulePath:         filepath.Join(storeDir, "schedule.json"),
			AdsManifestPath:      filepath.Join(storeDir, "ads-manifest.json"),
			BroadcastPath:        filepath.Join(storeDir, "broadcasts.json"),
			LoungeAdsPath:        filepath.Join(storeDir, "lounge-ads.json"),
		},
	}
}

func (a *Agent) Status() models.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *Agent) SetLanguage(language string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(language))
	switch normalized {
	case "en", "si", "ta":
		// allowed values
	default:
		return "", fmt.Errorf("invalid language %q (allowed: en, si, ta)", language)
	}

	a.mu.Lock()
	a.cfg.Language = normalized
	a.status.Language = normalized
	a.mu.Unlock()

	return normalized, nil
}

func displayModeFromPurpose(purpose string) string {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case models.TVPurposeSchedule:
		return "schedules"
	case models.TVPurposeAds:
		return "ads"
	default:
		return "both"
	}
}

func purposeFromDisplayMode(displayMode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(displayMode)) {
	case "schedules", "schedule", "schedule-only", "schedule_only", "schedules-only", "schedules_only":
		return models.TVPurposeSchedule, nil
	case "ads", "ads-only", "ads_only":
		return models.TVPurposeAds, nil
	case "both", "hybrid":
		return models.TVPurposeBoth, nil
	default:
		return "", fmt.Errorf("invalid displayMode %q (allowed: schedules, ads, both)", displayMode)
	}
}

func normalizeLayoutMode(layoutMode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(layoutMode)) {
	case "split", "split-screen", "split_screen":
		return "split", nil
	case "alternate", "full-screen-alternate", "full_screen_alternate":
		return "alternate", nil
	default:
		return "", fmt.Errorf("invalid layoutMode %q (allowed: split, alternate)", layoutMode)
	}
}

func (a *Agent) SetDisplayMode(displayMode string) (string, error) {
	purpose, err := purposeFromDisplayMode(displayMode)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.cfg.TVPurpose = purpose
	a.status.TVPurpose = purpose
	a.status.DisplayMode = displayModeFromPurpose(purpose)
	a.status.ScheduleEnabled = a.cfg.ScheduleEnabled()
	a.status.AdsEnabled = a.cfg.AdsEnabled()
	a.mu.Unlock()

	return displayModeFromPurpose(purpose), nil
}

func (a *Agent) SetLayoutMode(layoutMode string) (string, error) {
	normalized, err := normalizeLayoutMode(layoutMode)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.status.LayoutMode = normalized
	a.mu.Unlock()

	return normalized, nil
}

func (a *Agent) SetBroadcastsEnabled(enabled bool) bool {
	a.mu.Lock()
	a.status.BroadcastsEnabled = enabled
	a.mu.Unlock()
	return enabled
}

func (a *Agent) StoreDir() string { return a.storeDir }

func (a *Agent) ScheduleEnabled() bool { return a.cfg.ScheduleEnabled() }

func (a *Agent) AdsEnabled() bool { return a.cfg.AdsEnabled() }

func (a *Agent) SyncBroadcasts(ctx context.Context) error {
	broadcastURL := fmt.Sprintf("%s/api/tv/broadcasts", strings.TrimRight(a.cfg.ServerURL, "/"))
	v, err := a.fetchJSON(ctx, broadcastURL)
	if err != nil {
		a.setBroadcastError(err)
		return err
	}

	raw, err := json.Marshal(v)
	if err != nil {
		a.setBroadcastError(err)
		return err
	}

	var items []models.BroadcastMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		a.setBroadcastError(fmt.Errorf("unmarshal broadcasts: %w", err))
		return err
	}

	snapshot := models.BroadcastSnapshot{
		UpdatedAt: time.Now().UTC(),
		SourceURL: broadcastURL,
		Items:     items,
	}
	if err := writeJSONAtomic(a.broadcastPath, snapshot); err != nil {
		a.setBroadcastError(err)
		return err
	}

	a.mu.Lock()
	a.status.LastBroadcastSync = time.Now().UTC()
	a.status.LastBroadcastErr = ""
	a.status.BroadcastCount = len(items)
	a.mu.Unlock()
	return nil
}

func (a *Agent) SyncLoungeAds(ctx context.Context) error {
	loungeAdsURL := fmt.Sprintf("%s/api/lounge-ads/lounge/%s", strings.TrimRight(a.cfg.ServerURL, "/"), url.PathEscape(a.cfg.LoungeID))
	v, err := a.fetchJSON(ctx, loungeAdsURL)
	if err != nil {
		a.setLoungeAdsError(err)
		return err
	}

	raw, err := json.Marshal(v)
	if err != nil {
		a.setLoungeAdsError(err)
		return err
	}

	var items []models.LoungeAd
	if err := json.Unmarshal(raw, &items); err != nil {
		a.setLoungeAdsError(fmt.Errorf("unmarshal lounge ads: %w", err))
		return err
	}

	// Keep locally uploaded lounge ads so periodic remote sync does not overwrite
	// ads created directly on the TV box via /local/lounge-ads APIs.
	existingSnapshot, loadErr := a.loadLoungeAds()
	if loadErr != nil {
		log.Printf("warning: load existing lounge ads snapshot: %v", loadErr)
	}
	a.hydrateMissingLoungeMedia(existingSnapshot.Items)
	items = mergeLocalAndRemoteLoungeAds(existingSnapshot.Items, items)
	a.hydrateMissingLoungeMedia(items)

	snapshot := models.LoungeAdsSnapshot{
		LoungeID:  a.cfg.LoungeID,
		UpdatedAt: time.Now().UTC(),
		SourceURL: loungeAdsURL,
		Items:     items,
	}
	if err := writeJSONAtomic(a.loungeAdsPath, snapshot); err != nil {
		a.setLoungeAdsError(err)
		return err
	}

	a.mu.Lock()
	a.status.LastLoungeAdsSync = time.Now().UTC()
	a.status.LastLoungeAdsErr = ""
	a.status.LoungeAdsCount = len(items)
	a.mu.Unlock()
	return nil
}

func mergeLocalAndRemoteLoungeAds(existing, remote []models.LoungeAd) []models.LoungeAd {
	merged := make([]models.LoungeAd, 0, len(existing)+len(remote))
	seen := make(map[string]struct{})

	// Local ads first so TV playback prefers what was created on this device.
	for _, item := range existing {
		if !isLocalUploadedLoungeAd(item) {
			continue
		}
		key := loungeAdMergeKey(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		merged = append(merged, item)
		seen[key] = struct{}{}
	}

	for _, item := range remote {
		key := loungeAdMergeKey(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		merged = append(merged, item)
		seen[key] = struct{}{}
	}

	return merged
}

func isLocalUploadedLoungeAd(item models.LoungeAd) bool {
	return strings.HasPrefix(strings.TrimSpace(item.MediaURL), "/local/media/")
}

func loungeAdMergeKey(item models.LoungeAd) string {
	id := strings.TrimSpace(item.ID)
	if id != "" {
		return "id:" + id
	}
	url := strings.TrimSpace(item.MediaURL)
	if url != "" {
		return "url:" + url
	}
	return ""
}

func (a *Agent) SyncSchedule(ctx context.Context) error {
	if !a.cfg.ScheduleEnabled() {
		return nil
	}

	depURL := fmt.Sprintf("%s/api/departures/lounge/%s", strings.TrimRight(a.cfg.ServerURL, "/"), url.PathEscape(a.cfg.LoungeID))
	arrURL := fmt.Sprintf("%s/api/arrivals/lounge/%s", strings.TrimRight(a.cfg.ServerURL, "/"), url.PathEscape(a.cfg.LoungeID))

	depRaw, err := a.fetchJSON(ctx, depURL)
	if err != nil {
		a.setScheduleError(err)
		return err
	}
	arrRaw, err := a.fetchJSON(ctx, arrURL)
	if err != nil {
		a.setScheduleError(err)
		return err
	}

	snapshot := models.ScheduleSnapshot{
		LoungeID:      a.cfg.LoungeID,
		UpdatedAt:     time.Now().UTC(),
		SourceURL:     a.cfg.ServerURL,
		DeparturesRaw: depRaw,
		ArrivalsRaw:   arrRaw,
	}

	if err := writeJSONAtomic(a.schedulePath, snapshot); err != nil {
		a.setScheduleError(err)
		return err
	}

	a.mu.Lock()
	a.status.LastScheduleSync = time.Now().UTC()
	a.status.LastScheduleErr = ""
	a.mu.Unlock()
	return nil
}

func (a *Agent) SyncAds(ctx context.Context) error {
	if !a.cfg.AdsEnabled() {
		return nil
	}

	adsURL := fmt.Sprintf("%s/api/tv/ads/%s", strings.TrimRight(a.cfg.ServerURL, "/"), url.PathEscape(a.cfg.LoungeGroup))
	v, err := a.fetchJSON(ctx, adsURL)
	if err != nil {
		a.setAdsError(err)
		return err
	}

	raw, err := json.Marshal(v)
	if err != nil {
		a.setAdsError(err)
		return err
	}

	var remoteItems []models.TVAdManifestItem
	if err := json.Unmarshal(raw, &remoteItems); err != nil {
		a.setAdsError(fmt.Errorf("unmarshal tv ads manifest: %w", err))
		return err
	}

	manifest, err := a.loadAdsManifest()
	if err != nil {
		log.Printf("warning: load old ads manifest: %v", err)
	}
	oldByID := map[string]models.AdManifestItem{}
	for _, item := range manifest.Items {
		oldByID[item.ID] = item
	}

	items := make([]models.AdManifestItem, 0)
	keepFiles := map[string]bool{}

	for _, remote := range remoteItems {
		if strings.TrimSpace(remote.ID) == "" || strings.TrimSpace(remote.MediaURL) == "" {
			continue
		}

		mediaURL := strings.TrimSpace(remote.MediaURL)
		fileExt := strings.ToLower(filepath.Ext(mediaURL))
		if fileExt == "" {
			fileExt = ".avi"
		}
		localFile := remote.ID + fileExt
		localAbs := filepath.Join(a.mediaDir, localFile)

		mediaHash := strings.TrimSpace(remote.MediaHash)
		if mediaHash == "" {
			mediaHash = sha256Hex(mediaURL)
		}

		prev, hasPrev := oldByID[remote.ID]
		needsDownload := !hasPrev || prev.MediaHash != mediaHash || prev.LocalFile != localFile || !fileExists(localAbs)

		if needsDownload {
			if err := a.downloadToFile(ctx, mediaURL, localAbs); err != nil {
				if errors.Is(err, errMediaNotFound) {
					log.Printf("warning: skipping ad %s (%s): %v", remote.ID, remote.AdvertisementName, err)
					continue
				}
				a.setAdsError(fmt.Errorf("download ad %s: %w", remote.ID, err))
				return err
			}
		}

		keepFiles[localFile] = true
		items = append(items, models.AdManifestItem{
			ID:            remote.ID,
			Name:          remote.AdvertisementName,
			MediaURL:      mediaURL,
			MediaType:     remote.MediaType,
			MediaDuration: remote.MediaDuration,
			LocalFile:     localFile,
			MediaHash:     mediaHash,
			SourceVersion: mediaHash,
			Priority:      remote.Priority,
			ScheduleType:  remote.ScheduleType,
			Frequency:     remote.Frequency,
			StartDate:     remote.StartDate,
			EndDate:       remote.EndDate,
			StartTime:     remote.StartTime,
			EndTime:       remote.EndTime,
			PlayTimeSlot:  remote.PlayTimeSlot,
			PlayTimeSlots: remote.PlayTimeSlots,
			NextPlayAt:    remote.NextPlayAt,
		})
	}

	if err := cleanupMediaFiles(a.mediaDir, keepFiles); err != nil {
		log.Printf("warning: cleanup media files: %v", err)
	}

	snapshot := models.AdsSnapshot{
		LoungeGroup: a.cfg.LoungeGroup,
		UpdatedAt:   time.Now().UTC(),
		SourceURL:   adsURL,
		Items:       items,
	}
	if err := writeJSONAtomic(a.adsPath, snapshot); err != nil {
		a.setAdsError(err)
		return err
	}

	a.mu.Lock()
	a.status.LastAdsSync = time.Now().UTC()
	a.status.LastAdsErr = ""
	a.status.AdsCount = len(items)
	a.mu.Unlock()
	return nil
}

func (a *Agent) GetLocalLoungeAdSlotSummary() (models.LoungeAdLocalSlotSummary, error) {
	const scheduleWindow = 6
	const adWindow = 24

	manifest, err := a.loadAdsManifest()
	if err != nil {
		return models.LoungeAdLocalSlotSummary{}, fmt.Errorf("load ads manifest: %w", err)
	}

	booked := 0
	for _, item := range manifest.Items {
		if item.MediaDuration == nil || *item.MediaDuration <= 0 {
			continue
		}
		booked += *item.MediaDuration
	}

	if booked < 0 {
		booked = 0
	}
	if booked > adWindow {
		booked = adWindow
	}

	remaining := adWindow - booked
	timeSlots, availableSlots := buildHybridTimeSlots(scheduleWindow, adWindow, booked)

	return models.LoungeAdLocalSlotSummary{
		ScheduleWindowSeconds: scheduleWindow,
		AdWindowSeconds:       adWindow,
		BookedSeconds:         booked,
		RemainingSeconds:      remaining,
		TimeSlots:             timeSlots,
		AvailableSlots:        availableSlots,
		FallbackRequired:      remaining > 0,
		FallbackSeconds:       remaining,
	}, nil
}

func (a *Agent) CreateLocalLoungeAd(ctx context.Context, req models.LoungeAdCreateRequest) (*models.LoungeAd, error) {
	name := strings.TrimSpace(req.AdvertisementName)
	if name == "" {
		return nil, fmt.Errorf("advertisementName is required")
	}

	mediaURL := strings.TrimSpace(req.MediaURL)
	if mediaURL == "" {
		return nil, fmt.Errorf("mediaUrl is required")
	}

	mediaType := strings.ToLower(strings.TrimSpace(req.MediaType))
	if mediaType == "" {
		mediaType = "image"
	}
	if mediaType != "image" && mediaType != "video" {
		return nil, fmt.Errorf("mediaType must be image or video")
	}

	duration := req.DurationSeconds
	if duration < 0 {
		duration = 0
	}

	priority := strings.ToLower(strings.TrimSpace(req.Priority))
	if priority == "" {
		priority = "normal"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isDefaultForAll := false
	if req.IsDefaultForAll != nil {
		isDefaultForAll = *req.IsDefaultForAll
	}

	snapshot, err := a.loadLoungeAds()
	if err != nil {
		return nil, err
	}

	id := makeLocalLoungeAdID(name)
	localFile, err := a.prepareLoungeAdMedia(ctx, id, mediaURL)
	if err != nil {
		return nil, err
	}
	if localFile != "" {
		mediaURL = "/local/media/" + localFile
	}

	ad := models.LoungeAd{
		ID:                id,
		AdvertisementName: name,
		MediaURL:          mediaURL,
		MediaType:         mediaType,
		DurationSeconds:   duration,
		Priority:          priority,
		IsActive:          isActive,
		IsDefaultForAll:   isDefaultForAll,
	}
	if !isDefaultForAll {
		loungeID := strings.TrimSpace(a.cfg.LoungeID)
		if loungeID != "" {
			ad.LoungeID = &loungeID
		}
	}

	snapshot.LoungeID = a.cfg.LoungeID
	snapshot.SourceURL = "local://lounge-ads"
	snapshot.UpdatedAt = time.Now().UTC()
	snapshot.Items = append([]models.LoungeAd{ad}, snapshot.Items...)

	if err := writeJSONAtomic(a.loungeAdsPath, snapshot); err != nil {
		return nil, fmt.Errorf("write local lounge ads snapshot: %w", err)
	}

	a.mu.Lock()
	a.status.LastLoungeAdsSync = time.Now().UTC()
	a.status.LastLoungeAdsErr = ""
	a.status.LoungeAdsCount = len(snapshot.Items)
	a.mu.Unlock()

	return &ad, nil
}

func (a *Agent) CreateLocalLoungeAdFromUpload(req models.LoungeAdCreateRequest, fileName string, file io.Reader) (*models.LoungeAd, error) {
	name := strings.TrimSpace(req.AdvertisementName)
	if name == "" {
		return nil, fmt.Errorf("advertisementName is required")
	}

	id := makeLocalLoungeAdID(name)
	mediaType, localFile, err := a.saveUploadedLoungeMedia(id, fileName, file)
	if err != nil {
		return nil, err
	}

	req.MediaType = mediaType
	req.MediaURL = "/local/media/" + localFile
	req.DurationSeconds = 0
	req.Priority = "normal"

	return a.CreateLocalLoungeAd(context.Background(), req)
}

func (a *Agent) UpdateLocalLoungeAd(id string, req models.LoungeAdUpdateRequest) (*models.LoungeAd, error) {
	snapshot, err := a.loadLoungeAds()
	if err != nil {
		return nil, err
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	idx := -1
	for i := range snapshot.Items {
		if snapshot.Items[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("lounge ad not found")
	}

	item := snapshot.Items[idx]
	name := strings.TrimSpace(req.AdvertisementName)
	if name != "" {
		item.AdvertisementName = name
	}

	mediaURL := strings.TrimSpace(req.MediaURL)
	if mediaURL != "" {
		item.MediaURL = mediaURL
	}

	mediaType := strings.ToLower(strings.TrimSpace(req.MediaType))
	if mediaType != "" {
		if mediaType != "image" && mediaType != "video" {
			return nil, fmt.Errorf("mediaType must be image or video")
		}
		item.MediaType = mediaType
	}

	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}
	if req.IsDefaultForAll != nil {
		item.IsDefaultForAll = *req.IsDefaultForAll
		if item.IsDefaultForAll {
			item.LoungeID = nil
		} else {
			loungeID := strings.TrimSpace(a.cfg.LoungeID)
			if loungeID != "" {
				item.LoungeID = &loungeID
			}
		}
	}

	item.DurationSeconds = 0
	item.Priority = "normal"

	snapshot.Items[idx] = item
	snapshot.LoungeID = a.cfg.LoungeID
	snapshot.SourceURL = "local://lounge-ads"
	snapshot.UpdatedAt = time.Now().UTC()

	if err := writeJSONAtomic(a.loungeAdsPath, snapshot); err != nil {
		return nil, fmt.Errorf("write local lounge ads snapshot: %w", err)
	}

	a.mu.Lock()
	a.status.LastLoungeAdsSync = time.Now().UTC()
	a.status.LastLoungeAdsErr = ""
	a.status.LoungeAdsCount = len(snapshot.Items)
	a.mu.Unlock()

	return &snapshot.Items[idx], nil
}

func (a *Agent) UpdateLocalLoungeAdMedia(id, fileName string, file io.Reader) (*models.LoungeAd, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	snapshot, err := a.loadLoungeAds()
	if err != nil {
		return nil, err
	}

	idx := -1
	for i := range snapshot.Items {
		if snapshot.Items[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("lounge ad not found")
	}

	oldFile := localMediaFileFromURL(snapshot.Items[idx].MediaURL)
	mediaType, localFile, err := a.saveUploadedLoungeMedia(id, fileName, file)
	if err != nil {
		return nil, err
	}

	snapshot.Items[idx].MediaURL = "/local/media/" + localFile
	snapshot.Items[idx].MediaType = mediaType
	snapshot.Items[idx].DurationSeconds = 0
	snapshot.Items[idx].Priority = "normal"

	if oldFile != "" && oldFile != localFile {
		_ = os.Remove(filepath.Join(a.mediaDir, oldFile))
		a.removeMirroredLoungeMedia(oldFile)
	}

	snapshot.UpdatedAt = time.Now().UTC()
	snapshot.SourceURL = "local://lounge-ads"
	if err := writeJSONAtomic(a.loungeAdsPath, snapshot); err != nil {
		return nil, fmt.Errorf("write local lounge ads snapshot: %w", err)
	}

	a.mu.Lock()
	a.status.LastLoungeAdsSync = time.Now().UTC()
	a.status.LastLoungeAdsErr = ""
	a.status.LoungeAdsCount = len(snapshot.Items)
	a.mu.Unlock()

	return &snapshot.Items[idx], nil
}

func (a *Agent) DeleteLocalLoungeAd(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}

	snapshot, err := a.loadLoungeAds()
	if err != nil {
		return err
	}

	idx := -1
	for i := range snapshot.Items {
		if snapshot.Items[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("lounge ad not found")
	}

	removeFile := localMediaFileFromURL(snapshot.Items[idx].MediaURL)
	snapshot.Items = append(snapshot.Items[:idx], snapshot.Items[idx+1:]...)
	snapshot.UpdatedAt = time.Now().UTC()
	snapshot.SourceURL = "local://lounge-ads"

	if err := writeJSONAtomic(a.loungeAdsPath, snapshot); err != nil {
		return fmt.Errorf("write local lounge ads snapshot: %w", err)
	}

	if removeFile != "" {
		_ = os.Remove(filepath.Join(a.mediaDir, removeFile))
		a.removeMirroredLoungeMedia(removeFile)
	}

	a.mu.Lock()
	a.status.LastLoungeAdsSync = time.Now().UTC()
	a.status.LastLoungeAdsErr = ""
	a.status.LoungeAdsCount = len(snapshot.Items)
	a.mu.Unlock()

	return nil
}

func (a *Agent) Run(ctx context.Context) {
	if a.cfg.ScheduleEnabled() {
		if err := a.SyncSchedule(ctx); err != nil {
			log.Printf("initial schedule sync failed: %v", err)
		}
	}
	if a.cfg.AdsEnabled() {
		if err := a.SyncAds(ctx); err != nil {
			log.Printf("initial ads sync failed: %v", err)
		}
	}
	if err := a.SyncBroadcasts(ctx); err != nil {
		log.Printf("initial broadcast sync failed: %v", err)
	}
	if err := a.SyncLoungeAds(ctx); err != nil {
		log.Printf("initial lounge ad sync failed: %v", err)
	}

	sTicker := time.NewTicker(config.CronEveryDuration(a.cfg.ScheduleIntervalCron, 5*time.Minute))
	defer sTicker.Stop()
	aTicker := time.NewTicker(config.CronEveryDuration(a.cfg.AdsIntervalCron, 15*time.Minute))
	defer aTicker.Stop()
	bTicker := time.NewTicker(config.CronEveryDuration(a.cfg.BroadcastIntervalCron, 1*time.Minute))
	defer bTicker.Stop()
	lTicker := time.NewTicker(config.CronEveryDuration(a.cfg.BroadcastIntervalCron, 1*time.Minute))
	defer lTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sTicker.C:
			if err := a.SyncSchedule(ctx); err != nil {
				log.Printf("schedule sync failed: %v", err)
			}
		case <-aTicker.C:
			if err := a.SyncAds(ctx); err != nil {
				log.Printf("ads sync failed: %v", err)
			}
		case <-bTicker.C:
			if err := a.SyncBroadcasts(ctx); err != nil {
				log.Printf("broadcast sync failed: %v", err)
			}
		case <-lTicker.C:
			if err := a.SyncLoungeAds(ctx); err != nil {
				log.Printf("lounge ad sync failed: %v", err)
			}
		}
	}
}

func (a *Agent) fetchJSON(ctx context.Context, endpoint string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	var v interface{}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func (a *Agent) downloadToFile(ctx context.Context, mediaURL, dst string) error {
	tmp := dst + ".part"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return err
	}
	resp, err := a.downloadHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w: %s", errMediaNotFound, mediaURL)
		}
		return fmt.Errorf("download %s http %d", mediaURL, resp.StatusCode)
	}
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func (a *Agent) loadAdsManifest() (models.AdsSnapshot, error) {
	b, err := os.ReadFile(a.adsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.AdsSnapshot{}, nil
		}
		return models.AdsSnapshot{}, err
	}
	var m models.AdsSnapshot
	if err := json.Unmarshal(b, &m); err != nil {
		return models.AdsSnapshot{}, err
	}
	return m, nil
}

func (a *Agent) loadLoungeAds() (models.LoungeAdsSnapshot, error) {
	b, err := os.ReadFile(a.loungeAdsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.LoungeAdsSnapshot{
				LoungeID:  a.cfg.LoungeID,
				UpdatedAt: time.Now().UTC(),
				SourceURL: "local://lounge-ads",
				Items:     make([]models.LoungeAd, 0),
			}, nil
		}
		return models.LoungeAdsSnapshot{}, err
	}
	var snapshot models.LoungeAdsSnapshot
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return models.LoungeAdsSnapshot{}, err
	}
	if snapshot.Items == nil {
		snapshot.Items = make([]models.LoungeAd, 0)
	}
	a.hydrateMissingLoungeMedia(snapshot.Items)
	return snapshot, nil
}

func (a *Agent) hydrateMissingLoungeMedia(items []models.LoungeAd) {
	sharedDir := a.workspaceSharedMediaDir()
	if sharedDir == "" {
		return
	}

	for _, item := range items {
		localFile := localMediaFileFromURL(item.MediaURL)
		if localFile == "" {
			continue
		}

		dst := filepath.Join(a.mediaDir, localFile)
		if fileExists(dst) {
			continue
		}

		src := filepath.Join(sharedDir, localFile)
		if !fileExists(src) {
			continue
		}

		if err := copyFileWithDirs(src, dst); err != nil {
			log.Printf("warning: hydrate lounge media %s failed: %v", localFile, err)
		}
	}
}

func (a *Agent) prepareLoungeAdMedia(ctx context.Context, adID, mediaURL string) (string, error) {
	parsed, err := url.Parse(mediaURL)
	if err != nil {
		return "", fmt.Errorf("invalid mediaUrl: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", nil
	}

	ext := strings.ToLower(filepath.Ext(parsed.Path))
	if ext == "" {
		ext = ".avi"
	}
	localFile := adID + ext
	if err := a.downloadToFile(ctx, mediaURL, filepath.Join(a.mediaDir, localFile)); err != nil {
		return "", fmt.Errorf("download media: %w", err)
	}
	a.mirrorLoungeMediaToWorkspaceStore(localFile)
	return localFile, nil
}

func (a *Agent) saveUploadedLoungeMedia(adID, fileName string, file io.Reader) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".mp4"
	}
	mediaType := mediaTypeFromExtension(ext)
	if mediaType == "" {
		return "", "", fmt.Errorf("unsupported media file type")
	}

	localFile := adID + ext
	dst := filepath.Join(a.mediaDir, localFile)
	tmp := dst + ".part"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(0o644))
	if err != nil {
		return "", "", err
	}
	if _, err := io.Copy(f, file); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", "", err
	}
	a.mirrorLoungeMediaToWorkspaceStore(localFile)

	return mediaType, localFile, nil
}

func (a *Agent) mirrorLoungeMediaToWorkspaceStore(localFile string) {
	sharedDir := a.workspaceSharedMediaDir()
	if sharedDir == "" {
		return
	}

	src := filepath.Join(a.mediaDir, localFile)
	dst := filepath.Join(sharedDir, localFile)
	if err := copyFileWithDirs(src, dst); err != nil {
		log.Printf("warning: mirror lounge media %s to workspace store failed: %v", localFile, err)
	}
}

func (a *Agent) removeMirroredLoungeMedia(localFile string) {
	sharedDir := a.workspaceSharedMediaDir()
	if sharedDir == "" {
		return
	}
	_ = os.Remove(filepath.Join(sharedDir, localFile))
}

func (a *Agent) workspaceSharedMediaDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	candidate := filepath.Join(wd, "..", "local-store", "media")
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return ""
	}
	absAgentMedia, err := filepath.Abs(a.mediaDir)
	if err == nil && strings.EqualFold(absCandidate, absAgentMedia) {
		return ""
	}

	return absCandidate
}

func copyFileWithDirs(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func mediaTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4", ".webm", ".avi", ".mov", ".mkv", ".m4v":
		return "video"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return "image"
	default:
		return ""
	}
}

func localMediaFileFromURL(mediaURL string) string {
	const prefix = "/local/media/"
	if strings.HasPrefix(mediaURL, prefix) {
		name := filepath.Base(strings.TrimSpace(mediaURL[len(prefix):]))
		if name == "." || name == "" || name == "/" {
			return ""
		}
		return name
	}
	return ""
}

func buildHybridTimeSlots(scheduleSeconds, adWindowSeconds, bookedSeconds int) ([]models.LoungeAdTimeSlot, []models.LoungeAdTimeSlot) {
	if scheduleSeconds < 0 {
		scheduleSeconds = 0
	}
	if adWindowSeconds < 0 {
		adWindowSeconds = 0
	}
	if bookedSeconds < 0 {
		bookedSeconds = 0
	}
	if bookedSeconds > adWindowSeconds {
		bookedSeconds = adWindowSeconds
	}

	cycleStart := 0
	scheduleEnd := scheduleSeconds
	adStart := scheduleEnd
	companyEnd := adStart + bookedSeconds
	adEnd := adStart + adWindowSeconds

	slots := []models.LoungeAdTimeSlot{
		{
			Type:            "schedule",
			Label:           "Bus schedules",
			StartSecond:     cycleStart,
			EndSecond:       scheduleEnd,
			DurationSeconds: scheduleSeconds,
			Interactive:     false,
		},
	}

	if bookedSeconds > 0 {
		slots = append(slots, models.LoungeAdTimeSlot{
			Type:            "company",
			Label:           "Company ads",
			StartSecond:     adStart,
			EndSecond:       companyEnd,
			DurationSeconds: bookedSeconds,
			Interactive:     false,
		})
	}

	available := make([]models.LoungeAdTimeSlot, 0, 1)
	remaining := adEnd - companyEnd
	if remaining > 0 {
		slot := models.LoungeAdTimeSlot{
			Type:            "available",
			Label:           "Available for lounge-specific ads",
			StartSecond:     companyEnd,
			EndSecond:       adEnd,
			DurationSeconds: remaining,
			Interactive:     true,
		}
		slots = append(slots, slot)
		available = append(available, slot)
	}

	return slots, available
}

func cleanupMediaFiles(dir string, keep map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				log.Printf("warning: remove stale media %s: %v", e.Name(), err)
			}
		}
	}
	return nil
}

func writeJSONAtomic(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func makeLocalLoungeAdID(seed string) string {
	raw := fmt.Sprintf("%d:%s", time.Now().UTC().UnixNano(), seed)
	return sha256Hex(raw)[:32]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *Agent) setScheduleError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.LastScheduleErr = err.Error()
}

func (a *Agent) setBroadcastError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.LastBroadcastErr = err.Error()
}

func (a *Agent) setLoungeAdsError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.LastLoungeAdsErr = err.Error()
}

func (a *Agent) setAdsError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.LastAdsErr = err.Error()
}
