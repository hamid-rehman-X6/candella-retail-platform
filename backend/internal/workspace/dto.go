package workspace

import (
	"encoding/json"
	"net/http"
	"time"
)

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(dst); err != nil {
		return ErrInvalidInput
	}
	return nil
}

type createRequest struct {
	Name         string `json:"name"`
	IndustryType string `json:"industryType"`
	CurrencyCode string `json:"currencyCode"`
	Timezone     string `json:"timezone"`
}

type workspaceResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	IndustryType string    `json:"industryType"`
	Status       string    `json:"status"`
	CurrencyCode string    `json:"currencyCode"`
	Timezone     string    `json:"timezone"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toResponse(w *Workspace) workspaceResponse {
	return workspaceResponse{
		ID:           w.ID,
		Name:         w.Name,
		Slug:         w.Slug,
		IndustryType: w.IndustryType,
		Status:       w.Status,
		CurrencyCode: w.CurrencyCode,
		Timezone:     w.Timezone,
		Role:         w.Role,
		CreatedAt:    w.CreatedAt,
	}
}
