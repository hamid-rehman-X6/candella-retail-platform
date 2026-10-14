package catalog

import (
	"encoding/csv"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/response"
)

type ImportHandler struct {
	svc     *ImportService
	auditor *audit.Recorder
	log     *slog.Logger
}

func NewImportHandler(svc *ImportService, auditor *audit.Recorder, log *slog.Logger) *ImportHandler {
	return &ImportHandler{svc: svc, auditor: auditor, log: log}
}

func (h *ImportHandler) writeErr(w http.ResponseWriter, err error) {
	if typed, ok := err.(*Error); ok {
		response.Error(w, typed.Status, typed.Code, typed.Message)
		return
	}
	h.log.Error("catalog import failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Import failed. Please try again.")
}

func importScope(r *http.Request) (string, string, bool) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	return t.TenantID, u.ID, t.Role == "owner" || t.Role == "admin"
}

func readImportRequest(w http.ResponseWriter, r *http.Request) (string, []ImportRow, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return "", nil, ErrImportFile
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", nil, ErrImportFile
	}
	defer file.Close()
	if header.Size > 5<<20 {
		return "", nil, ErrImportFile
	}
	rows, err := parseImportCSV(file)
	return r.FormValue("mode"), rows, err
}

func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	tenantID, _, editable := importScope(r)
	if !editable {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin to import products.")
		return
	}
	mode, rows, err := readImportRequest(w, r)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	preview, err := h.svc.Preview(r.Context(), tenantID, mode, rows)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"preview": preview})
}

func (h *ImportHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, editable := importScope(r)
	if !editable {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin to import products.")
		return
	}
	mode, rows, err := readImportRequest(w, r)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	result, err := h.svc.Confirm(r.Context(), tenantID, userID, mode, r.FormValue("idempotencyKey"), r.FormValue("previewHash"), rows)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if !result.Replayed {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &userID, Action: "catalog.imported", EntityType: "catalog_import_batch", EntityID: r.FormValue("idempotencyKey"), Metadata: map[string]any{"mode": mode, "created": result.Created, "updated": result.Updated}})
	}
	response.JSON(w, http.StatusOK, map[string]any{"result": result})
}

func csvResponse(w http.ResponseWriter, name string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Header().Set("Cache-Control", "no-store")
	writer := csv.NewWriter(w)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
}

func (h *ImportHandler) Template(w http.ResponseWriter, r *http.Request) {
	_, _, editable := importScope(r)
	if !editable {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin to import products.")
		return
	}
	csvResponse(w, "candella-products-template.csv", [][]string{importColumns})
}

func (h *ImportHandler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, _, editable := importScope(r)
	if !editable {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin to export products.")
		return
	}
	rows, err := h.svc.Export(r.Context(), tenantID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	csvResponse(w, "candella-products.csv", rows)
}

func ImportRoutes(r chi.Router, h *ImportHandler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/products/import/template.csv", h.Template)
		g.Get("/products/export.csv", h.Export)
		g.Post("/products/import/preview", h.Preview)
		g.Post("/products/import/confirm", h.Confirm)
	})
}
