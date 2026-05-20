package search

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/internal/permission"
	"gorm.io/gorm"
)

type Handler struct {
	Service           *Service
	DB                *gorm.DB
	PermissionService *permission.Service
}

func NewHandler(service *Service, db *gorm.DB, permService *permission.Service) *Handler {
	return &Handler{Service: service, DB: db, PermissionService: permService}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	wsID, err := strconv.ParseUint(r.URL.Query().Get("workspace_id"), 10, 64)
	if err != nil || wsID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_id is required"})
		return
	}

	results, err := h.Service.Search(uint(wsID), q)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "search unavailable"})
		return
	}

	// Filter results by user's page-level permissions
	if h.PermissionService != nil {
		claims := auth.GetUserFromContext(r.Context())
		if claims != nil {
			results = h.filterByAccess(results, claims.UserID, uint(wsID))
		}
	}

	json.NewEncoder(w).Encode(results)
}

// filterByAccess removes results the user cannot view.
func (h *Handler) filterByAccess(results *SearchResults, userID, workspaceID uint) *SearchResults {
	// Workspace owner sees everything
	if h.isWorkspaceOwner(userID, workspaceID) {
		return results
	}

	filtered := &SearchResults{}

	// Collect page IDs from results
	type pageAccess struct {
		pageID uint
		ok     bool
	}
	accessCache := map[uint]bool{}

	checkAccess := func(pageID uint) bool {
		if v, ok := accessCache[pageID]; ok {
			return v
		}
		ok := h.PermissionService.CanView(userID, pageID)
		accessCache[pageID] = ok
		return ok
	}

	for _, p := range results.Pages {
		if checkAccess(p.ID) {
			filtered.Pages = append(filtered.Pages, p)
		}
	}

	for _, b := range results.Blocks {
		if checkAccess(b.PageID) {
			filtered.Blocks = append(filtered.Blocks, b)
		}
	}

	// Records need database → page resolution
	for _, rec := range results.Records {
		pageID := h.resolveRecordPageID(rec.DatabaseID)
		if pageID == 0 || checkAccess(pageID) {
			filtered.Records = append(filtered.Records, rec)
		}
	}

	return filtered
}

// isWorkspaceOwner returns true if the user owns the workspace.
func (h *Handler) isWorkspaceOwner(userID, workspaceID uint) bool {
	var role string
	h.DB.Table("workspace_members").
		Select("role").
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Scan(&role)
	return role == "owner"
}

// resolveRecordPageID returns the container page ID for a database.
func (h *Handler) resolveRecordPageID(databaseID uint) uint {
	var pageID uint
	h.DB.Table("databases").
		Select("page_id").
		Where("id = ?", databaseID).
		Scan(&pageID)
	return pageID
}

func (h *Handler) Reindex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.Service.ReindexAll(h.DB); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
