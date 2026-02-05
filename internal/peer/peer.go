package peer

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"p2p-chat/internal/protocol"
)

const heartbeatInterval = 15 * time.Second

type MessageCallback func(from, content string)
type FileCallback func(from, filename string, size int64)
type FileCompleteCallback func(from, filename string)
type DisconnectCallback func(peerName string)
type ConnectionRequestCallback func(from string) bool

type Peer struct {
	username     string
	stunClient   *STUNClient
	tcpServer    *TCPServer
	tcpClient    *TCPClient
	useTLS       bool
	downloadDir  string
	mu           sync.Mutex
	connectedTo  string
	activeConn   net.Conn
	activeReader *bufio.Reader
	registeredIP string
	stopHeartbeat chan struct{}

	onMessage      MessageCallback
	onFile         FileCallback
	onFileComplete FileCompleteCallback
	onDisconnect   DisconnectCallback
	onConnRequest  ConnectionRequestCallback
}

func New(username, stunURL string, port int, useTLS bool) (*Peer, error) {
	p := &Peer{
		username:    username,
		stunClient:  NewSTUNClient(stunURL),
		useTLS:      useTLS,
		downloadDir: "./downloads",
	}

	os.MkdirAll(p.downloadDir, 0755)

	server, err := NewTCPServer(port, username, useTLS, p.handleConnectionRequest)
	if err != nil {
		return nil, err
	}
	p.tcpServer = server

	return p, nil
}

func (p *Peer) SetCallbacks(onMsg MessageCallback, onFile FileCallback, onFileComplete FileCompleteCallback, onDisc DisconnectCallback, onReq ConnectionRequestCallback) {
	p.onMessage = onMsg
	p.onFile = onFile
	p.onFileComplete = onFileComplete
	p.onDisconnect = onDisc
	p.onConnRequest = onReq
}

func (p *Peer) handleConnectionRequest(conn net.Conn, msg protocol.TCPMessage) bool {
	if p.onConnRequest != nil {
		accepted := p.onConnRequest(msg.From)
		if accepted {
			p.mu.Lock()
			p.connectedTo = msg.From
			p.activeConn = conn
			p.activeReader = bufio.NewReader(conn)
			p.mu.Unlock()
			go p.receiveLoop()
		}
		return accepted
	}
	return false
}

func (p *Peer) Start() error {
	go p.tcpServer.Accept()
	return nil
}

func (p *Peer) Register(ip string) error {
	p.registeredIP = ip
	if err := p.stunClient.Register(p.username, ip, p.tcpServer.Port()); err != nil {
		return err
	}
	p.startHeartbeat()
	return nil
}

func (p *Peer) startHeartbeat() {
	p.stopHeartbeat = make(chan struct{})
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.stunClient.Register(p.username, p.registeredIP, p.tcpServer.Port())
			case <-p.stopHeartbeat:
				return
			}
		}
	}()
}

func (p *Peer) GetPeers() ([]protocol.PeerInfo, error) {
	return p.stunClient.GetPeers()
}

func (p *Peer) GetPeerInfo(username string) (protocol.PeerInfo, bool, error) {
	return p.stunClient.GetPeerInfo(username)
}

func (p *Peer) Connect(peerInfo protocol.PeerInfo) error {
	client, err := NewTCPClient(p.username, p.useTLS)
	if err != nil {
		return err
	}

	if err := client.Connect(peerInfo.IP, peerInfo.Port); err != nil {
		return err
	}

	p.mu.Lock()
	p.tcpClient = client
	p.connectedTo = peerInfo.Username
	p.activeConn = client.GetConn()
	p.activeReader = bufio.NewReader(p.activeConn)
	p.mu.Unlock()

	go p.receiveLoop()

	return nil
}

func (p *Peer) receiveLoop() {
	for {
		p.mu.Lock()
		reader := p.activeReader
		conn := p.activeConn
		connectedTo := p.connectedTo
		p.mu.Unlock()

		if reader == nil || conn == nil {
			break
		}

		msg, err := ReceiveMessage(reader)
		if err != nil {
			if p.onDisconnect != nil {
				p.onDisconnect(connectedTo)
			}
			p.Disconnect()
			break
		}

		switch msg.Type {
		case protocol.TypeChatMessage:
			if p.onMessage != nil {
				p.onMessage(msg.From, msg.Content)
			}
		case protocol.TypeFileTransfer:
			if p.onFile != nil {
				p.onFile(msg.From, msg.Filename, msg.FileSize)
			}
			if err := ReceiveFile(reader, msg.Filename, msg.FileSize, p.downloadDir); err != nil {
				fmt.Printf("Error receiving file: %v\n", err)
			} else if p.onFileComplete != nil {
				p.onFileComplete(msg.From, msg.Filename)
			}
		case protocol.TypeDisconnect:
			if p.onDisconnect != nil {
				p.onDisconnect(msg.From)
			}
			p.Disconnect()
		}
	}
}

func (p *Peer) SendMessage(content string) error {
	p.mu.Lock()
	conn := p.activeConn
	p.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	msg := protocol.TCPMessage{
		Type:    protocol.TypeChatMessage,
		From:    p.username,
		Content: content,
	}

	return SendMessage(conn, msg)
}

func (p *Peer) SendFile(filePath string) error {
	p.mu.Lock()
	conn := p.activeConn
	p.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	return SendFile(conn, p.username, filePath)
}

func (p *Peer) Disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeConn != nil {
		SendMessage(p.activeConn, protocol.TCPMessage{
			Type: protocol.TypeDisconnect,
			From: p.username,
		})
		p.activeConn.Close()
		p.activeConn = nil
		p.activeReader = nil
	}

	if p.tcpClient != nil {
		p.tcpClient.Close()
		p.tcpClient = nil
	}

	p.connectedTo = ""
}

func (p *Peer) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeConn != nil
}

func (p *Peer) ConnectedTo() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connectedTo
}

func (p *Peer) Port() int {
	return p.tcpServer.Port()
}

func (p *Peer) Username() string {
	return p.username
}

func (p *Peer) Close() {
	if p.stopHeartbeat != nil {
		close(p.stopHeartbeat)
	}
	p.Disconnect()
	p.tcpServer.Close()
}
