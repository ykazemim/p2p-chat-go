package protocol

type PeerInfo struct {
	Username string `json:"username"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
}

type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type PeersResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type PeerInfoResponse struct {
	Found bool     `json:"found"`
	Peer  PeerInfo `json:"peer,omitempty"`
}

type MessageType string

const (
	TypeConnectionRequest  MessageType = "connection_request"
	TypeConnectionAccepted MessageType = "connection_accepted"
	TypeConnectionRejected MessageType = "connection_rejected"
	TypeChatMessage        MessageType = "chat_message"
	TypeFileTransfer       MessageType = "file_transfer"
	TypeFileChunk          MessageType = "file_chunk"
	TypeFileComplete       MessageType = "file_complete"
	TypeDisconnect         MessageType = "disconnect"
)

type TCPMessage struct {
	Type     MessageType `json:"type"`
	From     string      `json:"from"`
	Content  string      `json:"content,omitempty"`
	Filename string      `json:"filename,omitempty"`
	FileSize int64       `json:"file_size,omitempty"`
	ChunkNum int         `json:"chunk_num,omitempty"`
	Data     []byte      `json:"data,omitempty"`
}
