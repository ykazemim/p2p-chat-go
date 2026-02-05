package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"p2p-chat/internal/stun"
)

func main() {
	addr := flag.String("addr", ":8080", "Server address")
	redisAddr := flag.String("redis", "", "Redis address (e.g., localhost:6379)")
	flag.Parse()

	var store stun.Store
	var err error

	if *redisAddr != "" {
		store, err = stun.NewRedisStore(*redisAddr)
		if err != nil {
			log.Printf("Failed to connect to Redis: %v, falling back to memory store", err)
			store = stun.NewMemoryStore()
		} else {
			log.Printf("Using Redis store at %s", *redisAddr)
		}
	} else {
		store = stun.NewMemoryStore()
		log.Println("Using in-memory store")
	}

	server := stun.NewServer(*addr, store)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
