package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"asset-service/internal/models"
	"asset-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type AssetHandler struct {
	repo *repository.AssetRepository
}

func NewAssetHandler(repo *repository.AssetRepository) *AssetHandler {
	return &AssetHandler{repo: repo}
}

func (h *AssetHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "asset-service",
		"status":  "healthy",
	})
}

func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var asset models.Asset

	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if asset.ID == "" || asset.Name == "" || asset.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "id, name and type are required",
		})
		return
	}

	if asset.Status == "" {
		asset.Status = "ACTIVE"
	}

	if asset.Location == "" {
		asset.Location = "UNKNOWN"
	}

	if err := h.repo.Create(r.Context(), &asset); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, assets)
}

func (h *AssetHandler) GetAssetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	asset, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "asset not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var asset models.Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if asset.Name == "" || asset.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name and type are required",
		})
		return
	}

	if asset.Status == "" {
		asset.Status = "ACTIVE"
	}

	if asset.Location == "" {
		asset.Location = "UNKNOWN"
	}

	if err := h.repo.Update(r.Context(), id, &asset); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "asset not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rowsAffected, err := h.repo.Delete(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if rowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "asset not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "asset deleted successfully",
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
