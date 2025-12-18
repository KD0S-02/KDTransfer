package transfer

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/config"
	"github.com/KD0S-02/KDTransfer/internal/crypto"
	"github.com/KD0S-02/KDTransfer/internal/network"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
	"github.com/KD0S-02/KDTransfer/internal/signallingserver"
)

type Client struct {
	Config     *config.Config
	SignalConn net.Conn
	ConnType   network.ConnType
	Transfers  sync.Map
	Key        []byte
}

func NewClient() (*Client, error) {
	cfg := config.LoadConfig()

	conn, err := net.DialTimeout("tcp",
		cfg.SignallingServerHost+":"+cfg.SignallingServerPort, 5*time.Second)
	if err != nil {
		return nil, err
	}

	return &Client{
		SignalConn: conn,
		Config:     cfg,
	}, nil
}

func (c *Client) RegisterWithServer(passphrase string) (string, error) {
	localAddrs, err := network.LocalAddresses(c.Config.TCPPort)
	if err != nil {
		return "", fmt.Errorf("failed to get local addresses: %w", err)
	}

	peerInfo := signallingserver.PeerInfo{
		LocalAddr: localAddrs,
		Type:      signallingserver.PeerTypeNative,
	}

	if len(passphrase) != 0 {
		saltData := crypto.GenerateRandSalt()
		c.Key, err = crypto.GenerateKey(passphrase, saltData)
		if err != nil {
			return "", fmt.Errorf("failed to generate key: %w", err)
		}

		for i, addr := range localAddrs {
			data, err := crypto.EncryptData([]byte(addr), c.Key)
			if err != nil {
				return "", fmt.Errorf("failed to encrypt address: %w", err)
			}
			localAddrs[i] = base64.StdEncoding.EncodeToString(data)
		}
		peerInfo.SaltData = saltData
	}

	payload, err := json.Marshal(peerInfo)
	if err != nil {
		return "", fmt.Errorf("failed to encode peer info: %w", err)
	}

	buf := make([]byte, 8192)

	n, err := protocol.MakeMessage(protocol.ServerHello, payload, buf)
	if err != nil {
		return "", fmt.Errorf("failed to create message: %w", err)
	}

	if _, err := c.SignalConn.Write(buf[:n]); err != nil {
		return "", fmt.Errorf("failed to send registration: %w", err)
	}

	opCode, n, err := protocol.ReadMessage(c.SignalConn, buf)
	if err != nil {
		return "", fmt.Errorf("failed to read server response: %w", err)
	}

	if opCode != protocol.ServerAck {
		if opCode == protocol.Error {
			return "", fmt.Errorf("server error: %s", string(buf[:n]))
		}
		return "", fmt.Errorf("unexpected response: opcode %d", opCode)
	}

	peerID := string(buf[:n])
	log.Printf("Registered with peer ID: %s", peerID)

	return peerID, nil
}

func (c *Client) Transfer(transferID uint32) (*FileTransfer, bool) {
	value, ok := c.Transfers.Load(transferID)

	if !ok {
		return nil, false
	}

	ftr, ok := value.(*FileTransfer)

	return ftr, ok
}

func (c *Client) CompleteTransfer(transferID uint32, message string) {
	ft, ok := c.Transfer(transferID)

	if !ok {
		return
	}

	ft.File.Close()

	c.Transfers.Delete(transferID)

	duration := time.Since(ft.StartTime)

	log.Printf("Transfer %d: %s successfully in %s",
		transferID, message, duration.String())
}

func (c *Client) AddTransfer(transferID uint32, ft *FileTransfer) {
	c.Transfers.Store(transferID, ft)
}

func (c *Client) sendTransferStart(conn net.Conn, transferID uint32, filename string,
	fileSize uint64, numChunks uint32) error {

	buf := make([]byte, protocol.TotalTCPSize)

	n, err := protocol.CreateFileTransferStartPayload(transferID, filename, fileSize, numChunks, buf)
	if err != nil {
		return fmt.Errorf("failed to create start payload: %w", err)
	}

	payload := make([]byte, n)
	copy(payload, buf[:n])

	if len(c.Key) != 0 {
		payload, err = crypto.EncryptData(payload, c.Key)
		if err != nil {
			return fmt.Errorf("failed to encrypt start payload: %w", err)
		}
	}

	msgSize, err := protocol.MakeMessage(protocol.FileTransferStart, payload, buf)
	if err != nil {
		return fmt.Errorf("failed to create start message: %w", err)
	}

	if _, err := conn.Write(buf[:msgSize]); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	return nil
}

func (c *Client) sendTransferEnd(conn net.Conn, transferID uint32, buf []byte) error {
	endPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(endPayload, transferID)

	if len(c.Key) != 0 {
		var err error
		endPayload, err = crypto.EncryptData(endPayload, c.Key)
		if err != nil {
			return fmt.Errorf("failed to encrypt end payload: %w", err)
		}
	}

	n, err := protocol.MakeMessage(protocol.FileTransferEnd, endPayload, buf)
	if err != nil {
		return fmt.Errorf("failed to create end message: %w", err)
	}

	if _, err := conn.Write(buf[:n]); err != nil {
		return fmt.Errorf("failed to send end message: %w", err)
	}

	return nil
}
