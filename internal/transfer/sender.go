package transfer

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"

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

func (c *Client) sendFile(transferID uint32, filepath string,
	chunkSize int, peerConn net.Conn) error {

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	i := uint32(0)
	for {
		n, err := file.Read(chunk)
		if n == 0 {
			break
		}
		if err != nil {
			return err
		}
		actualChunk := chunk[:n]
		payload := protocol.CreateFileTransferDataPayload(transferID,
			i, actualChunk)
		request := protocol.MakeMessage(protocol.FILE_TRANSFER_DATA,
			payload)
		_, err = peerConn.Write(request)
		if err != nil {
			return fmt.Errorf("failed to send file chunk: %s", err.Error())
		}
		i++
	}
	return nil
}

func (c *Client) HandleSendCommand(peer string, filepath string) error {

	localAddrs, err := network.GetAllLocalAddresses(c.Config.TCPPort)
	if err != nil {
		return err
	}

	var senderInfo signallingserver.PeerLookUp
	senderInfo.PeerID = peer
	senderInfo.SenderInfo.LocalAddr = localAddrs

	encodedSenderInfo, err := json.Marshal(senderInfo)
	if err != nil {
		return err
	}

	request := protocol.MakeMessage(protocol.PEER_INFO_LOOKUP, encodedSenderInfo)
	c.SignalConn.Write(request)

	opCode, payload, err := protocol.ReadMessage(c.SignalConn)

	if err != nil || opCode == protocol.ERROR {
		return fmt.Errorf("%s", "unexpected response from server on connection")
	}

	var peerInfo signallingserver.PeerInfo

	err = json.Unmarshal(payload, &peerInfo)

	fmt.Printf("Peer Info found for %s\n", peer)

	if err != nil {
		return fmt.Errorf("failed to decode peer address:  %v", err)
	}

	peerConn, connType, err := network.RaceConnections(peerInfo.LocalAddr)

	c.ConnType = connType

	if err != nil {
		return err
	}

	defer peerConn.Close()

	var chunkSize int

	switch c.ConnType {
	case network.TCPConn:
		chunkSize = protocol.TCP_CHUNK_SIZE
	case network.WEBRTCConn:
		chunkSize = protocol.WEBRTC_CHUNK_SIZE
	}

	filename, fileSize, numChunks, err := c.processFile(filepath, chunkSize)

	if err != nil {
		return fmt.Errorf("failed to process file: %s", err.Error())
	}

	transferID := generateTransferID(filename, peerConn.LocalAddr().String())

	payload = protocol.CreateFileTransferStartPayload(transferID,
		filename, fileSize, numChunks)

	request = protocol.MakeMessage(protocol.FILE_TRANSFER_START, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		return fmt.Errorf("failed to send file transfer start message: %s", err.Error())
	}

	log.Printf("Sending file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
		filename, transferID, fileSize, numChunks)

	ft := NewFileTransfer(filename, fileSize, transferID)

	c.AddTransfer(transferID, ft)

	c.sendFile(transferID, filepath, chunkSize, peerConn)

	payload = binary.BigEndian.AppendUint32(nil, transferID)
	request = protocol.MakeMessage(protocol.FILE_TRANSFER_END, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		return fmt.Errorf("failed to send file transfer end message: %s", err.Error())
	}

	c.CompleteTransfer(transferID, "uploaded")

	return nil
}
