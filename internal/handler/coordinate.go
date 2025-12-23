package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"volumetric-backend/internal/model"
)

type CoordinateHandler struct {
	DB *sql.DB
}

func (h *CoordinateHandler) CreateCoordinate(w http.ResponseWriter, r *http.Request) {
	var c model.Coordinate

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO coordinates (x, y, z) VALUES ($1, $2, $3)`
	_, err := h.DB.Exec(query, c.X, c.Y, c.Z)
	if err != nil {
		http.Error(w, "failed to insert data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "coordinate saved",
	})
}
