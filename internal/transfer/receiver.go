package transfer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/crypto"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func (c *Client) Receiver(passphrase string) error {

	listener, err := net.Listen("tcp", ":"+c.Config.TCPPort)

	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	defer listener.Close()

	go func() {
		for {
			peerConn, err := listener.Accept()
			if err != nil {
				fmt.Printf("failed to accept connection: %v", err)
			}
			go handlePeerConnection(peerConn, c)
		}
	}()

	return waitForUserInput()
}

func waitForUserInput() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type 'disconnect' to exit")
	for scanner.Scan() {
		if scanner.Text() == "disconnect" {
			return nil
		}
	}
	return scanner.Err()
}

func handlePeerConnection(conn net.Conn, c *Client) error {
	defer conn.Close()

	for {
		shouldClose, err := handleMessages(conn, c)

		if err != nil {
			return err
		}

		if shouldClose {
			return nil
		}
	}
}

func (c *Client) handleDecrpyption(data []byte) ([]byte, error) {
	if len(c.Key) == 0 {
		return data, nil
	}
	return crypto.DecryptData(data, c.Key)
}

func handleMessages(peerConn net.Conn, c *Client) (close bool, err error) {

	buf := make([]byte, protocol.TotalTCPSize)
	opCode, n, err := protocol.ReadMessage(peerConn, buf)
	if err != nil {
		return true, err
	}

	switch opCode {

	case protocol.FileTransferStart:
		payload, err := c.handleDecrpyption(buf[:n])
		if err != nil {
			return true, fmt.Errorf("error while decrypting transfer start payload: %s",
				err.Error())
		}

		transferID, filename, filesize, nChunks :=
			protocol.ParseFileTransferPayload(payload)

		ft := &FileTransfer{
			TransferID: transferID,
			Filename:   filename,
			Filesize:   filesize,
			StartTime:  time.Now(),
		}

		log.Printf("Receiving file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
			filename, transferID, filesize, nChunks)

		file, err := os.Create(filename)
		if err != nil {
			return true, err
		}
		ft.File = file
		c.AddTransfer(transferID, ft)

	case protocol.FileTransferData:
		transferID, _, chunkData := protocol.
			ParseFileTransferDataPayload(buf[:n])

		if len(c.Key) != 0 {
			chunkData, err = crypto.DecryptData(chunkData, c.Key)
			if err != nil {
				return true, fmt.Errorf("error while decrypting transfer payload: %s",
					err.Error())
			}
		}

		ft, ok := c.Transfer(transferID)
		if !ok {
			return true, fmt.Errorf("chunk received for invalid transfer id: %d",
				transferID)
		}

		file := ft.File
		if file == nil {
			return true, fmt.Errorf("file not open for transfer ID: %d", transferID)
		}

		_, err := file.Write(chunkData)
		if err != nil {
			return true, err
		}

	case protocol.FileTransferEnd:
		payload, err := c.handleDecrpyption(buf[:n])
		if err != nil {
			return true, fmt.Errorf("error while decrypting transfer start payload: %s",
				err.Error())
		}

		transferID := binary.BigEndian.Uint32(payload)
		ft, ok := c.Transfer(transferID)

		if !ok {
			return true, fmt.Errorf("invalid ID for FILE_TRANSFER_END message")
		}

		c.CompleteTransfer(ft.TransferID, "received")

		return true, nil
	}

	return false, nil
}
