package main

import (
	"flag"
	"log"

	"p2p-chat/internal/ui"
)

func main() {
	stunURL := flag.String("stun", "http://localhost:8080", "STUN server URL")
	localIP := flag.String("ip", "127.0.0.1", "Local IP address to advertise")
	useTLS := flag.Bool("tls", true, "Use TLS for peer connections")
	flag.Parse()

	log.Printf("Starting P2P Chat client...")
	log.Printf("STUN Server: %s", *stunURL)
	log.Printf("Local IP: %s", *localIP)
	log.Printf("TLS: %v", *useTLS)

	app := ui.New(*stunURL, *localIP, *useTLS)
	app.Run()
}
