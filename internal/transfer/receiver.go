package transfer

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/crypto"
	"github.com/KD0S-02/KDTransfer/internal/network"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
	"github.com/KD0S-02/KDTransfer/internal/signallingserver"
)

func (c *Client) Receiver(passphrase string) error {

	listener, err := net.Listen("tcp", ":"+c.Config.TCPPort)

	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	defer listener.Close()

	localAddrs, err := network.GetAllLocalAddresses(c.Config.TCPPort)

	if err != nil {
		return fmt.Errorf("error while reading network address: %s",
			err.Error())
	}

	peerInfo := signallingserver.PeerInfo{
		LocalAddr: localAddrs,
		Type:      signallingserver.PeerTypeNative,
	}

	if len(passphrase) != 0 {
		saltData := crypto.GenerateRandSalt()
		c.Key, err = crypto.GenerateKey(passphrase, saltData)
		if err != nil {
			return fmt.Errorf("error while generating key: %s",
				err.Error())
		}
		for i, addr := range localAddrs {
			data, err := crypto.EncrpyptData([]byte(addr), c.Key)
			if err != nil {
				return fmt.Errorf("error while encrypting addrs: %s",
					err.Error())
			}
			localAddrs[i] = base64.StdEncoding.EncodeToString(data)
		}
		peerInfo.SaltData = saltData
	}

	payload, err := json.Marshal(peerInfo)

	if err != nil {
		return fmt.Errorf("error while encoding peer info as json: %v",
			err)
	}

	request := protocol.MakeMessage(protocol.SERVER_HELLO, []byte(payload))
	if _, err := c.SignalConn.Write(request); err != nil {
		return fmt.Errorf("failed to send port to server: %w", err)
	}

	_, payload, err = protocol.ReadMessage(c.SignalConn)
	if err != nil {
		return fmt.Errorf("failed to read server response: %w", err)
	}

	log.Println("Current ID:", string(payload))

	go func() {
		if err := listenForDirectConnections(listener, c); err != nil {
			log.Printf("error in peer listener: %v", err)
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

func listenForDirectConnections(listener net.Listener, c *Client) error {
	for {
		peerConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}
		go handlePeerConnection(peerConn, c)
	}
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

	opCode, payload, err := protocol.ReadMessage(peerConn)

	if err != nil {
		return true, err
	}

	switch opCode {

	case protocol.FILE_TRANSFER_START:
		payload, err = c.handleDecrpyption(payload)
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

	case protocol.FILE_TRANSFER_DATA:
		transferID, _, chunkData := protocol.
			ParseFileTransferDataPayload(payload)

		if len(c.Key) != 0 {
			chunkData, err = crypto.DecryptData(chunkData, c.Key)
			if err != nil {
				return true, fmt.Errorf("error while decrypting transfer payload: %s",
					err.Error())
			}
		}

		ft, ok := c.GetTransfer(transferID)
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

	case protocol.FILE_TRANSFER_END:
		payload, err = c.handleDecrpyption(payload)
		if err != nil {
			return true, fmt.Errorf("error while decrypting transfer start payload: %s",
				err.Error())
		}

		transferID := binary.BigEndian.Uint32(payload)
		ft, ok := c.GetTransfer(transferID)

		if !ok {
			return true, fmt.Errorf("invalid ID for FILE_TRANSFER_END message")
		}

		c.CompleteTransfer(ft.TransferID, "received")

		return true, nil
	}

	return false, nil
}
