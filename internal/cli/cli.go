package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/transfer"
)

type CLI struct {
	Command    string
	Passphrase string
	File       string
	Peer       string
}

func NewCLI() *CLI {
	return &CLI{}
}

func (c *CLI) Parse(args []string) error {

	if len(args) < 2 {
		return fmt.Errorf("usage: kdtransfer <send|recv> [options]")
	}

	c.Command = args[1]
	flags := flag.NewFlagSet(args[1], flag.ContinueOnError)

	flags.StringVar(&c.Passphrase, "passphrase", "",
		"Encryption passphrase to be used in E2EE")

	if c.Command == "send" {
		flags.StringVar(&c.File, "file", "", "Path of file to send")
		flags.StringVar(&c.Peer, "peer", "", "Peer ID")
	}

	if c.Command != "send" && c.Command != "recv" {
		return fmt.Errorf("unknown command: %s", c.Command)
	}

	return flags.Parse(args[2:])
}

func (c *CLI) Run() error {
	err := c.Parse(os.Args)
	if err != nil {
		return err
	}

	client, err := transfer.NewClient()
	if err != nil {
		return err
	}

	switch c.Command {
	case "send":
		if c.File == "" || c.Peer == "" {
			return fmt.Errorf("--file and --peer required")
		}
		return client.HandleSendCommand(c.Peer, c.File, c.Passphrase)
	case "recv":
		return client.Receiver(c.Passphrase)
	default:
		return fmt.Errorf("unknown command: %s", c.Command)
	}
}
