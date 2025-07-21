package protocol

import "net"

// Command byte values as constants
const (
	SERVER_HELLO      byte = 0x01
	SERVER_ACK        byte = 0x02
	PEER_INFO_LOOKUP  byte = 0x03
	PEER_LOOKUP_ACK   byte = 0x04
	PEER_INFO_FORWARD byte = 0x05
	ERROR             byte = 0x06
	BYE               byte = 0x07
)

// ReadMessage reads a message from the connection and returns the operation code and payload.
// It returns an error if the read operation fails.
func ReadMessage(conn net.Conn) (opCode byte, payload []byte, err error) {

	header := make([]byte, 5)
	_, err = conn.Read(header)
	if err != nil {
		return ERROR, nil, err
	}

	command := header[0]
	payloadLength := getPayloadLength(header)

	if payloadLength > 0 {
		payload = make([]byte, payloadLength)
		_, err = conn.Read(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	return command, payload, nil

}

// MakeResponse constructs a response message with the given operation code and payload.
func MakeMessage(opCode byte, payload []byte) []byte {
	payloadLength := len(payload)

	response := make([]byte, 5+payloadLength)
	response[0] = opCode
	response[1] = byte((payloadLength >> 24) & 0xFF)
	response[2] = byte((payloadLength >> 16) & 0xFF)
	response[3] = byte((payloadLength >> 8) & 0xFF)
	response[4] = byte(payloadLength & 0xFF)

	copy(response[5:], payload)

	return response
}

// GetPayloadLength calculates the length of the payload from the header bytes.
func getPayloadLength(payload []byte) int {
	return int(payload[1])<<24 | int(payload[2])<<8 | int(payload[3])<<4 | int(payload[4])
}
