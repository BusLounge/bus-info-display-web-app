package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tv-sync-agent-go/internal/models"
)

func Load(path string) (models.Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return models.Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg models.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return models.Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.ServerURL == "" || cfg.LoungeID == "" || cfg.LoungeGroup == "" {
		return models.Config{}, fmt.Errorf("serverUrl, loungeId, loungeGroup are required")
	}

	// Set defaults for new fields if they are empty
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	if cfg.DisplayMode == "" {
		cfg.DisplayMode = "both"
	}
	if cfg.LayoutMode == "" {
		cfg.LayoutMode = "split-screen"
	}
	if cfg.SyncFrequencySeconds <= 0 {
		cfg.SyncFrequencySeconds = 30
	}
	cfg.DisplayResolution = normalizeDisplayResolution(cfg.DisplayResolution)
	cfg.DisplayOrientation = normalizeDisplayOrientation(cfg.DisplayOrientation)

	purpose := strings.ToLower(strings.TrimSpace(cfg.TVPurpose))
	switch purpose {
	case "", models.TVPurposeBoth:
		cfg.TVPurpose = models.TVPurposeBoth
	case models.TVPurposeSchedule, "schedules", "schedules-only", "schedules_only", "schedule-only", "schedule_only":
		cfg.TVPurpose = models.TVPurposeSchedule
	case models.TVPurposeAds, "ads-only", "ads_only":
		cfg.TVPurpose = models.TVPurposeAds
	default:
		return models.Config{}, fmt.Errorf("invalid tvPurpose %q (allowed: both, schedule, ads)", cfg.TVPurpose)
	}

	if cfg.LocalBridgePort == 0 {
		cfg.LocalBridgePort = 3001
	}
	if cfg.RequestTimeoutMs == 0 {
		cfg.RequestTimeoutMs = 10000
	}
	if cfg.DownloadTimeoutMs == 0 {
		cfg.DownloadTimeoutMs = 120000
	}
	if cfg.StoreDir == "" {
		cfg.StoreDir = "./local-store"
	}
	if strings.TrimSpace(cfg.BroadcastIntervalCron) == "" {
		cfg.BroadcastIntervalCron = "*/1 * * * *"
	}

	cfg.StoreDir = filepath.Clean(cfg.StoreDir)
	return cfg, nil
}

// normalizeDisplayResolution returns "auto" or a "WIDTHxHEIGHT" canvas size.
// The kiosk display draws on that canvas and scales it to fit the screen, so
// any sensible size is allowed; anything else falls back to "auto".
func normalizeDisplayResolution(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.ReplaceAll(v, "×", "x")
	v = strings.ReplaceAll(v, " ", "")
	switch v {
	case "720p", "hd":
		return "1280x720"
	case "1080p", "fullhd":
		return "1920x1080"
	case "4k", "4kultrahd", "7680x4320", "8k", "8kultrahd":
		// An 8K canvas would show ~66 tiny rows; 4K is the largest preset.
		return "3840x2160"
	}
	m := displayResolutionPattern.FindStringSubmatch(v)
	if len(m) == 3 {
		width, _ := strconv.Atoi(m[1])
		height, _ := strconv.Atoi(m[2])
		if width >= 640 && height >= 480 && width <= 3840 && height <= 3840 {
			return fmt.Sprintf("%dx%d", width, height)
		}
	}
	return "auto"
}

var displayResolutionPattern = regexp.MustCompile(`^(\d{3,4})x(\d{3,4})$`)

func normalizeDisplayOrientation(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "portrait", "portrait-primary":
		return "portrait"
	default:
		return "landscape"
	}
}

func CronEveryDuration(expr string, fallback time.Duration) time.Duration {
	// Supports patterns like */5 * * * *
	re := regexp.MustCompile(`^\*/(\d+)\s+\*\s+\*\s+\*\s+\*$`)
	m := re.FindStringSubmatch(strings.TrimSpace(expr))
	if len(m) != 2 {
		return fallback
	}
	minutes, err := strconv.Atoi(m[1])
	if err != nil || minutes <= 0 {
		return fallback
	}
	return time.Duration(minutes) * time.Minute
}
