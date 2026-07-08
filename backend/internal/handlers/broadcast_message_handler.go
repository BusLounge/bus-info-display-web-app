package handlers

import (
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type BroadcastMessageHandler struct {
	service *services.BroadcastMessageService
}

func NewBroadcastMessageHandler(service *services.BroadcastMessageService) *BroadcastMessageHandler {
	return &BroadcastMessageHandler{service: service}
}

func (h *BroadcastMessageHandler) GetAll(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.GetAll()
	if err != nil {
		broadcastRespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	broadcastRespondWithJSON(w, http.StatusOK, items)
}

func (h *BroadcastMessageHandler) GetActiveForTV(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.GetActiveForTV()
	if err != nil {
		broadcastRespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	broadcastRespondWithJSON(w, http.StatusOK, items)
}

func (h *BroadcastMessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.BroadcastMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		broadcastRespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Create(&req)
	if err != nil {
		broadcastRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	broadcastRespondWithJSON(w, http.StatusCreated, item)
}

func (h *BroadcastMessageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		broadcastRespondWithError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req models.BroadcastMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		broadcastRespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Update(id, &req)
	if err != nil {
		broadcastRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	broadcastRespondWithJSON(w, http.StatusOK, item)
}

func (h *BroadcastMessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		broadcastRespondWithError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := h.service.Delete(id); err != nil {
		broadcastRespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	broadcastRespondWithJSON(w, http.StatusOK, map[string]string{"message": "broadcast message deleted"})
}

func broadcastRespondWithError(w http.ResponseWriter, code int, message string) {
	broadcastRespondWithJSON(w, code, map[string]string{"error": message})
}

func broadcastRespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
