package health

import (
	"net/http"

	"candella-ecosystem/backend/pkg/response"
)

type Status struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Check(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, Status{
		Status:  "ok",
		Service: "candella-backend",
	})
}
