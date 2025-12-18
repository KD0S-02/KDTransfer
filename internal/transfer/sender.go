package transfer

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/crypto"
	"github.com/KD0S-02/KDTransfer/internal/network"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
	"github.com/KD0S-02/KDTransfer/internal/signallingserver"
)

func generateTransferID(filename string, senderIP string) uint32 {
	timeStamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s-%s-%d", filename, senderIP, timeStamp)
	hash := sha256.Sum256([]byte(data))
	return binary.BigEndian.Uint32(hash[:4])
}

func (c *Client) processFile(filepath string,
	chunkSize int) (string, uint64, uint32, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return "", 0, 0, err
	}
	filename := fileInfo.Name()
	filesize := uint64(fileInfo.Size())
	numChunks := uint32(math.Ceil(float64(filesize) / float64(chunkSize)))
	return filename, filesize, numChunks, nil
}

func (c *Client) HandleSendCommand(peer string, filepath string, passphrase string) error {
	receiverInfo, err := c.lookupPeer(peer)
	if err != nil {
		return fmt.Errorf("peer lookup failed: %w", err)
	}

	log.Printf("Found peer %s", peer)

	if len(passphrase) != 0 {
		if err := c.decryptPeerAddresses(&receiverInfo, passphrase); err != nil {
			return fmt.Errorf("failed to decrypt addresses: %w", err)
		}
	}

	peerConn, connType, err := network.RaceConnections(receiverInfo.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}
	defer peerConn.Close()

	c.ConnType = connType
	log.Printf("Connected to peer via %s", connType)

	return c.transferFile(filepath, peerConn)
}
func (c *Client) lookupPeer(peerID string) (signallingserver.PeerInfo, error) {
	lookupRequest := signallingserver.PeerLookUp{PeerID: peerID}
	payload, err := json.Marshal(lookupRequest)
	if err != nil {
		return signallingserver.PeerInfo{}, fmt.Errorf("failed to encode lookup: %w", err)
	}

	buf := make([]byte, 8192)
	n, err := protocol.MakeMessage(protocol.PeerInfoLookup, payload, buf)
	if err != nil {
		return signallingserver.PeerInfo{}, fmt.Errorf("failed to create message: %w", err)
	}

	if _, err := c.SignalConn.Write(buf[:n]); err != nil {
		return signallingserver.PeerInfo{}, fmt.Errorf("failed to send lookup: %w", err)
	}

	opCode, n, err := protocol.ReadMessage(c.SignalConn, buf)
	if err != nil {
		return signallingserver.PeerInfo{}, fmt.Errorf("failed to read response: %w", err)
	}

	if opCode == protocol.Error {
		return signallingserver.PeerInfo{}, fmt.Errorf("server error: %s", string(buf[:n]))
	}

	if opCode != protocol.PeerLookupAck {
		return signallingserver.PeerInfo{}, fmt.Errorf("unexpected response: opcode %d", opCode)
	}

	var peerInfo signallingserver.PeerInfo
	if err := json.Unmarshal(buf[:n], &peerInfo); err != nil {
		return signallingserver.PeerInfo{}, fmt.Errorf("failed to decode peer info: %w", err)
	}

	return peerInfo, nil
}

func (c *Client) decryptPeerAddresses(peerInfo *signallingserver.PeerInfo, passphrase string) error {
	key, err := crypto.GenerateKey(passphrase, peerInfo.SaltData)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	c.Key = key

	for i, addr := range peerInfo.LocalAddr {
		encrypted, err := base64.StdEncoding.DecodeString(addr)
		if err != nil {
			return fmt.Errorf("failed to decode address %d: %w", i, err)
		}

		decrypted, err := crypto.DecryptData(encrypted, c.Key)
		if err != nil {
			return fmt.Errorf("failed to decrypt address %d: %w", i, err)
		}

		peerInfo.LocalAddr[i] = string(decrypted)
	}
	return nil
}

func (c *Client) transferFile(filepath string, peerConn net.Conn) error {
	chunkSize := protocol.TCPChunkSize
	if c.ConnType == network.WEBRTCConn {
		chunkSize = protocol.WebRTCChunkSize
	}

	filename, fileSize, numChunks, err := c.processFile(filepath, chunkSize)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	transferID := generateTransferID(filename, peerConn.LocalAddr().String())

	if err := c.sendTransferStart(peerConn, transferID, filename, fileSize, numChunks); err != nil {
		return err
	}

	log.Printf("Sending file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
		filename, transferID, fileSize, numChunks)

	ft := NewFileTransfer(filename, fileSize, transferID)
	c.AddTransfer(transferID, ft)

	buf := make([]byte, protocol.TotalTCPSize)
	if err := c.sendFile(transferID, filepath, chunkSize, peerConn, buf); err != nil {
		return fmt.Errorf("file transfer failed: %w", err)
	}

	if err := c.sendTransferEnd(peerConn, transferID, buf); err != nil {
		return err
	}

	c.CompleteTransfer(transferID, "sent")

	return nil
}

func (c *Client) sendFile(transferID uint32, filepath string, chunkSize int,
	peerConn net.Conn, buf []byte) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	chunkIndex := uint32(0)

	for {
		n, err := file.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error at chunk %d: %w", chunkIndex, err)
		}
		if n == 0 {
			break
		}

		actualChunk := chunk[:n]

		if len(c.Key) != 0 {
			actualChunk, err = crypto.EncryptData(actualChunk, c.Key)
			if err != nil {
				return fmt.Errorf("failed to encrypt chunk %d: %w", chunkIndex, err)
			}
		}

		msgSize, err := protocol.CreateFileTransferDataRequest(transferID, chunkIndex, actualChunk, buf)
		if err != nil {
			return fmt.Errorf("failed to create chunk %d message: %w", chunkIndex, err)
		}

		if _, err := peerConn.Write(buf[:msgSize]); err != nil {
			return fmt.Errorf("failed to send chunk %d: %w", chunkIndex, err)
		}

		chunkIndex++
	}

	return nil
}
