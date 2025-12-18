package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Message operation codes for client-server and peer-to-peer communication
const (
	// Signaling server operations
	ServerHello    byte = iota + 1 // Client registration
	ServerAck                      // Registration acknowledgment
	PeerInfoLookup                 // Request peer information
	PeerLookupAck                  // Peer information response
	Bye                            // Clean disconnect
	Error                          // Error message

	// File transfer operations
	FileTransferStart // Initiate file transfer
	FileTransferData  // File data chunk
	FileTransferEnd   // Transfer completion

	// Peer coordination
	PeerInfoForward // Forward peer info to target
	Heartbeat       // Keep-alive ping
	HeartbeatAck    // Keep-alive response
)

// Transport buffer sizes
const (
	// Maximum message sizes for different transports
	TotalTCPSize    = 256 * 1024 // 256KB - optimal for TCP
	TotalWebRTCSize = 16 * 1024  // 16KB - WebRTC SCTP limit

	// Protocol overhead
	TransferHeaderSize = 8 // 4 bytes transfer ID + 4 bytes chunk index

	// Available payload space after headers
	TCPChunkSize    = TotalTCPSize - TransferHeaderSize    // ~256KB
	WebRTCChunkSize = TotalWebRTCSize - TransferHeaderSize // ~16KB
)

type Message struct {
	OpCode     byte
	PayloadLen [4]byte
	Payload    []byte
}

func PingMessage(buf []byte) (int, error) {
	return MakeMessage(Heartbeat, nil, buf)
}

func PongMessage(buf []byte) (int, error) {
	return MakeMessage(HeartbeatAck, nil, buf)
}

func ReadMessage(conn net.Conn, buf []byte) (opCode byte, n int, err error) {
	header := make([]byte, 5)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		return Error, 0, err
	}

	opCode = header[0]
	payloadLen := int(binary.BigEndian.Uint32(header[1:5]))

	if payloadLen > len(buf) {
		return opCode, 0, fmt.Errorf("payload %d exceeds buffer %d", payloadLen, len(buf))
	}

	if payloadLen > 0 {
		_, err = io.ReadFull(conn, buf[:payloadLen])
		if err != nil {
			return opCode, 0, err
		}
	}

	return opCode, payloadLen, nil
}

func MakeMessage(opCode byte, payload []byte, buf []byte) (int, error) {
	payloadLength := len(payload)
	totalSize := 5 + payloadLength

	if len(buf) < totalSize {
		return 0, fmt.Errorf("buffer too small: need %d bytes have %d", totalSize, len(buf))
	}

	buf[0] = opCode
	binary.BigEndian.PutUint32(buf[1:5], uint32(payloadLength))
	copy(buf[5:], payload)

	return totalSize, nil
}

func ParseFileTransferPayload(payload []byte) (transferID uint32,
	fileName string, fileSize uint64, nChunks uint32) {

	// File transfer payload format:
	// [transferID (4 bytes)][fileNameLength (2 bytes)]
	// [fileName (variable length)]
	// [fileSize (8 bytes)][nChunks (4 bytes)]

	// check if the payload is long enough to contain the transfer ID and
	// file name length
	if len(payload) < 6 {
		return 0, "", 0, 0
	}

	transferID = binary.BigEndian.Uint32(payload[:4])
	fileNameLength := binary.BigEndian.Uint16(payload[4:6])

	// check if the payload is long enough to contain the file name, size, and nChunks
	if len(payload) < 6+int(fileNameLength)+8+4 {
		return 0, "", 0, 0
	}

	fileNameStart := 6
	fileNameEnd := fileNameStart + int(fileNameLength)
	fileName = string(payload[fileNameStart:fileNameEnd])

	fileSizeStart := fileNameEnd
	fileSizeEnd := fileNameEnd + 8
	fileSize = uint64(binary.BigEndian.Uint64(
		payload[fileSizeStart:fileSizeEnd]))

	nChunksStart := fileSizeEnd
	nChunks = uint32(binary.BigEndian.Uint32(
		payload[nChunksStart : nChunksStart+4]))

	return transferID, fileName, fileSize, nChunks
}

func ParseFileTransferDataPayload(payload []byte) (transferID uint32,
	chunkIndex uint32, chunkData []byte) {

	// File transfer data payload format:
	// [transferID (4 bytes)][chunkIndex (4 bytes)]
	// [chunkData (256KB if not the last chunk)]

	// check if the payload is long enough to contain the
	//  transfer ID and chunk index
	if len(payload) < 8 {
		return 0, 0, nil
	}

	transferID = binary.BigEndian.Uint32(payload[:4])
	chunkIndex = binary.BigEndian.Uint32(payload[4:8])

	chunkData = payload[8:]

	return transferID, chunkIndex, chunkData
}

func CreateFileTransferStartPayload(transferID uint32,
	fileName string, fileSize uint64, nChunks uint32, buf []byte) (int, error) {
	fileNameBytes := []byte(fileName)
	fileNameLength := len(fileNameBytes)

	// File transfer payload format:
	// [transferID (4 bytes)][fileNameLength (2 bytes)]
	// [fileName (variable length)]
	// [fileSize (8 bytes)][nChunks (4 bytes)]
	totalSize := 6 + fileNameLength + 8 + 4

	if len(buf) < totalSize {
		return 0, fmt.Errorf("buffer too small for file transfer start payload")
	}

	binary.BigEndian.PutUint32(buf[:4], transferID)
	binary.BigEndian.PutUint16(buf[4:6], uint16(fileNameLength))

	copy(buf[6:6+fileNameLength], fileNameBytes)

	fileSizeOffset := 6 + fileNameLength
	nChunksOffset := fileSizeOffset + 8

	binary.BigEndian.PutUint64(buf[fileSizeOffset:nChunksOffset], fileSize)
	binary.BigEndian.PutUint32(buf[nChunksOffset:totalSize], nChunks)

	return totalSize, nil
}

func CreateFileTransferDataRequest(transferID uint32, chunkIndex uint32,
	chunkData []byte, buf []byte) (int, error) {
	totalSize := 13 + len(chunkData)

	if len(buf) < totalSize {
		return 0, fmt.Errorf("buffer too small for file transfer data payload")
	}

	buf[0] = FileTransferData
	binary.BigEndian.PutUint32(buf[1:5], uint32(8+len(chunkData)))
	binary.BigEndian.PutUint32(buf[5:9], transferID)
	binary.BigEndian.PutUint32(buf[9:13], chunkIndex)
	copy(buf[13:], chunkData)

	return totalSize, nil
}
