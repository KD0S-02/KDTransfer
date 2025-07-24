package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

var ftrMap = make(map[uint32]*FileTransfer)

func AddFileTransfer(transferID uint32, filename string, filesize uint64,
	nChunks uint32) {

	ftr := &FileTransfer{
		transferID: transferID,
		filename:   filename,
		filesize:   filesize,
		nChunks:    nChunks,
	}

	ftrMap[transferID] = ftr

}

func RemoveFileTransfer(transferID uint32) {
	delete(ftrMap, transferID)
}

func HandleMessages(peerConn net.Conn) (close bool) {

	opCode, payload, err := protocol.ReadMessage(peerConn)

	if err != nil {
		closeWithError("Failed when reading from direct connection: "+err.Error(), peerConn)
		return
	}

	switch opCode {

	case protocol.FILE_TRANSFER_START:
		transferID, filename, filesize, nChunks := protocol.ParseFileTransferPayload(payload)
		log.Printf("Receiving file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
			filename, transferID, filesize, nChunks)

		ftr := &FileTransfer{
			transferID: transferID,
			filename:   filename,
			filesize:   filesize,
			nChunks:    nChunks,
		}
		ftrMap[transferID] = ftr
		ftrMap[transferID].file, err = os.Create(filename)

		if err != nil {
			log.Println("Error creating file:", err)
			return true
		}

	case protocol.FILE_TRANSFER_DATA:
		transferID, chunkIndex, chunkData := protocol.ParseFileTransferDataPayload(payload)
		log.Printf("Received chunk %d for transfer ID %d", chunkIndex, transferID)

		file := ftrMap[transferID].file

		if file == nil {
			log.Println("file not open for writing:", err)
			return true
		}

		_, err := file.Write(chunkData)
		if err != nil {
			closeWithError("Failed to write chunk to file: "+err.Error(), peerConn)
		}

	case protocol.FILE_TRANSFER_END:
		transferID := binary.BigEndian.Uint32(payload)
		log.Printf("File transfer completed for ID: %d", transferID)
		log.Println("File saved as:", ftrMap[transferID].filename)
		return true
	}

	return false
}
