package permission

import (
	"net/http"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/go-chi/chi/v5"
)

// PageAccess creates middleware that checks if the current user can access the
// page identified by the "id" URL parameter.
func PageAccess(service *Service, minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.GetUserFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			pageIDStr := chi.URLParam(r, "id")
			if pageIDStr == "" {
				pageIDStr = chi.URLParam(r, "pageId")
			}
			if pageIDStr == "" {
				// No page ID in URL — skip permission check
				next.ServeHTTP(w, r)
				return
			}

			pageID, err := parseUint(pageIDStr)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			role := service.CheckAccess(claims.UserID, pageID)

			switch minRole {
			case "editor":
				if role != "editor" {
					http.Error(w, `{"error":"forbidden: editor access required"}`, http.StatusForbidden)
					return
				}
			case "commenter":
				if role != "editor" && role != "commenter" {
					http.Error(w, `{"error":"forbidden: commenter access required"}`, http.StatusForbidden)
					return
				}
			default: // "viewer"
				if role == "" {
					http.Error(w, `{"error":"forbidden: viewer access required"}`, http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseUint(s string) (uint, error) {
	var n uint
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, http.ErrAbortHandler
		}
		n = n*10 + uint(c-'0')
	}
	return n, nil
}
