package protocol

import (
	"encoding/binary"
	"io"
	"net"
)

const (
	SERVER_HELLO        byte = 0x01
	SERVER_ACK          byte = 0x02
	PEER_INFO_LOOKUP    byte = 0x03
	PEER_LOOKUP_ACK     byte = 0x04
	BYE                 byte = 0x05
	ERROR               byte = 0x06
	FILE_TRANSFER_START byte = 0x07
	FILE_TRANSFER_DATA  byte = 0x08
	FILE_TRANSFER_END   byte = 0x09
	PEER_INFO_FORWARD   byte = 0x0A
)

const TCP_CHUNK_SIZE = 512 * 1024   // 512KB
const WEBRTC_CHUNK_SIZE = 16 * 1024 // 16KB

type Message struct {
	OpCode     byte
	PayloadLen [4]byte
	Payload    []byte
}

type TransferPayload struct {
	TranferID   [4]byte
	FilenameLen [2]byte
	Filename    []byte
	Filesize    [8]byte
	NChunks     [4]byte
}

func ReadMessage(conn net.Conn) (opCode byte, payload []byte, err error) {

	header := make([]byte, 5)
	_, err = conn.Read(header)
	if err != nil {
		return ERROR, nil, err
	}

	command := header[0]
	payloadLength := int(binary.BigEndian.Uint32(header[1:5]))

	payload = make([]byte, payloadLength)

	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return 0, nil, err
	}

	return command, payload, nil

}

// Constructs a response message with the given operation code and payload.
func MakeMessage(opCode byte, payload []byte) []byte {
	payloadLength := len(payload)

	response := make([]byte, 5+payloadLength)
	response[0] = opCode
	binary.BigEndian.PutUint32(response[1:5], uint32(payloadLength))
	copy(response[5:], payload)

	return response
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
	fileName string, fileSize uint64, nChunks uint32) []byte {
	fileNameBytes := []byte(fileName)
	fileNameLength := len(fileNameBytes)

	payload := make([]byte, 6+fileNameLength+8+4)
	binary.BigEndian.PutUint32(payload[:4], transferID)
	binary.BigEndian.PutUint16(payload[4:6], uint16(fileNameLength))
	copy(payload[6:6+fileNameLength], fileNameBytes)
	binary.BigEndian.PutUint64(
		payload[6+fileNameLength:6+fileNameLength+8], fileSize)
	binary.BigEndian.PutUint32(payload[6+fileNameLength+8:], nChunks)

	return payload
}

func CreateFileTransferDataRequest(transferID uint32, chunkIndex uint32,
	chunkData []byte) []byte {

	request := make([]byte, 5+8+len(chunkData))
	request[0] = FILE_TRANSFER_DATA
	binary.BigEndian.PutUint32(request[1:5], uint32(8+len(chunkData)))
	binary.BigEndian.PutUint32(request[5:9], transferID)
	binary.BigEndian.PutUint32(request[9:13], chunkIndex)
	copy(request[13:], chunkData)

	return request
}
