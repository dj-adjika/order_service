package handler

import (
	"encoding/json"
	"net/http"
	"order-service/internal/cache"
	"order-service/internal/database"
	"order-service/internal/models"

	"github.com/gorilla/mux"
)

type Handler struct {
	cache *cache.Cache
	db    *database.Postgres
}

func (h *Handler) Debug(w http.ResponseWriter, r *http.Request) {
    orders := h.cache.GetAll()
    respondWithJSON(w, http.StatusOK, map[string]interface{}{
        "cache_size": len(orders),
        "order_ids":  getKeys(orders),
    })
}

func getKeys(m map[string]*models.Order) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

func New(cache *cache.Cache, db *database.Postgres) *Handler {
	return &Handler{
		cache: cache,
		db:    db,
	}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderUID := vars["id"]

	if order, exists := h.cache.Get(orderUID); exists {
		respondWithJSON(w, http.StatusOK, order)
		return
	}

	order, err := h.db.GetOrder(orderUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	h.cache.Set(order)

	respondWithJSON(w, http.StatusOK, order)
}

func (h *Handler) ServeHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/static/index.html")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}