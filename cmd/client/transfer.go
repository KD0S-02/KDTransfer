package main

import (
	"log"
	"os"
	"sync"
	"time"
)

type FileTransfer struct {
	file       *os.File
	transferID uint32
	filename   string
	filesize   uint64
	startTime  time.Time
}

var Transfers sync.Map

func GetTransfer(transferID uint32) (*FileTransfer, bool) {
	value, ok := Transfers.Load(transferID)

	if !ok {
		return nil, false
	}

	ftr, ok := value.(*FileTransfer)

	return ftr, ok
}

func CompleteTransfer(transferID uint32, message string) {
	ft, ok := GetTransfer(transferID)
	if !ok {
		return
	}
	Transfers.Delete(transferID)

	duration := time.Since(ft.startTime)

	log.Printf("Transfer %d: %s successfully in %v",
		transferID, message, duration.Round(time.Millisecond))
}
