package workspace

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.con/falasefemi2/taskflow/api/db/generated"
	"github.con/falasefemi2/taskflow/api/internal/config"
	"github.con/falasefemi2/taskflow/api/internal/middleware"
	"github.con/falasefemi2/taskflow/api/internal/utils"
)

type Handler struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewHandler(queries *db.Queries, cfg *config.Config) *Handler {
	return &Handler{queries: queries, cfg: cfg}
}

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateWorkspaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"` // active | archived
}

type WorkspaceResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaginatedWorkspacesResponse struct {
	Data   []WorkspaceResponse `json:"data"`
	Limit  int32               `json:"limit"`
	Offset int32               `json:"offset"`
}

func toResponse(ws db.Workspace) WorkspaceResponse {
	return WorkspaceResponse{
		ID:          ws.ID,
		Name:        ws.Name,
		Slug:        ws.Slug,
		Description: ws.Description.String,
		OwnerID:     ws.OwnerID,
		Status:      ws.Status,
		CreatedAt:   ws.CreatedAt.Time,
		UpdatedAt:   ws.UpdatedAt.Time,
	}
}

func getPagination(r *http.Request) (int32, int32) {
	const (
		defaultLimit = 20
		maxLimit     = 100
	)

	limit := defaultLimit
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > maxLimit {
				parsed = maxLimit
			}
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return int32(limit), int32(offset)
}

func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkspaceRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" {
		utils.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	slug := utils.GenerateSlug(req.Name)

	ws, err := h.queries.CreateWorkspace(r.Context(), db.CreateWorkspaceParams{
		Name: req.Name,
		Slug: slug,
		Description: pgtype.Text{
			String: req.Description,
			Valid:  req.Description != "",
		},
		OwnerID: ownerID,
		Status:  "active",
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	utils.JSON(w, http.StatusCreated, toResponse(ws))
}

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	limit, offset := getPagination(r)

	workspaces, err := h.queries.ListWorkspacesByOwnerID(
		r.Context(),
		db.ListWorkspacesByOwnerIDParams{
			OwnerID: ownerID,
			Limit:   limit,
			Offset:  offset,
		},
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to fetch workspaces")
		return
	}

	res := make([]WorkspaceResponse, len(workspaces))
	for i, ws := range workspaces {
		res[i] = toResponse(ws)
	}

	utils.JSON(w, http.StatusOK, PaginatedWorkspacesResponse{
		Data:   res,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}

	ws, err := h.queries.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	if ws.OwnerID != ownerID {
		utils.Error(w, http.StatusForbidden, "access denied")
		return
	}

	utils.JSON(w, http.StatusOK, toResponse(ws))
}

func (h *Handler) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}

	var req UpdateWorkspaceRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.queries.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	if existing.OwnerID != ownerID {
		utils.Error(w, http.StatusForbidden, "access denied")
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	description := existing.Description
	if req.Description != nil {
		description = pgtype.Text{
			String: *req.Description,
			Valid:  *req.Description != "",
		}
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	ws, err := h.queries.UpdateWorkspace(r.Context(), db.UpdateWorkspaceParams{
		ID:          workspaceID,
		Name:        name,
		Slug:        existing.Slug, // keep slug stable
		Description: description,
		OwnerID:     ownerID,
		Status:      status,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}

	utils.JSON(w, http.StatusOK, toResponse(ws))
}

func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}

	existing, err := h.queries.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	if existing.OwnerID != ownerID {
		utils.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.queries.DeleteWorkspace(r.Context(), workspaceID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
