package handlers

import (
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type AdvertisementHandler struct {
	service  *services.AdvertisementService
	cache    *responseCache
	mediaDir string
}

func NewAdvertisementHandler(service *services.AdvertisementService, mediaDir string) *AdvertisementHandler {
	return &AdvertisementHandler{
		service:  service,
		cache:    newResponseCache(10 * time.Second),
		mediaDir: mediaDir,
	}
}

func (h *AdvertisementHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid upload payload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	allowedExt := map[string]bool{
		".avi":  true,
		".mp4":  true,
		".webm": true,
		".ogg":  true,
		".mov":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowedExt[ext] {
		respondWithError(w, http.StatusBadRequest, "Unsupported file format")
		return
	}

	mediaType := "image"
	if ext == ".avi" || ext == ".mp4" || ext == ".webm" || ext == ".ogg" || ext == ".mov" {
		mediaType = "video"
	}

	if err := os.MkdirAll(h.mediaDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to prepare media directory")
		return
	}

	safeBase := sanitizeFileName(strings.TrimSuffix(header.Filename, ext))
	if safeBase == "" {
		safeBase = "media"
	}

	storedFileName := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), safeBase, ext)
	storedFilePath := filepath.Join(h.mediaDir, storedFileName)
	log.Printf("Saving uploaded media file to: %s", storedFilePath)

	dst, err := os.Create(storedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to store file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"fileName":  storedFileName,
		"mediaUrl":  "/media/" + storedFileName,
		"mediaType": mediaType,
	})
}

var fileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9-_]+`)

func sanitizeFileName(name string) string {
	s := fileNameSanitizer.ReplaceAllString(name, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

// Conflict check
func (h *AdvertisementHandler) CheckConflict(w http.ResponseWriter, r *http.Request) {
	var req models.ConflictCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request data")
		return
	}

	resp, err := h.service.CheckConflicts(&req)
	if err != nil {
		log.Printf("Error checking conflicts: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to check conflicts")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *AdvertisementHandler) GetCalculationRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.service.GetCalculationRates()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch calculation rates")
		return
	}
	respondWithJSON(w, http.StatusOK, rates)
}

func (h *AdvertisementHandler) UpsertCalculationRate(w http.ResponseWriter, r *http.Request) {
	trafficLevel := strings.TrimSpace(mux.Vars(r)["trafficLevel"])
	if trafficLevel == "" {
		respondWithError(w, http.StatusBadRequest, "trafficLevel is required")
		return
	}

	var req models.AdvertisementCalculationRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rate, err := h.service.UpsertCalculationRate(trafficLevel, req.CostPerSecond)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, rate)
}

func (h *AdvertisementHandler) RecordPlaybackLog(w http.ResponseWriter, r *http.Request) {
	var req models.AdvertisementPlaybackLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	item, err := h.service.RecordPlaybackLog(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, item)
}

func (h *AdvertisementHandler) GetPlaybackLogs(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDate := strings.TrimSpace(r.URL.Query().Get("endDate"))
	advertisementID := strings.TrimSpace(r.URL.Query().Get("advertisementId"))
	trafficLevel := strings.TrimSpace(r.URL.Query().Get("trafficLevel"))

	limit := 200
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			respondWithError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	logs, err := h.service.GetPlaybackLogs(startDate, endDate, advertisementID, trafficLevel, limit)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"startDate":       startDate,
		"endDate":         endDate,
		"advertisementId": advertisementID,
		"trafficLevel":    trafficLevel,
		"count":           len(logs),
		"rows":            logs,
	})
}

func (h *AdvertisementHandler) GetCostReport(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDate := strings.TrimSpace(r.URL.Query().Get("endDate"))

	report, err := h.service.GetCostReport(startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"startDate": startDate,
		"endDate":   endDate,
		"rows":      report,
	})
}

func (h *AdvertisementHandler) SyncScheduledAdsCosts(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDate := strings.TrimSpace(r.URL.Query().Get("endDate"))

	if startDate == "" || endDate == "" {
		respondWithError(w, http.StatusBadRequest, "startDate and endDate are required")
		return
	}

	count, err := h.service.SyncScheduledAdsCosts(startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":             "Scheduled ad costs synced successfully",
		"playbackLogsCreated": count,
		"startDate":           startDate,
		"endDate":             endDate,
	})
}

func (h *AdvertisementHandler) CreateAdvertisement(w http.ResponseWriter, r *http.Request) {
	var req models.AdvertisementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request data")
		return
	}

	// Debug logging
	log.Printf("=== Received Advertisement Create Request ===")
	log.Printf("Advertisement Name: %s", req.AdvertisementName)
	log.Printf("Schedule Type: %s", req.ScheduleType)
	log.Printf("Frequency: %v", req.Frequency)
	log.Printf("Recurrence Interval: %v", req.RecurrenceInterval)
	log.Printf("Occurs Every Interval: %v", req.OccursEveryInterval)
	log.Printf("Start Date: %v", req.StartDate)
	log.Printf("End Date: %v", req.EndDate)
	log.Printf("Start Time: %v", req.StartTime)
	log.Printf("End Time: %v", req.EndTime)
	log.Printf("Occurs Once At: %v", req.OccursOnceAt)
	log.Printf("Max Idle Loop Duration: %v", req.MaxIdleLoopDuration)
	log.Printf("==============================================")

	ad, err := h.service.CreateAdvertisement(&req)
	if err != nil {
		log.Printf("Error creating advertisement: %v", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusCreated, ad)
}

func (h *AdvertisementHandler) GetAllAdvertisements(w http.ResponseWriter, r *http.Request) {
	log.Println("=== GetAllAdvertisements called ===")
	payload, err := h.cache.getOrSet("ads:all", func() ([]byte, error) {
		ads, err := h.service.GetAllAdvertisements()
		if err != nil {
			return nil, err
		}
		log.Printf("Successfully fetched %d advertisements", len(ads))
		return json.Marshal(ads)
	})
	if err != nil {
		log.Printf("Error fetching advertisements: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch advertisements")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

func (h *AdvertisementHandler) GetTVAdsManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loungeGroup := strings.TrimSpace(vars["loungeGroup"])
	if loungeGroup == "" {
		respondWithError(w, http.StatusBadRequest, "loungeGroup is required")
		return
	}

	baseURL := requestBaseURL(r)
	cacheKey := "ads:tv:" + strings.ToLower(loungeGroup) + ":" + baseURL
	payload, err := h.cache.getOrSet(cacheKey, func() ([]byte, error) {
		items, err := h.service.GetTVAdsManifest(loungeGroup, baseURL)
		if err != nil {
			return nil, err
		}
		return json.Marshal(items)
	})
	if err != nil {
		log.Printf("Error building tv ads manifest: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch TV ads manifest")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

func requestBaseURL(r *http.Request) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func (h *AdvertisementHandler) GetAdvertisementByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ad, err := h.service.GetAdvertisementByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Advertisement not found")
		return
	}
	respondWithJSON(w, http.StatusOK, ad)
}

func (h *AdvertisementHandler) UpdateAdvertisement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req models.AdvertisementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request data")
		return
	}

	err := h.service.UpdateAdvertisement(id, &req)
	if err != nil {
		log.Printf("Error updating advertisement: %v", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Advertisement updated successfully"})
}

func (h *AdvertisementHandler) DeleteAdvertisement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.service.DeleteAdvertisement(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Advertisement deleted successfully"})
}

// Advertisement Group handlers
func (h *AdvertisementHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req models.AdvertisementGroupCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Log the request for debugging
	log.Printf("Creating group: %s with %d lounges", req.GroupName, len(req.Lounges))

	group, err := h.service.CreateGroup(&req)
	if err != nil {
		log.Printf("Error creating group: %v", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusCreated, group)
}

func (h *AdvertisementHandler) GetAllGroups(w http.ResponseWriter, r *http.Request) {
	payload, err := h.cache.getOrSet("ads:groups:all", func() ([]byte, error) {
		groups, err := h.service.GetAllGroups()
		if err != nil {
			return nil, err
		}
		return json.Marshal(groups)
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch groups")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

func (h *AdvertisementHandler) GetGroupByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	group, err := h.service.GetGroupByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Group not found")
		return
	}
	respondWithJSON(w, http.StatusOK, group)
}

func (h *AdvertisementHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req models.AdvertisementGroupCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.service.UpdateGroup(id, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Group updated successfully"})
}

func (h *AdvertisementHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.service.DeleteGroup(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.cache.clear()

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
