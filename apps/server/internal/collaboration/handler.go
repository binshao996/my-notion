package collaboration

import (
	"log"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/internal/user"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler handles WebSocket upgrade requests for collaboration.
type Handler struct {
	Hub      *Hub
	DocStore *DocStore
	Auth     *auth.Service
	UserSvc  *user.Service
}

func NewHandler(hub *Hub, docStore *DocStore, authSvc *auth.Service, userSvc *user.Service) *Handler {
	return &Handler{
		Hub:      hub,
		DocStore: docStore,
		Auth:     authSvc,
		UserSvc:  userSvc,
	}
}

// HandleWebSocket handles GET /ws/page/{id}?token=<jwt>
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	pageIDStr := chi.URLParam(r, "id")
	pageID, err := strconv.ParseUint(pageIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}

	// Validate JWT from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.Auth.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Get user name
	userName := ""
	u, err := h.UserSvc.FindByID(claims.UserID)
	if err == nil && u != nil {
		userName = u.Name
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("collab: ws upgrade failed: %v", err)
		return
	}

	client := NewClient(h.Hub, conn, uint(pageID), claims.UserID, userName)

	// Send sync_init with full document
	initMsg := h.DocStore.GetEncodedDocument(uint(pageID))
	if len(initMsg) > 0 {
		client.SendInit(initMsg)
	}

	log.Printf("collab: user %d (%s) joined page %d", claims.UserID, userName, pageID)
}
