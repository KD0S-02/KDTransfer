package signallingserver

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/config"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

const (
	bufferSize = 8 * 1024 // 8 KB
)

type SignallingServer struct {
	UserMap     sync.Map
	TCPListener net.Listener
	bufferPool  sync.Pool
}

func NewSignallingServer(cfg *config.Config) (*SignallingServer, error) {
	port := cfg.SignallingServerPort
	address := ":" + port
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return nil, err
	}

	log.Printf("Signalling Server started at addr %s", address)
	return &SignallingServer{
		TCPListener: listener,
		bufferPool: sync.Pool{
			New: func() any {
				return make([]byte, bufferSize)
			},
		},
	}, nil
}

func (ss *SignallingServer) SendToPeer(peer *Peer, opCode byte,
	payload []byte) error {

	buf := ss.GetBuffer()
	n, err := protocol.MakeMessage(opCode, payload, buf)
	if err != nil {
		ss.PutBuffer(buf)
		return err
	}

	message := make([]byte, n)
	copy(message, buf[:n])
	ss.PutBuffer(buf)

	select {
	case peer.Outgoing <- message:
		return nil
	case <-time.After(1 * time.Second):
		ss.RemoveUser(peer.ID)
		return fmt.Errorf("timeout sending message to peer %s",
			peer.ID)
	}
}

func (ss *SignallingServer) GetBuffer() []byte {
	buf := ss.bufferPool.Get().([]byte)
	return buf[:cap(buf)]
}

func (ss *SignallingServer) PutBuffer(buf []byte) {
	if cap(buf) != 8192 {
		return
	}
	ss.bufferPool.Put(buf[:cap(buf)])
}

func (ss *SignallingServer) Start() error {
	for {
		conn, err := ss.TCPListener.Accept()
		if err != nil {
			return err
		}
		go ss.HandleConnection(conn)
	}
}

func (ss *SignallingServer) AddUser(id string, peer *Peer) {
	ss.UserMap.Store(id, peer)
}

func (ss *SignallingServer) RemoveUser(id string) {
	if user, ok := ss.UserMap.LoadAndDelete(id); ok {
		if peer, ok := user.(*Peer); ok {
			peer.CloseOutgoing()
		}
	}
}

func (ss *SignallingServer) GetUser(id string) (*Peer, bool) {
	if user, ok := ss.UserMap.Load(id); ok {
		if peer, ok := user.(*Peer); ok {
			return peer, true
		}
	}
	return nil, false
}
