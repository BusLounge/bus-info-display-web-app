package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tv-sync-agent-go/internal/models"
	"tv-sync-agent-go/internal/sync"
)

type Server struct {
	agent *sync.Agent
	port  int
}

func New(agent *sync.Agent, port int) *Server {
	return &Server{agent: agent, port: port}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/local/status", s.handleStatus)
	mux.HandleFunc("/local/language", s.handleLanguage)
	mux.HandleFunc("/local/display-mode", s.handleDisplayMode)
	mux.HandleFunc("/local/layout-mode", s.handleLayoutMode)
	mux.HandleFunc("/local/broadcasts", s.handleBroadcasts)
	mux.HandleFunc("/local/broadcasts/sync", s.handleBroadcastsSync)
	mux.HandleFunc("/local/broadcasts-enabled", s.handleBroadcastsEnabled)
	mux.HandleFunc("/local/lounge-ads", s.handleLoungeAds)
	mux.HandleFunc("/local/lounge-ads/", s.handleLoungeAdByID)
	mux.HandleFunc("/local/lounge-ads/slots", s.handleLoungeAdSlots)
	mux.HandleFunc("/local/schedule", s.handleSchedule)
	mux.HandleFunc("/local/ads", s.handleAds)
	mux.HandleFunc("/local/media/", s.handleMedia)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	addr := ":" + itoa(s.port)
	log.Printf("local bridge listening on %s", addr)
	return http.ListenAndServe(addr, handler)
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, s.agent.Status())
}

type languageUpdateRequest struct {
	Language string `json:"language"`
}

type displayModeUpdateRequest struct {
	DisplayMode string `json:"displayMode"`
}

type layoutModeUpdateRequest struct {
	LayoutMode string `json:"layoutMode"`
}

type broadcastsEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req languageUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	language, err := s.agent.SetLanguage(req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"language": language,
		"message":  "language updated",
	})
}

func (s *Server) handleDisplayMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req displayModeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	displayMode, err := s.agent.SetDisplayMode(req.DisplayMode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"displayMode": displayMode,
		"message":     "display mode updated",
	})
}

func (s *Server) handleLayoutMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req layoutModeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	layoutMode, err := s.agent.SetLayoutMode(req.LayoutMode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"layoutMode": layoutMode,
		"message":    "layout mode updated",
	})
}

func (s *Server) handleBroadcasts(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(s.agent.StoreDir(), "broadcasts.json")
	streamFile(w, r, p)
}

func (s *Server) handleBroadcastsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.agent.SyncBroadcasts(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":           "broadcasts synced",
		"broadcastCount":    s.agent.Status().BroadcastCount,
		"lastBroadcastSync": s.agent.Status().LastBroadcastSync,
	})
}

func (s *Server) handleLoungeAds(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleCreateLocalLoungeAd(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p := filepath.Join(s.agent.StoreDir(), "lounge-ads.json")
	streamFile(w, r, p)
}

func (s *Server) handleLoungeAdByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/local/lounge-ads/"))
	if id == "" || id == "slots" || strings.Contains(id, "/") {
		respondError(w, http.StatusNotFound, "lounge ad id is invalid")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleUpdateLocalLoungeAd(w, r, id)
	case http.MethodDelete:
		s.handleDeleteLocalLoungeAd(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleLoungeAdSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := s.agent.GetLocalLoungeAdSlotSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) handleCreateLocalLoungeAd(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		s.handleCreateLocalLoungeAdUpload(w, r)
		return
	}

	var req syncLoungeAdCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	ad, err := s.agent.CreateLocalLoungeAd(context.Background(), req.toModel())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, ad)
}

func (s *Server) handleCreateLocalLoungeAdUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form payload")
		return
	}

	name := strings.TrimSpace(r.FormValue("advertisementName"))
	if name == "" {
		respondError(w, http.StatusBadRequest, "advertisementName is required")
		return
	}

	file, header, err := r.FormFile("mediaFile")
	if err != nil {
		respondError(w, http.StatusBadRequest, "mediaFile is required")
		return
	}
	defer file.Close()

	req := models.LoungeAdCreateRequest{
		AdvertisementName: name,
		IsActive:          boolPtrFromForm(r.FormValue("isActive"), true),
		IsDefaultForAll:   boolPtrFromForm(r.FormValue("isDefaultForAll"), false),
	}

	ad, err := s.agent.CreateLocalLoungeAdFromUpload(req, header.Filename, file)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, ad)
}

