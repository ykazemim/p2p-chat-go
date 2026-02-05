package stun

import (
	"encoding/json"
	"net/http"

	"p2p-chat/internal/protocol"
)

type Handlers struct {
	store Store
}

func NewHandlers(store Store) *Handlers {
	return &Handlers{store: store}
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req protocol.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, protocol.RegisterResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Username == "" || req.IP == "" || req.Port <= 0 {
		writeJSON(w, http.StatusBadRequest, protocol.RegisterResponse{
			Success: false,
			Message: "Missing required fields",
		})
		return
	}

	peer := protocol.PeerInfo{
		Username: req.Username,
		IP:       req.IP,
		Port:     req.Port,
	}

	if err := h.store.Set(r.Context(), peer); err != nil {
		writeJSON(w, http.StatusInternalServerError, protocol.RegisterResponse{
			Success: false,
			Message: "Failed to store peer info",
		})
		return
	}

	writeJSON(w, http.StatusCreated, protocol.RegisterResponse{
		Success: true,
		Message: "Peer registered successfully",
	})
}

func (h *Handlers) GetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peers, err := h.store.GetAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch peers"})
		return
	}

	writeJSON(w, http.StatusOK, protocol.PeersResponse{Peers: peers})
}

func (h *Handlers) GetPeerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Username parameter required"})
		return
	}

	peer, found, err := h.store.Get(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch peer info"})
		return
	}

	if !found {
		writeJSON(w, http.StatusNotFound, protocol.PeerInfoResponse{Found: false})
		return
	}

	writeJSON(w, http.StatusOK, protocol.PeerInfoResponse{Found: true, Peer: peer})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
