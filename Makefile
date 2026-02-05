.PHONY: build stun peer docker-up docker-down clean

build: stun peer

stun:
	go build -o bin/stun-server ./cmd/stun-server

peer:
	go build -o bin/peer ./cmd/peer

run-stun:
	go run ./cmd/stun-server -addr :8080

run-peer:
	go run ./cmd/peer -stun http://localhost:8080 -ip 127.0.0.1

docker-up:
	cd docker && docker-compose up -d

docker-down:
	cd docker && docker-compose down

docker-logs:
	cd docker && docker-compose logs -f

clean:
	rm -rf bin/
	cd docker && docker-compose down -v --rmi local 2>/dev/null || true
