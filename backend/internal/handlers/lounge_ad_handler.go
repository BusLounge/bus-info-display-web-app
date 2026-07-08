package handlers

import (
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type LoungeAdHandler struct {
	service *services.LoungeAdService
}

func NewLoungeAdHandler(service *services.LoungeAdService) *LoungeAdHandler {
	return &LoungeAdHandler{service: service}
}

func (h *LoungeAdHandler) GetAll(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.GetAll()
	if err != nil {
		loungeAdRespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusOK, items)
}

func (h *LoungeAdHandler) GetForLounge(w http.ResponseWriter, r *http.Request) {
	loungeID := strings.TrimSpace(mux.Vars(r)["loungeId"])
	if loungeID == "" {
		loungeAdRespondWithError(w, http.StatusBadRequest, "loungeId is required")
		return
	}
	items, err := h.service.GetForLounge(loungeID)
	if err != nil {
		loungeAdRespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusOK, items)
}

func (h *LoungeAdHandler) GetSlotSummary(w http.ResponseWriter, r *http.Request) {
	loungeID := strings.TrimSpace(mux.Vars(r)["loungeId"])
	if loungeID == "" {
		loungeAdRespondWithError(w, http.StatusBadRequest, "loungeId is required")
		return
	}
	summary, err := h.service.GetSlotSummary(loungeID)
	if err != nil {
		loungeAdRespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusOK, summary)
}

func (h *LoungeAdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.LoungeAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		loungeAdRespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.service.Create(&req)
	if err != nil {
		loungeAdRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusCreated, item)
}

func (h *LoungeAdHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		loungeAdRespondWithError(w, http.StatusBadRequest, "id is required")
		return
	}
	var req models.LoungeAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		loungeAdRespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.service.Update(id, &req)
	if err != nil {
		loungeAdRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusOK, item)
}

func (h *LoungeAdHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		loungeAdRespondWithError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := h.service.Delete(id); err != nil {
		loungeAdRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	loungeAdRespondWithJSON(w, http.StatusOK, map[string]string{"message": "lounge ad deleted"})
}

func loungeAdRespondWithError(w http.ResponseWriter, code int, message string) {
	loungeAdRespondWithJSON(w, code, map[string]string{"error": message})
}

func loungeAdRespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
