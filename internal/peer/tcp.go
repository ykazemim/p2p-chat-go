package peer

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"p2p-chat/internal/protocol"
)

const FileChunkSize = 32 * 1024

type ConnectionHandler func(conn net.Conn, msg protocol.TCPMessage) bool

type TCPServer struct {
	listener   net.Listener
	tlsConfig  *tls.Config
	username   string
	onRequest  ConnectionHandler
	mu         sync.Mutex
	activeConn net.Conn
}

func NewTCPServer(port int, username string, useTLS bool, onRequest ConnectionHandler) (*TCPServer, error) {
	var listener net.Listener
	var tlsConfig *tls.Config
	var err error

	addr := fmt.Sprintf(":%d", port)

	if useTLS {
		tlsConfig, err = GenerateTLSConfig()
		if err != nil {
			return nil, err
		}
		listener, err = tls.Listen("tcp", addr, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	return &TCPServer{
		listener:  listener,
		tlsConfig: tlsConfig,
		username:  username,
		onRequest: onRequest,
	}, nil
}

func (s *TCPServer) Accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		conn.Close()
		return
	}

	var msg protocol.TCPMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		conn.Close()
		return
	}

	if msg.Type == protocol.TypeConnectionRequest {
		if s.onRequest != nil && s.onRequest(conn, msg) {
			s.mu.Lock()
			s.activeConn = conn
			s.mu.Unlock()

			response := protocol.TCPMessage{
				Type: protocol.TypeConnectionAccepted,
				From: s.username,
			}
			SendMessage(conn, response)
		} else {
			response := protocol.TCPMessage{
				Type: protocol.TypeConnectionRejected,
				From: s.username,
			}
			SendMessage(conn, response)
			conn.Close()
		}
	}
}

func (s *TCPServer) GetActiveConn() net.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeConn
}

func (s *TCPServer) Close() error {
	return s.listener.Close()
}

func (s *TCPServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

type TCPClient struct {
	conn      net.Conn
	tlsConfig *tls.Config
	username  string
}

func NewTCPClient(username string, useTLS bool) (*TCPClient, error) {
	var tlsConfig *tls.Config
	var err error

	if useTLS {
		tlsConfig, err = GenerateTLSConfig()
		if err != nil {
			return nil, err
		}
	}

	return &TCPClient{
		tlsConfig: tlsConfig,
		username:  username,
	}, nil
}

func (c *TCPClient) Connect(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	var err error

	if c.tlsConfig != nil {
		c.conn, err = tls.Dial("tcp", addr, c.tlsConfig)
	} else {
		c.conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		return err
	}

	request := protocol.TCPMessage{
		Type: protocol.TypeConnectionRequest,
		From: c.username,
	}

	if err := SendMessage(c.conn, request); err != nil {
		c.conn.Close()
		return err
	}

	reader := bufio.NewReader(c.conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		c.conn.Close()
		return err
	}

	var response protocol.TCPMessage
	if err := json.Unmarshal(line, &response); err != nil {
		c.conn.Close()
		return err
	}

	if response.Type == protocol.TypeConnectionRejected {
		c.conn.Close()
		return fmt.Errorf("connection rejected by %s", response.From)
	}

	return nil
}

func (c *TCPClient) GetConn() net.Conn {
	return c.conn
}

func (c *TCPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func SendMessage(conn net.Conn, msg protocol.TCPMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

func ReceiveMessage(reader *bufio.Reader) (protocol.TCPMessage, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return protocol.TCPMessage{}, err
	}

	var msg protocol.TCPMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return protocol.TCPMessage{}, err
	}

	return msg, nil
}

func SendFile(conn net.Conn, username, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	startMsg := protocol.TCPMessage{
		Type:     protocol.TypeFileTransfer,
		From:     username,
		Filename: filepath.Base(filePath),
		FileSize: stat.Size(),
	}
	if err := SendMessage(conn, startMsg); err != nil {
		return err
	}

	buf := make([]byte, FileChunkSize)
	chunkNum := 0
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		chunkMsg := protocol.TCPMessage{
			Type:     protocol.TypeFileChunk,
			From:     username,
			ChunkNum: chunkNum,
			Data:     buf[:n],
		}
		if err := SendMessage(conn, chunkMsg); err != nil {
			return err
		}
		chunkNum++
	}

	completeMsg := protocol.TCPMessage{
		Type: protocol.TypeFileComplete,
		From: username,
	}
	return SendMessage(conn, completeMsg)
}

func ReceiveFile(reader *bufio.Reader, filename string, fileSize int64, downloadDir string) error {
	filePath := filepath.Join(downloadDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var received int64
	for {
		msg, err := ReceiveMessage(reader)
		if err != nil {
			return err
		}

		if msg.Type == protocol.TypeFileChunk {
			n, err := file.Write(msg.Data)
			if err != nil {
				return err
			}
			received += int64(n)
		} else if msg.Type == protocol.TypeFileComplete {
			break
		}
	}

	return nil
}