func (s *Server) handleUpdateLocalLoungeAd(w http.ResponseWriter, r *http.Request, id string) {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		s.handleUpdateLocalLoungeAdUpload(w, r, id)
		return
	}

	var req syncLoungeAdUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	ad, err := s.agent.UpdateLocalLoungeAd(id, req.toModel())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, ad)
}

func (s *Server) handleUpdateLocalLoungeAdUpload(w http.ResponseWriter, r *http.Request, id string) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form payload")
		return
	}

	name := strings.TrimSpace(r.FormValue("advertisementName"))
	isActive := boolPtrFromForm(r.FormValue("isActive"), true)
	isDefaultForAll := boolPtrFromForm(r.FormValue("isDefaultForAll"), false)

	var fileReader io.Reader
	var fileName string
	file, header, err := r.FormFile("mediaFile")
	if err == nil {
		defer file.Close()
		fileReader = file
		fileName = header.Filename
	}

	ad, err := s.agent.UpdateLocalLoungeAd(id, models.LoungeAdUpdateRequest{
		AdvertisementName: name,
		IsActive:          isActive,
		IsDefaultForAll:   isDefaultForAll,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if fileReader != nil {
		updated, updateErr := s.agent.UpdateLocalLoungeAdMedia(id, fileName, fileReader)
		if updateErr != nil {
			respondError(w, http.StatusBadRequest, updateErr.Error())
			return
		}
		ad = updated
	}

	respondJSON(w, http.StatusOK, ad)
}

func (s *Server) handleDeleteLocalLoungeAd(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.agent.DeleteLocalLoungeAd(id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "lounge ad deleted"})
}

type syncLoungeAdCreateRequest struct {
	AdvertisementName string `json:"advertisementName"`
	MediaURL          string `json:"mediaUrl"`
	MediaType         string `json:"mediaType"`
	DurationSeconds   int    `json:"durationSeconds"`
	Priority          string `json:"priority"`
	IsActive          *bool  `json:"isActive"`
	IsDefaultForAll   *bool  `json:"isDefaultForAll"`
}

type syncLoungeAdUpdateRequest struct {
	AdvertisementName string `json:"advertisementName"`
	MediaURL          string `json:"mediaUrl"`
	MediaType         string `json:"mediaType"`
	IsActive          *bool  `json:"isActive"`
	IsDefaultForAll   *bool  `json:"isDefaultForAll"`
}

func (r syncLoungeAdCreateRequest) toModel() models.LoungeAdCreateRequest {
	return models.LoungeAdCreateRequest{
		AdvertisementName: r.AdvertisementName,
		MediaURL:          r.MediaURL,
		MediaType:         r.MediaType,
		DurationSeconds:   r.DurationSeconds,
		Priority:          r.Priority,
		IsActive:          r.IsActive,
		IsDefaultForAll:   r.IsDefaultForAll,
	}
}

func (r syncLoungeAdUpdateRequest) toModel() models.LoungeAdUpdateRequest {
	return models.LoungeAdUpdateRequest{
		AdvertisementName: r.AdvertisementName,
		MediaURL:          r.MediaURL,
		MediaType:         r.MediaType,
		IsActive:          r.IsActive,
		IsDefaultForAll:   r.IsDefaultForAll,
	}
}

func boolPtrFromForm(value string, defaultValue bool) *bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		v := defaultValue
		return &v
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		v := defaultValue
		return &v
	}
	return &parsed
}

func (s *Server) handleBroadcastsEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, map[string]bool{"enabled": s.agent.Status().BroadcastsEnabled})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req broadcastsEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	enabled := s.agent.SetBroadcastsEnabled(req.Enabled)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled": enabled,
		"message": "broadcast display setting updated",
	})
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if !s.agent.ScheduleEnabled() {
		http.Error(w, "schedule sync is disabled for this TV", http.StatusNotFound)
		return
	}
	p := filepath.Join(s.agent.StoreDir(), "schedule.json")
	streamFile(w, r, p)
}

func (s *Server) handleAds(w http.ResponseWriter, r *http.Request) {
	if !s.agent.AdsEnabled() {
		http.Error(w, "ads sync is disabled for this TV", http.StatusNotFound)
		return
	}
	p := filepath.Join(s.agent.StoreDir(), "ads-manifest.json")
	streamFile(w, r, p)
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	if !s.agent.AdsEnabled() {
		http.Error(w, "ads sync is disabled for this TV", http.StatusNotFound)
		return
	}
	name := filepath.Base(r.URL.Path)
	if name == "." || name == "/" || name == "" {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(s.agent.StoreDir(), "media", name)
	streamFile(w, r, p)
}

func streamFile(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	buf := make([]byte, 0, 11)
	for v > 0 {
		buf = append(buf, byte('0'+(v%10)))
		v /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
