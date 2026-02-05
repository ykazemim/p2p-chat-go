package stun

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/redis/go-redis/v9"
	"p2p-chat/internal/protocol"
)

type Store interface {
	Set(ctx context.Context, peer protocol.PeerInfo) error
	Get(ctx context.Context, username string) (protocol.PeerInfo, bool, error)
	GetAll(ctx context.Context) ([]protocol.PeerInfo, error)
	Delete(ctx context.Context, username string) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	peers map[string]protocol.PeerInfo
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		peers: make(map[string]protocol.PeerInfo),
	}
}

func (m *MemoryStore) Set(_ context.Context, peer protocol.PeerInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[peer.Username] = peer
	return nil
}

func (m *MemoryStore) Get(_ context.Context, username string) (protocol.PeerInfo, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	peer, ok := m.peers[username]
	return peer, ok, nil
}

func (m *MemoryStore) GetAll(_ context.Context) ([]protocol.PeerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	peers := make([]protocol.PeerInfo, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	return peers, nil
}

func (m *MemoryStore) Delete(_ context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, username)
	return nil
}

type RedisStore struct {
	client *redis.Client
	prefix string
}

func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisStore{
		client: client,
		prefix: "peer:",
	}, nil
}

func (r *RedisStore) Set(ctx context.Context, peer protocol.PeerInfo) error {
	data, err := json.Marshal(peer)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.prefix+peer.Username, data, 0).Err()
}

func (r *RedisStore) Get(ctx context.Context, username string) (protocol.PeerInfo, bool, error) {
	data, err := r.client.Get(ctx, r.prefix+username).Bytes()
	if err == redis.Nil {
		return protocol.PeerInfo{}, false, nil
	}
	if err != nil {
		return protocol.PeerInfo{}, false, err
	}
	var peer protocol.PeerInfo
	if err := json.Unmarshal(data, &peer); err != nil {
		return protocol.PeerInfo{}, false, err
	}
	return peer, true, nil
}

func (r *RedisStore) GetAll(ctx context.Context) ([]protocol.PeerInfo, error) {
	keys, err := r.client.Keys(ctx, r.prefix+"*").Result()
	if err != nil {
		return nil, err
	}
	peers := make([]protocol.PeerInfo, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var peer protocol.PeerInfo
		if err := json.Unmarshal(data, &peer); err != nil {
			continue
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

func (r *RedisStore) Delete(ctx context.Context, username string) error {
	return r.client.Del(ctx, r.prefix+username).Err()
}
