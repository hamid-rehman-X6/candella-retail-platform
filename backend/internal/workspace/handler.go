package workspace

import (
	"errors"
	"net/http"

	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/pkg/response"
)

// Handler adapts HTTP requests to the workspace Service. It relies on the auth
// middleware having placed the authenticated user in the request context.
type Handler struct {
	svc *Service
}

// NewHandler builds the workspace HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var de *Error
	if errors.As(err, &de) {
		response.Error(w, de.Status, de.Code, de.Message)
		return
	}
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

// POST /workspaces — create a workspace owned by the current user.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	var req createRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	ws, err := h.svc.Create(r.Context(), user.ID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"workspace": toResponse(ws)})
}

// GET /workspaces — list the current user's workspaces.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	list, err := h.svc.List(r.Context(), user.ID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	out := make([]workspaceResponse, 0, len(list))
	for i := range list {
		out = append(out, toResponse(&list[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"workspaces": out})
}
