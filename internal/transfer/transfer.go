package transfer

import (
	"os"
	"time"
)

type FileTransfer struct {
	File       *os.File
	TransferID uint32
	Filename   string
	Filesize   uint64
	StartTime  time.Time
}

func NewFileTransfer(filename string, filesize uint64, transferID uint32) *FileTransfer {
	return &FileTransfer{
		TransferID: transferID,
		Filename:   filename,
		Filesize:   filesize,
		StartTime:  time.Now(),
	}
}
