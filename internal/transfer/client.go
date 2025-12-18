package transfer

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/config"
	"github.com/KD0S-02/KDTransfer/internal/network"
)

type Client struct {
	Config     *config.Config
	SignalConn net.Conn
	ConnType   network.ConnType
	Transfers  sync.Map
	Key        []byte
}

func NewClient() (*Client, error) {
	cfg := config.LoadConfig()

	conn, err := net.DialTimeout("tcp",
		cfg.SignallingServerHost+":"+cfg.SignallingServerPort, 5*time.Second)
	if err != nil {
		return nil, err
	}

	return &Client{
		SignalConn: conn,
		Config:     cfg,
	}, nil
}

func (c *Client) Transfer(transferID uint32) (*FileTransfer, bool) {
	value, ok := c.Transfers.Load(transferID)

	if !ok {
		return nil, false
	}

	ftr, ok := value.(*FileTransfer)

	return ftr, ok
}

func (c *Client) CompleteTransfer(transferID uint32, message string) {
	ft, ok := c.Transfer(transferID)

	if !ok {
		return
	}

	ft.File.Close()

	c.Transfers.Delete(transferID)

	duration := time.Since(ft.StartTime)

	log.Printf("Transfer %d: %s successfully in %s",
		transferID, message, duration.String())
}

func (c *Client) AddTransfer(transferID uint32, ft *FileTransfer) {
	c.Transfers.Store(transferID, ft)
}
