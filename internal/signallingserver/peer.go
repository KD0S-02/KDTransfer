package signallingserver

import (
	"fmt"
	"sync"
)

type PeerInfo struct {
	SaltData   string
	Type       PeerType
	LocalAddr  []string
	PublicAddr []string
}

type Peer struct {
	ID       string
	PeerInfo PeerInfo
	Outgoing chan []byte
	once     sync.Once
}

func NewPeer(id string, info PeerInfo) *Peer {
	return &Peer{
		ID:       id,
		PeerInfo: info,
		Outgoing: make(chan []byte, 64),
	}
}

func (p *Peer) CloseOutgoing() {
	p.once.Do(func() {
		close(p.Outgoing)
	})
}

func (p *Peer) SendMessage(msg []byte) error {
	select {
	case p.Outgoing <- msg:
		return nil
	default:
		return fmt.Errorf("peer %s buffer full or closed", p.ID)
	}
}

type PeerLookUp struct {
	PeerID     string
	SenderInfo PeerInfo
}

type PeerType string

const (
	PeerTypeNative  PeerType = "native"
	PeerTypeBrowser PeerType = "browser"
)
