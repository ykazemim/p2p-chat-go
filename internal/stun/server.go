package stun

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	handlers   *Handlers
}

func NewServer(addr string, store Store) *Server {
	handlers := NewHandlers(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", handlers.Register)
	mux.HandleFunc("/peers", handlers.GetPeers)
	mux.HandleFunc("/peerinfo", handlers.GetPeerInfo)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		handlers: handlers,
	}
}

func (s *Server) Start() error {
	log.Printf("STUN server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("\nShutting down STUN server...")
	return s.httpServer.Shutdown(ctx)
}
