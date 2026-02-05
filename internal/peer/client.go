package peer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"p2p-chat/internal/protocol"
)

type STUNClient struct {
	serverURL  string
	httpClient *http.Client
}

func NewSTUNClient(serverURL string) *STUNClient {
	return &STUNClient{
		serverURL:  serverURL,
		httpClient: &http.Client{},
	}
}

func (c *STUNClient) Register(username, ip string, port int) error {
	req := protocol.RegisterRequest{
		Username: username,
		IP:       ip,
		Port:     port,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.serverURL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result protocol.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("registration failed: %s", result.Message)
	}

	return nil
}

func (c *STUNClient) GetPeers() ([]protocol.PeerInfo, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/peers")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result protocol.PeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Peers, nil
}

func (c *STUNClient) GetPeerInfo(username string) (protocol.PeerInfo, bool, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/peerinfo?username=" + username)
	if err != nil {
		return protocol.PeerInfo{}, false, err
	}
	defer resp.Body.Close()

	var result protocol.PeerInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return protocol.PeerInfo{}, false, err
	}

	return result.Peer, result.Found, nil
}
