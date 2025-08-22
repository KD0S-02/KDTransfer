package signallingserver

import (
	"log"
	"net"
	"sync"

	"github.com/KD0S-02/KDTransfer/internal/config"
)

type SignallingServer struct {
	UserMap     sync.Map
	TCPListener net.Listener
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
	}, nil
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

func (ss *SignallingServer) AddUser(id string, peer Peer) {
	ss.UserMap.Store(id, peer)
}

func (ss *SignallingServer) RemoveUser(id string) {
	ss.UserMap.Delete(id)
}

func (ss *SignallingServer) GetUser(id string) (Peer, bool) {
	if user, ok := ss.UserMap.Load(id); ok {
		if peer, ok := user.(Peer); ok {
			return peer, true
		}
	}
	return Peer{}, false
}
